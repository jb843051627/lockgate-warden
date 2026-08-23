package service

import (
	"fmt"

	"github.com/jb843051627/lockgate-warden/internal/engine"
	"github.com/jb843051627/lockgate-warden/internal/metrics"
	"github.com/jb843051627/lockgate-warden/internal/model"
)

// RegisterVessel 登记或更新船舶档案。
func (s *Service) RegisterVessel(v *model.Vessel) error {
	if v.MMSI == "" {
		return fmt.Errorf("mmsi is required")
	}
	if err := s.store.UpsertVessel(v); err != nil {
		return fmt.Errorf("vessel %s: %w", v.MMSI, err)
	}
	return nil
}

// GetVessel 查询船舶档案。
func (s *Service) GetVessel(id int64) (model.Vessel, error) {
	v, err := s.store.GetVessel(id)
	if err != nil {
		return v, fmt.Errorf("vessel %d: %w", id, err)
	}
	return v, nil
}

// BeginTransit 船舶进闸：校验净空与方向后登记过闸记录。
// 待闸时长由排档槽起点到进闸时刻估算。
func (s *Service) BeginTransit(mmsi string, chamberID int64, direction string) (model.Transit, error) {
	var out model.Transit
	if !model.TransitDirection[direction] {
		return out, fmt.Errorf("invalid transit direction %q", direction)
	}
	v, err := s.store.GetVesselByMMSI(mmsi)
	if err != nil {
		return out, fmt.Errorf("vessel %s: %w", mmsi, err)
	}
	chamber, err := s.store.GetChamber(chamberID)
	if err != nil {
		return out, fmt.Errorf("chamber %d: %w", chamberID, err)
	}
	if !v.FitsChamber(&chamber) {
		return out, fmt.Errorf("%w: vessel %s exceeds chamber %s clearance",
			model.ErrConflict, v.Name, chamber.Code)
	}
	now := s.clock.Now()
	t := &model.Transit{
		VesselID:  v.ID,
		ChamberID: chamber.ID,
		Direction: direction,
		EnteredAt: now,
		Priority:  engine.PriorityScore(v, 0),
	}
	if err := s.store.InsertTransit(t); err != nil {
		return out, err
	}
	return *t, nil
}

// CompleteTransit 出闸登记：累计待闸时长并计数。
func (s *Service) CompleteTransit(transitID int64) (model.Transit, error) {
	now := s.clock.Now()
	if err := s.store.CompleteTransitRaw(transitID, now); err != nil {
		return model.Transit{}, err
	}
	t, err := s.store.GetTransit(transitID)
	if err != nil {
		return model.Transit{}, err
	}
	waiting := t.ExitedAt.Sub(t.EnteredAt)
	if err := s.store.UpdateTransitWaiting(transitID, int64(waiting.Seconds())); err != nil {
		return t, err
	}
	if s.metrics != nil {
		s.metrics.Inc(metrics.TransitsCompleted)
	}
	t.WaitingSec = int64(waiting.Seconds())
	return t, nil
}
