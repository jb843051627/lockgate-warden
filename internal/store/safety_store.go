package store

import (
	"database/sql"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// UpsertBaseline 写入水位基线（同闸室同传感器保留最新）。
func (s *Store) UpsertBaseline(b *model.LevelBaseline) error {
	res, err := s.db.Exec(`INSERT INTO level_baselines(chamber_id,sensor_code,expected_m,temp_coeff_m,ambient_temp_c,tolerance_m,valid_from)
		VALUES(?,?,?,?,?,?,?)`,
		b.ChamberID, b.SensorCode, b.ExpectedM, b.TempCoeffM, b.AmbientTempC, b.ToleranceM, formatTime(b.ValidFrom))
	if err != nil {
		return err
	}
	b.ID, err = res.LastInsertId()
	return err
}

// ActiveBaselineForSensor 取闸室指定传感器的生效基线（最近一条）。
func (s *Store) ActiveBaselineForSensor(chamberID int64, sensorCode string, now time.Time) (model.LevelBaseline, error) {
	row := s.db.QueryRow(`SELECT id,chamber_id,sensor_code,expected_m,temp_coeff_m,ambient_temp_c,tolerance_m,valid_from
		FROM level_baselines WHERE chamber_id=? AND sensor_code=? AND valid_from<=?
		ORDER BY id DESC LIMIT 1`, chamberID, sensorCode, formatTime(now))
	var (
		b         model.LevelBaseline
		validFrom string
	)
	err := row.Scan(&b.ID, &b.ChamberID, &b.SensorCode, &b.ExpectedM, &b.TempCoeffM,
		&b.AmbientTempC, &b.ToleranceM, &validFrom)
	if err == sql.ErrNoRows {
		return b, notFound
	}
	if err != nil {
		return b, err
	}
	b.ValidFrom = parseTime(validFrom)
	return b, nil
}

// ListBaselines 列出闸室全部基线。
func (s *Store) ListBaselines(chamberID int64) ([]model.LevelBaseline, error) {
	rows, err := s.db.Query(`SELECT id,chamber_id,sensor_code,expected_m,temp_coeff_m,ambient_temp_c,tolerance_m,valid_from
		FROM level_baselines WHERE chamber_id=? ORDER BY id`, chamberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LevelBaseline{}
	for rows.Next() {
		var (
			b         model.LevelBaseline
			validFrom string
		)
		if err := rows.Scan(&b.ID, &b.ChamberID, &b.SensorCode, &b.ExpectedM, &b.TempCoeffM,
			&b.AmbientTempC, &b.ToleranceM, &validFrom); err != nil {
			return nil, err
		}
		b.ValidFrom = parseTime(validFrom)
		out = append(out, b)
	}
	return out, rows.Err()
}

// InsertAssessment 保存评估结论。
func (s *Store) InsertAssessment(a *model.SafetyAssessment) error {
	res, err := s.db.Exec(`INSERT INTO assessments(chamber_id,head_score,gate_score,pump_score,integrity_rate,level,frost_active,notes,assessed_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		a.ChamberID, a.HeadScore, a.GateScore, a.PumpScore, a.IntegrityRate, string(a.Level),
		boolToInt(a.FrostActive), a.Notes, formatTime(a.AssessedAt))
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

// ListAssessments 查询闸室评估历史。
func (s *Store) ListAssessments(chamberID int64, limit int) ([]model.SafetyAssessment, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id,chamber_id,head_score,gate_score,pump_score,integrity_rate,level,frost_active,notes,assessed_at
		FROM assessments WHERE chamber_id=? ORDER BY id DESC LIMIT ?`, chamberID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SafetyAssessment{}
	for rows.Next() {
		var (
			a          model.SafetyAssessment
			level      string
			frost      int64
			assessedAt string
		)
		if err := rows.Scan(&a.ID, &a.ChamberID, &a.HeadScore, &a.GateScore, &a.PumpScore,
			&a.IntegrityRate, &level, &frost, &a.Notes, &assessedAt); err != nil {
			return nil, err
		}
		a.Level = model.LockStatus(level)
		a.FrostActive = frost == 1
		a.AssessedAt = parseTime(assessedAt)
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateHold 新建检修锁（planned）。
func (s *Store) CreateHold(h *model.MaintenanceHold) error {
	res, err := s.db.Exec(`INSERT INTO maintenance_holds(chamber_id,reason,operator,status)
		VALUES(?,?,?, 'planned')`, h.ChamberID, h.Reason, h.Operator)
	if err != nil {
		return err
	}
	h.ID, err = res.LastInsertId()
	h.Status = "planned"
	return err
}

// ActivateHold 激活检修锁。
func (s *Store) ActivateHold(id int64, at time.Time) error {
	res, err := s.db.Exec(`UPDATE maintenance_holds SET status='active', activated_at=? WHERE id=? AND status='planned'`,
		formatTime(at), id)
	if err != nil {
		return err
	}
	return requireAffected(res, model.ErrConflict)
}

// LiftHold 解除检修锁。
func (s *Store) LiftHold(id int64, at time.Time) error {
	res, err := s.db.Exec(`UPDATE maintenance_holds SET status='lifted', lifted_at=? WHERE id=? AND status='active'`,
		formatTime(at), id)
	if err != nil {
		return err
	}
	return requireAffected(res, model.ErrConflict)
}

// GetHold 按 ID 查询检修锁。
func (s *Store) GetHold(id int64) (model.MaintenanceHold, error) {
	row := s.db.QueryRow(`SELECT id,chamber_id,reason,operator,status,activated_at,lifted_at
		FROM maintenance_holds WHERE id=?`, id)
	var (
		h           model.MaintenanceHold
		activatedAt sql.NullString
		liftedAt    sql.NullString
	)
	err := row.Scan(&h.ID, &h.ChamberID, &h.Reason, &h.Operator, &h.Status, &activatedAt, &liftedAt)
	if err == sql.ErrNoRows {
		return h, notFound
	}
	if err != nil {
		return h, err
	}
	if activatedAt.Valid && activatedAt.String != "" {
		t := parseTime(activatedAt.String)
		h.ActivatedAt = &t
	}
	if liftedAt.Valid && liftedAt.String != "" {
		t := parseTime(liftedAt.String)
		h.LiftedAt = &t
	}
	return h, nil
}

// ActiveHoldForLine 取闸室当前激活的检修锁；至多一把。
func (s *Store) ActiveHoldForChamber(chamberID int64) (model.MaintenanceHold, error) {
	row := s.db.QueryRow(`SELECT id FROM maintenance_holds WHERE chamber_id=? AND status='active' ORDER BY id DESC LIMIT 1`, chamberID)
	var id int64
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return model.MaintenanceHold{}, notFound
	}
	if err != nil {
		return model.MaintenanceHold{}, err
	}
	return s.GetHold(id)
}

// InsertPumpEvent 记录泵站启停事件。
func (s *Store) InsertPumpEvent(e *model.PumpEvent) error {
	res, err := s.db.Exec(`INSERT INTO pump_events(chamber_id,pump_code,action,flow_cms,at)
		VALUES(?,?,?,?,?)`, e.ChamberID, e.PumpCode, e.Action, e.FlowCMS, formatTime(e.At))
	if err != nil {
		return err
	}
	e.ID, err = res.LastInsertId()
	return err
}

// WorstPumpFlowSince 统计闸室自 since 以来的最大泵流量与是否运行。
func (s *Store) WorstPumpFlowSince(chamberID int64, since time.Time) (float64, bool, error) {
	var flow float64
	err := s.db.QueryRow(`SELECT COALESCE(MAX(flow_cms),0) FROM pump_events
		WHERE chamber_id=? AND action='start' AND at>=?`, chamberID, formatTime(since)).Scan(&flow)
	if err != nil {
		return 0, false, err
	}
	return flow, flow > 0, nil
}

// InsertOpsEvent 写运维留痕。
func (s *Store) InsertOpsEvent(e *model.OpsEvent) error {
	res, err := s.db.Exec(`INSERT INTO ops_events(chamber_id,actor,action,detail,at)
		VALUES(?,?,?,?,?)`, e.ChamberID, e.Actor, e.Action, e.Detail, formatTime(e.At))
	if err != nil {
		return err
	}
	e.ID, err = res.LastInsertId()
	return err
}

// CountOpsEventsSince 统计闸室自 since 以来的运维操作次数。
func (s *Store) CountOpsEventsSince(chamberID int64, since time.Time) (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM ops_events WHERE chamber_id=? AND at>=?`,
		chamberID, formatTime(since)).Scan(&n)
	return n, err
}
