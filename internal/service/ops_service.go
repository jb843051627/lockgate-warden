package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// CreateHold 新建检修锁。
func (s *Service) CreateHold(h *model.MaintenanceHold) error {
	if h.Reason == "" || h.Operator == "" {
		return fmt.Errorf("reason and operator are required")
	}
	return s.store.CreateHold(h)
}

// ActivateHold 激活检修锁并留痕。
func (s *Service) ActivateHold(id int64) (model.MaintenanceHold, error) {
	now := s.clock.Now()
	if err := s.store.ActivateHold(id, now); err != nil {
		return model.MaintenanceHold{}, err
	}
	h, err := s.store.GetHold(id)
	if err != nil {
		return h, err
	}
	s.recordOps(h.ChamberID, h.Operator, "hold.activated", h.Reason)
	return h, nil
}

// LiftHold 解除检修锁并留痕。
func (s *Service) LiftHold(id int64) (model.MaintenanceHold, error) {
	h, err := s.store.GetHold(id)
	if err != nil {
		return h, err
	}
	if err := s.store.LiftHold(id, s.clock.Now()); err != nil {
		return h, err
	}
	s.recordOps(h.ChamberID, h.Operator, "hold.lifted", h.Reason)
	return s.store.GetHold(id)
}

// PumpControl 记录泵站启停事件（供评估维度使用）。
func (s *Service) PumpControl(chamberID int64, pumpCode, action string, flowCMS float64) (model.PumpEvent, error) {
	var out model.PumpEvent
	if action != "start" && action != "stop" {
		return out, fmt.Errorf("invalid pump action %q", action)
	}
	e := &model.PumpEvent{
		ChamberID: chamberID,
		PumpCode:  pumpCode,
		Action:    action,
		FlowCMS:   flowCMS,
		At:        s.clock.Now(),
	}
	if err := s.store.InsertPumpEvent(e); err != nil {
		return out, err
	}
	return *e, nil
}

// ExportChamberCSV 导出闸室近 24h 遥测点 CSV（UTC，RFC4180 简化转义）。
func (s *Service) ExportChamberCSV(chamberID int64, limit int) (string, error) {
	since := s.clock.Now().Add(-24 * time.Hour)
	points, err := s.store.RecentChamberPoints(chamberID, since, limit)
	if err != nil {
		return "", fmt.Errorf("chamber %d export: %w", chamberID, err)
	}
	var b strings.Builder
	b.WriteString("taken_at,sensor_code,seq,value,quality\n")
	for _, p := range points {
		code := strings.ReplaceAll(p.SensorCode, ",", ";")
		fmt.Fprintf(&b, "%s,%s,%d,%.3f,%s\n",
			p.TakenAt.UTC().Format(time.RFC3339), code, p.Seq, p.Value, p.Quality)
	}
	return b.String(), nil
}

// ListChambers 列出全部闸室。
func (s *Service) ListChambers() ([]model.Chamber, error) {
	return s.store.ListChambers()
}

func (s *Service) recordOps(chamberID int64, actor, action, detail string) {
	_ = s.store.InsertOpsEvent(&model.OpsEvent{
		ChamberID: chamberID,
		Actor:     actor,
		Action:    action,
		Detail:    detail,
		At:        s.clock.Now(),
	})
}
