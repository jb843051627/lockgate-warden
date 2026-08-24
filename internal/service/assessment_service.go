package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/engine"
	"github.com/jb843051627/lockgate-warden/internal/metrics"
	"github.com/jb843051627/lockgate-warden/internal/model"
)

// UpsertBaseline 写入水位基线（温度补偿期望差）。
func (s *Service) UpsertBaseline(b *model.LevelBaseline) (model.LevelBaseline, error) {
	if err := b.Validate(); err != nil {
		return model.LevelBaseline{}, err
	}
	sensor, err := s.store.GetSensorByCode(b.SensorCode)
	if err != nil {
		return model.LevelBaseline{}, fmt.Errorf("sensor %s: %s", b.SensorCode, err)
	}
	if sensor.ChamberID != b.ChamberID || sensor.Kind != model.KindLevel {
		return model.LevelBaseline{}, fmt.Errorf("%w: sensor %s is not a level sensor of chamber %d",
			model.ErrOrphanSensor, b.SensorCode, b.ChamberID)
	}
	if err := s.store.UpsertBaseline(b); err != nil {
		return model.LevelBaseline{}, err
	}
	list, err := s.store.ListBaselines(b.ChamberID)
	if err != nil {
		return model.LevelBaseline{}, err
	}
	for _, item := range list {
		if item.SensorCode == b.SensorCode && item.ValidFrom.Equal(b.ValidFrom.UTC()) {
			return item, nil
		}
	}
	return *b, nil
}

// ListBaselines 查询闸室水位基线。
func (s *Service) ListBaselines(chamberID int64) ([]model.LevelBaseline, error) {
	return s.store.ListBaselines(chamberID)
}

// RunAssessment 执行一次闸室安全评估：
// 水位 / 闸门 / 泵站三维度 → 开放等级合成 → 落库并同步闸室状态。
func (s *Service) RunAssessment(chamberID int64) (model.SafetyAssessment, error) {
	chamber, err := s.store.GetChamber(chamberID)
	if err != nil {
		return model.SafetyAssessment{}, err
	}
	now := s.clock.Now()
	frost := s.activeFrost(now)
	dims := []engine.Dimension{}
	notes := []string{frost.Describe()}

	headDim, headScore, headNote := s.assessHead(chamber.ID, now)
	dims = append(dims, headDim)
	if headNote != "" {
		notes = append(notes, headNote)
	}

	gateDim, gateScore, gateNote := s.assessGates(chamber.ID, frost, now)
	dims = append(dims, gateDim)
	if gateNote != "" {
		notes = append(notes, gateNote)
	}

	pumpDim, pumpScore, pumpNote := s.assessPumps(chamber.ID, now)
	dims = append(dims, pumpDim)
	if pumpNote != "" {
		notes = append(notes, pumpNote)
	}

	tally, err := s.store.QualityCountsSince(chamber.ID, now.Add(-24*time.Hour))
	if err != nil {
		return model.SafetyAssessment{}, err
	}

	holdActive := false
	if _, holdErr := s.store.ActiveHoldForChamber(chamber.ID); holdErr == nil {
		holdActive = true
	}

	level := engine.Synthesize(dims, holdActive)
	assessment := &model.SafetyAssessment{
		ChamberID:     chamber.ID,
		HeadScore:     headScore,
		GateScore:     gateScore,
		PumpScore:     pumpScore,
		IntegrityRate: tally.IntegrityRate(),
		Level:         level,
		FrostActive:   frost.Active,
		Notes:         engine.Explain(dims, holdActive, level) + " | " + strings.Join(notes, "; "),
		AssessedAt:    now,
	}
	if err := s.store.InsertAssessment(assessment); err != nil {
		return *assessment, err
	}
	if chamber.Status != level {
		if err := s.store.UpdateChamberStatus(chamber.ID, level, now); err != nil {
			return *assessment, err
		}
		s.cache.PutChamberStatus(chamber.ID, string(level))
	}
	if s.metrics != nil {
		s.metrics.Inc(metrics.AssessmentsRun)
	}
	return *assessment, nil
}

// ListAssessments 查询闸室评估历史。
func (s *Service) ListAssessments(chamberID int64, limit int) ([]model.SafetyAssessment, error) {
	return s.store.ListAssessments(chamberID, limit)
}

func (s *Service) assessHead(chamberID int64, now time.Time) (engine.Dimension, float64, string) {
	sensors, err := s.store.ListSensors(chamberID)
	if err != nil {
		return engine.Dimension{Name: "head", Level: engine.LevelOk}, 100, ""
	}
	worst := engine.HeadOffset{Ratio: 0}
	found := false
	for _, sen := range sensors {
		if sen.Kind != model.KindLevel || !sen.Enabled {
			continue
		}
		hb, hbErr := s.store.LatestHeartbeat(sen.ID)
		if hbErr != nil {
			continue
		}
		expected, tolerance, baseErr := s.baselineFor(sen, now)
		if baseErr != nil {
			continue
		}
		offset, evalErr := engine.EvaluateHead(hb.Value, expected, tolerance)
		if evalErr != nil {
			continue
		}
		found = true
		if offset.Ratio > worst.Ratio {
			worst = offset
		}
	}
	if !found {
		return engine.Dimension{Name: "head", Level: engine.LevelOk}, 100, "no level data"
	}
	dim := engine.Dimension{Name: "head"}
	switch worst.Level {
	case engine.HeadCritical:
		dim.Level = engine.LevelCritical
	case engine.HeadSuspect:
		dim.Level = engine.LevelRestricted
	default:
		dim.Level = engine.LevelOk
	}
	note := fmt.Sprintf("worst head ratio %.2f (%s)", worst.Ratio, worst.Level)
	dim.Detail = note
	return dim, engine.HeadScore(worst.Level, worst.Ratio), note
}

func (s *Service) assessGates(chamberID int64, frost engine.FrostPolicy, now time.Time) (engine.Dimension, float64, string) {
	gates, err := s.store.ListGates(chamberID)
	if err != nil {
		return engine.Dimension{Name: "gate", Level: engine.LevelOk}, 100, ""
	}
	verdicts := []engine.MisalignVerdict{}
	sensors, _ := s.store.ListSensors(chamberID)
	angleByGate := map[int64]float64{}
	for _, sen := range sensors {
		if sen.Kind != model.KindPos || !sen.Enabled || sen.GateRefID <= 0 {
			continue
		}
		hb, hbErr := s.store.LatestHeartbeat(sen.ID)
		if hbErr != nil {
			continue
		}
		if angle, ok := angleByGate[sen.GateRefID]; ok && angle >= hb.Value {
			continue
		}
		angleByGate[sen.GateRefID] = hb.Value
	}
	for _, g := range gates {
		angle, ok := angleByGate[g.ID]
		if !ok {
			continue
		}
		verdict, vErr := engine.EvaluateMisalignment(g.ID, g.Code, angle, frost.MisalignLimit(g.MisalignLimitDeg))
		if vErr == nil {
			verdicts = append(verdicts, verdict)
		}
	}
	worst := engine.WorstMisalignment(verdicts)
	level := engine.LevelOk
	switch {
	case worst >= 1:
		level = engine.LevelCritical
	case worst >= 0.8:
		level = engine.LevelRestricted
	}
	note := fmt.Sprintf("worst misalign ratio %.2f", worst)
	dim := engine.Dimension{Name: "gate", Level: level, Detail: note}
	return dim, engine.GateScore(worst), note
}

func (s *Service) assessPumps(chamberID int64, now time.Time) (engine.Dimension, float64, string) {
	flow, running, err := s.store.WorstPumpFlowSince(chamberID, now.Add(-time.Hour))
	if err != nil || !running {
		return engine.Dimension{Name: "pump", Level: engine.LevelOk}, 100, "pump idle"
	}
	score := engine.PumpScore(flow, running)
	level := engine.LevelOk
	if score < 70 {
		level = engine.LevelRestricted
	}
	note := fmt.Sprintf("pump flow %.1f cms", flow)
	dim := engine.Dimension{Name: "pump", Level: level, Detail: note}
	return dim, score, note
}
