// Package engine 承载全部纯判定逻辑：水位差、闸门行程、风限、错位、
// 开放等级合成、过闸通行率与季节防冻收紧规则。不依赖存储与网络。
package engine

import "fmt"

// windScaleBounds 蒲福风级上边界（m/s），索引即风级，最后一档为 12 级开放上界。
var windScaleBounds = [13]float64{
	0.3, 1.6, 3.4, 5.5, 8.0, 10.8, 13.9, 17.2, 20.8, 24.5, 28.5, 32.7, 1 << 30,
}

// windScaleNames 各风级中文描述。
var windScaleNames = [13]string{
	"无风", "软风", "轻风", "微风", "和风", "清劲风",
	"强风", "疾风", "大风", "烈风", "狂风", "暴风", "飓风",
}

// WindScale 将风速（m/s）映射为蒲福风级 0-12。
func WindScale(windMS float64) int {
	scale := 0
	for i, bound := range windScaleBounds {
		if windMS < bound {
			scale = i
			break
		}
		scale = i
	}
	return scale
}

// WindName 返回风级中文描述；越界返回占位文本。
func WindName(scale int) string {
	if scale < 0 || scale > 12 {
		return "未知"
	}
	return windScaleNames[scale]
}

// HeadThresholds 水位差判据：达到 restricted 差值收紧运行，
// 达到 critical 差值触发停航。
type HeadThresholds struct {
	RestrictedDiffM float64
	CriticalDiffM   float64
}

// DefaultHeadThresholds 客运船闸常用判据：2.5m 水位差限行，4.0m 停航。
func DefaultHeadThresholds() HeadThresholds {
	return HeadThresholds{RestrictedDiffM: 2.5, CriticalDiffM: 4.0}
}

// WindThresholds 风限判据：达到 restrictedScale 收紧，达到 criticalScale 停航。
type WindThresholds struct {
	RestrictedScale int
	CriticalScale   int
}

// DefaultWindThresholds 内河船闸常用判据：8 级大风限航，10 级停航。
func DefaultWindThresholds() WindThresholds {
	return WindThresholds{RestrictedScale: 8, CriticalScale: 10}
}

// Verdict 单维判定通用结论。
type Verdict struct {
	Value      float64 `json:"value"`
	Scale      int     `json:"scale,omitempty"`
	Name       string  `json:"name,omitempty"`
	Restricted bool    `json:"restricted"`
	Critical   bool    `json:"critical"`
	Detail     string  `json:"detail"`
}

// EvaluateHeadLimit 直接按水位差绝对值判定（独立于基线的粗判）。
func EvaluateHeadLimit(headDiffM float64, th HeadThresholds) Verdict {
	v := Verdict{Value: headDiffM}
	switch {
	case headDiffM >= th.CriticalDiffM:
		v.Critical = true
		v.Detail = fmt.Sprintf("head diff %.2fm >= critical %.2fm", headDiffM, th.CriticalDiffM)
	case headDiffM >= th.RestrictedDiffM:
		v.Restricted = true
		v.Detail = fmt.Sprintf("head diff %.2fm >= restricted %.2fm", headDiffM, th.RestrictedDiffM)
	default:
		v.Detail = fmt.Sprintf("head diff %.2fm within limits", headDiffM)
	}
	return v
}

// EvaluateWindLimit 综合风级与阈值给出风限结论。
func EvaluateWindLimit(windMS float64, th WindThresholds) Verdict {
	scale := WindScale(windMS)
	v := Verdict{Value: windMS, Scale: scale, Name: WindName(scale)}
	switch {
	case scale >= th.CriticalScale:
		v.Critical = true
		v.Detail = fmt.Sprintf("wind scale %d (%s) >= critical %d", scale, v.Name, th.CriticalScale)
	case scale >= th.RestrictedScale:
		v.Restricted = true
		v.Detail = fmt.Sprintf("wind scale %d (%s) >= restricted %d", scale, v.Name, th.RestrictedScale)
	default:
		v.Detail = fmt.Sprintf("wind scale %d (%s) within limits", scale, v.Name)
	}
	return v
}
