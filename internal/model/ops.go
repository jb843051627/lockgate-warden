package model

import (
	"errors"
	"time"
)

// 领域错误哨兵：上层用 errors.Is 分类处置。
var (
	ErrNotFound       = errors.New("record not found")
	ErrConflict       = errors.New("state conflict")
	ErrAckRequired    = errors.New("acknowledgement required")
	ErrBadWindow      = errors.New("invalid telemetry window")
	ErrFutureWindow   = errors.New("telemetry window ends in the future")
	ErrExpiredWindow  = errors.New("telemetry window is older than retention")
	ErrEmptyBatch     = errors.New("no points left in batch")
	ErrOrphanSensor   = errors.New("sensor does not belong to chamber")
	ErrDisabledSensor = errors.New("sensor is disabled")
	ErrChecksum       = errors.New("checksum mismatch")
)

// Alert 告警实体（去重键维度：chamber|kind|sensor）。
type Alert struct {
	ID           int64         `json:"id"`
	ChamberID    int64         `json:"chamber_id"`
	SensorID     int64         `json:"sensor_id"`
	DedupKey     string        `json:"dedup_key"`
	Kind         string        `json:"kind"`
	Severity     AlertSeverity `json:"severity"`
	Message      string        `json:"message"`
	Status       AlertStatus   `json:"status"`
	Occurrences  int64         `json:"occurrences"`
	FirstSeenAt  time.Time     `json:"first_seen_at"`
	LatestSeenAt time.Time     `json:"latest_seen_at"`
	AckedBy      string        `json:"acked_by"`
	AckedAt      *time.Time    `json:"acked_at"`
	ClosedAt     *time.Time    `json:"closed_at"`
	CloseNote    string        `json:"close_note"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// AlertStatus 告警状态机。
type AlertStatus string

// open → acked → closed；非 critical 允许 open → closed。
const (
	AlertOpen   AlertStatus = "open"
	AlertAcked  AlertStatus = "acked"
	AlertClosed AlertStatus = "closed"
)

// AlertSeverity 告警级别。
type AlertSeverity string

// warning / critical 两级。
const (
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// ParseAlertSeverity 解析告警级别。
func ParseAlertSeverity(raw string) (AlertSeverity, error) {
	switch s := AlertSeverity(raw); s {
	case SeverityWarning, SeverityCritical:
		return s, nil
	default:
		return "", errors.New("unknown alert severity " + raw)
	}
}
