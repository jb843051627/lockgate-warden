package model

import (
	"fmt"
	"time"
)

// LevelBaseline 水位基线：温度补偿的期望水位差 + 容差。
// 金属结构随气温伸缩，期望差按 20°C 基准温度修正。
type LevelBaseline struct {
	ID           int64     `json:"id"`
	ChamberID    int64     `json:"chamber_id"`
	SensorCode   string    `json:"sensor_code"`
	ExpectedM    float64   `json:"expected_m"`
	TempCoeffM   float64   `json:"temp_coeff_m"`
	AmbientTempC float64   `json:"ambient_temp_c"`
	ToleranceM   float64   `json:"tolerance_m"`
	ValidFrom    time.Time `json:"valid_from"`
}

// Validate 基线合法性：容差必须为正，系数非负。
func (b *LevelBaseline) Validate() error {
	if b.ToleranceM <= 0 {
		return fmt.Errorf("tolerance must be positive")
	}
	if b.TempCoeffM < 0 {
		return fmt.Errorf("temp coefficient must not be negative")
	}
	if b.SensorCode == "" {
		return fmt.Errorf("sensor code is required")
	}
	return nil
}

// EffectiveExpected 计算当前气温下的期望水位差：
// 以 20°C 为基准点，偏离基准按线性系数修正（高温水体膨胀，期望差增大）。
func (b *LevelBaseline) EffectiveExpected() float64 {
	return b.ExpectedM + b.TempCoeffM*(b.AmbientTempC-20.0)
}

// SafetyAssessment 闸室安全评估结论。
type SafetyAssessment struct {
	ID            int64      `json:"id"`
	ChamberID     int64      `json:"chamber_id"`
	HeadScore     float64    `json:"head_score"`
	GateScore     float64    `json:"gate_score"`
	PumpScore     float64    `json:"pump_score"`
	IntegrityRate float64    `json:"integrity_rate"`
	Level         LockStatus `json:"level"`
	FrostActive   bool       `json:"frost_active"`
	Notes         string     `json:"notes"`
	AssessedAt    time.Time  `json:"assessed_at"`
}
