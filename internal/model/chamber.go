package model

import "time"

// Chamber 闸室档案。
type Chamber struct {
	ID             int64      `json:"id"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	LengthM        float64    `json:"length_m"`
	WidthM         float64    `json:"width_m"`
	NormLevelUpM   float64    `json:"norm_level_up_m"`
	NormLevelDownM float64    `json:"norm_level_down_m"`
	MaxHeadDiffM   float64    `json:"max_head_diff_m"`
	Status         LockStatus `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

// RopeSensor→GateSensor：闸上传感器档案（水位计/闸位计/风速/流速）。
type GateSensor struct {
	ID            int64      `json:"id"`
	ChamberID     int64      `json:"chamber_id"`
	Code          string     `json:"code"`
	Kind          SensorKind `json:"kind"`
	Unit          string     `json:"unit"`
	GateRefID     int64      `json:"gate_ref_id"`
	Enabled       bool       `json:"enabled"`
	ExpectedValue float64    `json:"expected_value"`
	Tolerance     float64    `json:"tolerance"`
	SoftMin       float64    `json:"soft_min"`
	SoftMax       float64    `json:"soft_max"`
	HardMin       float64    `json:"hard_min"`
	HardMax       float64    `json:"hard_max"`
}

// SensorHeartbeat 传感器最新心跳（由入库链异步推进）。
type SensorHeartbeat struct {
	SensorID int64
	Code     string
	Kind     SensorKind
	Value    float64
	Quality  Quality
	SeenAt   time.Time
	BatchID  int64
}

// MaintenanceHold 检修锁：激活期间评估等级强制 maintenance。
type MaintenanceHold struct {
	ID          int64      `json:"id"`
	ChamberID   int64      `json:"chamber_id"`
	Reason      string     `json:"reason"`
	Operator    string     `json:"operator"`
	Status      string     `json:"status"` // planned / active / lifted
	ActivatedAt *time.Time `json:"activated_at"`
	LiftedAt    *time.Time `json:"lifted_at"`
}
