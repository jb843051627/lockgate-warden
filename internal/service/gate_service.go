package service

import (
	"fmt"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/engine"
	"github.com/jb843051627/lockgate-warden/internal/model"
)

// CreateGate 新建闸门档案。
func (s *Service) CreateGate(g *model.Gate) error {
	if err := g.Validate(); err != nil {
		return err
	}
	return s.store.CreateGate(g)
}

// ListGates 列出闸室全部闸门。
func (s *Service) ListGates(chamberID int64) ([]model.Gate, error) {
	return s.store.ListGates(chamberID)
}

// SetGateEnabled 启用/停用闸门；目标不存在时向上返回 notFound。
func (s *Service) SetGateEnabled(id int64, enabled bool) error {
	return s.store.SetGateEnabled(id, enabled)
}

// CommandGate 下发闸门状态机指令：仅接受合法流转，非法流转返回冲突。
func (s *Service) CommandGate(id int64, to model.GateStatus) (model.Gate, error) {
	gate, err := s.store.GetGate(id)
	if err != nil {
		return gate, err
	}
	if !model.IsValidTransition(gate.Status, to) {
		return gate, fmt.Errorf("%w: gate %d cannot move %s -> %s",
			model.ErrConflict, id, gate.Status, to)
	}
	if err := s.store.UpdateGateStatus(id, gate.Status, to, s.clock.Now()); err != nil {
		return gate, err
	}
	return s.store.GetGate(id)
}

// ScheduleSlot 为船舶排定过闸槽位：按危险品/客船/普货赋优先级。
func (s *Service) ScheduleSlot(mmsi string, chamberID int64, start, end time.Time) (model.ScheduleEntry, error) {
	var out model.ScheduleEntry
	v, err := s.store.GetVesselByMMSI(mmsi)
	if err != nil {
		return out, fmt.Errorf("vessel %s: %w", mmsi, err)
	}
	entry := &model.ScheduleEntry{
		ChamberID: chamberID,
		VesselID:  v.ID,
		SlotStart: start,
		SlotEnd:   end,
		Priority:  engine.PriorityScore(v, 0),
		Status:    "queued",
	}
	if err := entry.ValidatePlan(s.clock.Now()); err != nil {
		return out, err
	}
	if err := s.store.CreateScheduleEntry(entry); err != nil {
		return out, err
	}
	return *entry, nil
}

// NextBatch 取闸室下一批待执行排档（按优先级+时间）。
func (s *Service) NextBatch(chamberID int64, limit int) ([]model.ScheduleEntry, error) {
	return s.store.ListQueuedByChamber(chamberID, limit)
}

// FinishSlot 排档出队。
func (s *Service) FinishSlot(id int64) error {
	return s.store.MarkScheduleDone(id, s.clock.Now())
}
