package model

import (
	"errors"
	"time"
)

// TelemetryPointInput 单个遥测点输入。
type TelemetryPointInput struct {
	SensorCode string    `json:"sensor_code"`
	Seq        int64     `json:"seq"`
	TakenAt    time.Time `json:"taken_at"`
	Value      float64   `json:"value"`
}

// BatchInput 批量入库请求：窗口声明 + 校验和 + 点集。
type BatchInput struct {
	ChamberCode string                `json:"chamber_code"`
	WindowStart time.Time             `json:"window_start"`
	WindowEnd   time.Time             `json:"window_end"`
	Checksum    uint32                `json:"checksum"`
	Points      []TelemetryPointInput `json:"points"`
}

// ValidateBatch 结构校验：非空、窗口存在、点集非空。
func (b *BatchInput) ValidateBatch() error {
	if b.ChamberCode == "" {
		return errors.New("chamber code is required")
	}
	if b.WindowStart.IsZero() || b.WindowEnd.IsZero() {
		return errors.New("telemetry window is required")
	}
	if len(b.Points) == 0 {
		return errors.New("batch must contain points")
	}
	return nil
}

// TelemetryBatch 批次元数据（落库实体）。
type TelemetryBatch struct {
	ID          int64     `json:"id"`
	ChamberID   int64     `json:"chamber_id"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	PointCount  int       `json:"point_count"`
	Checksum    uint32    `json:"checksum"`
	ReceivedAt  time.Time `json:"received_at"`
}

// BatchResult 入库链响应。
type BatchResult struct {
	BatchID     int64             `json:"batch_id"`
	Accepted    bool              `json:"accepted"`
	Inserted    int64             `json:"inserted"`
	Duplicate   int64             `json:"duplicate"`
	QualityHits map[Quality]int64 `json:"quality_hits"`
	AlertsNew   []string          `json:"alerts_new"`
	Notes       []string          `json:"notes"`
}

// StoredPoint 落库后的遥测点（查询视图）。
type StoredPoint struct {
	ID         int64
	BatchID    int64
	SensorID   int64
	SensorCode string
	Seq        int64
	TakenAt    time.Time
	Value      float64
	Quality    Quality
}

// ChecksumResult 校验和重算结果。
type ChecksumResult struct {
	Checksum uint32
	Points   int
}

// QualityTally 三级质量计数器。
type QualityTally struct {
	Good     int64
	Suspect  int64
	Rejected int64
}

// Add 按质量等级累加。
func (t *QualityTally) Add(q Quality) {
	switch q {
	case QualityGood:
		t.Good++
	case QualitySuspect:
		t.Suspect++
	default:
		t.Rejected++
	}
}

// IntegrityRate 优良率：无数据窗口视为全优（返回 1）。
func (t QualityTally) IntegrityRate() float64 {
	total := t.Good + t.Suspect + t.Rejected
	if total == 0 {
		return 1
	}
	return float64(t.Good) / float64(total)
}
