package engine

import (
	"math"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// ClearanceSample 闸位计累计行程样本（Cumulative 单位：米）。
type ClearanceSample struct {
	At         time.Time
	Cumulative float64
}

// ClearanceRate 计算滑动窗口内的闸门动作速率：
// 返回 (速率 m/h, true)；零跨度或计数器反转视为无有效数据返回 (0, false)。
func ClearanceRate(samples []ClearanceSample, window time.Duration, now time.Time) (float64, bool) {
	if len(samples) == 0 {
		return 0, false
	}
	firstIdx := 0
	for i, s := range samples {
		if !s.At.Before(now.Add(-window)) {
			firstIdx = i
			break
		}
	}
	first := samples[firstIdx]
	last := samples[len(samples)-1]
	if len(samples) < 2 {
		return 0, false
	}
	spanHours := last.At.Sub(first.At).Hours()
	if spanHours <= 0 {
		return 0, false
	}
	delta := last.Cumulative - first.Cumulative
	if delta < 0 {
		return 0, false
	}
	return delta / spanHours, true
}

// PriorityScore 排档优先级评分：数值越小越优先出闸。
// 危险品 > 客船 > 普货；同级按待闸时长加权。
func PriorityScore(v model.Vessel, waited time.Duration) int {
	base := model.PriorityCargo
	switch {
	case v.Hazmat:
		base = model.PriorityHazmat
	case v.Tonnage <= 0:
		base = model.PriorityPassenger
	}
	waitBonus := int(waited.Minutes()) / 30
	if waitBonus > 3 {
		waitBonus = 3
	}
	score := base - waitBonus
	if score < 0 {
		score = 0
	}
	return score
}

// ThroughputScore 日通过量达成率评分。
func ThroughputScore(actual, target int64) float64 {
	if target <= 0 {
		return 100
	}
	ratio := float64(actual) / float64(target)
	if ratio >= 1 {
		return 100
	}
	return math.Max(40, ratio*100)
}
