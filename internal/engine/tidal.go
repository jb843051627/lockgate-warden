package engine

import (
	"fmt"
	"math"
	"time"
)

// TideSample 潮位观测样本（感潮河段闸下水位）。
type TideSample struct {
	At    time.Time
	Level float64
}

// HarmonicConst 分潮调和常数。
type HarmonicConst struct {
	// AmplitudeM 振幅（米）。
	AmplitudeM float64
	// PhaseRad 相位（弧度）。
	PhaseRad float64
	// SpeedDegH 角速度（度/小时）：M2≈28.98, S2≈30.0, K1≈15.04。
	SpeedDegH float64
}

// DefaultHarmonics 半日潮为主的三分潮近似（M2/S2/K1）。
func DefaultHarmonics() []HarmonicConst {
	return []HarmonicConst{
		{AmplitudeM: 1.2, PhaseRad: 0.4, SpeedDegH: 28.984},
		{AmplitudeM: 0.35, PhaseRad: 1.1, SpeedDegH: 30.000},
		{AmplitudeM: 0.25, PhaseRad: 2.2, SpeedDegH: 15.041},
	}
}

// PredictTide 调和法潮位预报：以 epoch 为相位基准。
func PredictTide(epoch, at time.Time, meanLevel float64, harmonics []HarmonicConst) float64 {
	hours := at.Sub(epoch).Hours()
	level := meanLevel
	for _, h := range harmonics {
		rad := h.PhaseRad + (h.SpeedDegH*hours/180.0)*math.Pi
		level += h.AmplitudeM * math.Cos(rad)
	}
	return level
}

// NextLowWater 预报下一次低潮时刻：从 from 开始按 10 分钟步进搜索极小值，
// 最长搜索 26 小时（覆盖一个完整潮周期）。找不到返回零值 false。
func NextLowWater(from time.Time, meanLevel float64, harmonics []HarmonicConst) (time.Time, float64, bool) {
	const (
		step    = 10 * time.Minute
		maxSpan = 26 * time.Hour
	)
	var prevLevel float64
	prevAt := from
	prevLevel = PredictTide(from, from, meanLevel, harmonics)
	for t := from.Add(step); t.Sub(from) <= maxSpan; t = t.Add(step) {
		lv := PredictTide(from, t, meanLevel, harmonics)
		if lv > prevLevel {
			// prev 即极小点。
			return prevAt, prevLevel, true
		}
		prevAt, prevLevel = t, lv
	}
	return time.Time{}, 0, false
}

// SlackWindow 平潮窗口：低潮前后各 minutes 分钟内适合大型船舶过闸。
func SlackWindow(lowAt time.Time, minutes int, now time.Time) (bool, string) {
	start := lowAt.Add(-time.Duration(minutes) * time.Minute)
	end := lowAt.Add(time.Duration(minutes) * time.Minute)
	if now.Before(start) || now.After(end) {
		return false, fmt.Sprintf("slack window %s~%s",
			start.Format("15:04"), end.Format("15:04"))
	}
	return true, "within slack window"
}

// SurgeRisk 增水风险：实测潮位 - 预报潮位的偏差超过阈值即存在风暴增水。
func SurgeRisk(observed, predicted, thresholdM float64) Verdict {
	v := Verdict{Value: observed - predicted}
	switch {
	case v.Value >= thresholdM*2:
		v.Critical = true
		v.Detail = fmt.Sprintf("surge %.2fm >= severe %.2fm", v.Value, thresholdM*2)
	case v.Value >= thresholdM:
		v.Restricted = true
		v.Detail = fmt.Sprintf("surge %.2fm >= warning %.2fm", v.Value, thresholdM)
	default:
		v.Detail = fmt.Sprintf("surge %.2fm normal", v.Value)
	}
	return v
}
