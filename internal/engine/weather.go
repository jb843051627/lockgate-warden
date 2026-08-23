package engine

import "fmt"

// VisibilityLimit 能见度限航判据。
type VisibilityLimit struct {
	// RestrictedM 低于此能见度限行（米）。
	RestrictedM float64
	// CriticalM 低于此能见度停航（米）。
	CriticalM float64
}

// DefaultVisibilityLimits 内河船闸常用判据：1000m 限行，500m 停航。
func DefaultVisibilityLimits() VisibilityLimit {
	return VisibilityLimit{RestrictedM: 1000, CriticalM: 500}
}

// EvaluateVisibility 能见度判定：低于 critical 停航，低于 restricted 限行。
func EvaluateVisibility(meters float64, lim VisibilityLimit) Verdict {
	v := Verdict{Value: meters}
	switch {
	case meters < lim.CriticalM:
		v.Critical = true
		v.Detail = fmt.Sprintf("visibility %.0fm < critical %.0fm", meters, lim.CriticalM)
	case meters < lim.RestrictedM:
		v.Restricted = true
		v.Detail = fmt.Sprintf("visibility %.0fm < restricted %.0fm", meters, lim.RestrictedM)
	default:
		v.Detail = fmt.Sprintf("visibility %.0fm within limits", meters)
	}
	return v
}

// ThunderstormRisk 雷暴风险指数合成：
// 闪电频次（次/10min）与阵风风速加权，指数 >=60 禁止排档。
func ThunderstormRisk(lightningPer10Min int, gustMS float64) (float64, string) {
	score := float64(lightningPer10Min)*12 + gustMS*2.2
	level := "low"
	switch {
	case score >= 60:
		level = "severe"
	case score >= 35:
		level = "moderate"
	}
	detail := fmt.Sprintf("thunderstorm risk %.0f (%s): lightning %d/10min, gust %.1fm/s",
		score, level, lightningPer10Min, gustMS)
	return score, detail
}

// FogPersistence 雾情持续性判断：连续 low-visibility 样本数达到阈值即认定持续雾情。
// 样本间隔固定 10 分钟；返回持续小时数。
func FogPersistence(lowFlags []bool, thresholdCount int) (float64, bool) {
	if len(lowFlags) == 0 {
		return 0, false
	}
	longest := 0
	cur := 0
	for _, f := range lowFlags {
		if f {
			cur++
			if cur > longest {
				longest = cur
			}
		} else {
			cur = 0
		}
	}
	hours := float64(longest) * 10.0 / 60.0
	return hours, longest >= thresholdCount
}

// FlowLimit 流速限航判据：闸段流速超过上限禁止大型船舶通行。
func FlowLimit(flowCMS float64, maxSafeCMS float64) Verdict {
	v := Verdict{Value: flowCMS}
	if flowCMS > maxSafeCMS*1.3 {
		v.Critical = true
		v.Detail = "navigation suspended"
	} else if flowCMS > maxSafeCMS {
		v.Restricted = true
		v.Detail = "large vessels restricted"
	} else {
		v.Detail = "flow within safe range"
	}
	return v
}
