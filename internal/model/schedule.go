package model

import (
	"errors"
	"time"
)

// ScheduleEntry 过闸排档计划条目。
type ScheduleEntry struct {
	ID        int64     `json:"id"`
	ChamberID int64     `json:"chamber_id"`
	VesselID  int64     `json:"vessel_id"`
	SlotStart time.Time `json:"slot_start"`
	SlotEnd   time.Time `json:"slot_end"`
	Priority  int       `json:"priority"`
	Status    string    `json:"status"` // queued / admitted / done / cancelled
}

// PriorityWeight 排档优先级权重：数值越小越优先。
const (
	PriorityHazmat    = 0
	PriorityPassenger = 1
	PriorityCargo     = 2
	PriorityWaiting   = 3
)

// ValidatePlan 校验排档时间槽：起点必须早于终点，且不允许跨天。
func (s *ScheduleEntry) ValidatePlan(now time.Time) error {
	if !s.SlotStart.Before(s.SlotEnd) {
		return errors.New("schedule slot start must precede end")
	}
	if s.SlotStart.Before(now.Add(-24 * time.Hour)) {
		return errors.New("schedule slot is in the far past")
	}
	return nil
}

// PumpEvent 排水泵站事件：启停与流量读数。
type PumpEvent struct {
	ID        int64     `json:"id"`
	ChamberID int64     `json:"chamber_id"`
	PumpCode  string    `json:"pump_code"`
	Action    string    `json:"action"` // start / stop
	FlowCMS   float64   `json:"flow_cms"`
	At        time.Time `json:"at"`
}

// OpsEvent 运维操作留痕（审计）。
type OpsEvent struct {
	ID        int64     `json:"id"`
	ChamberID int64     `json:"chamber_id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	At        time.Time `json:"at"`
}
