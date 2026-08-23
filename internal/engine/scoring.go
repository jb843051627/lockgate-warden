package engine

import (
	"errors"
	"fmt"
	"math"
)

// ErrBadTolerance 容差非法哨兵：调用方须先保证容差为正。
var ErrBadTolerance = errors.New("tolerance must be positive")

// DeltaRatio 计算水位差相对基线的偏移比值：
// ratio = |measured - expected| / tolerance。tolerance<=0 视为配置错误。
func DeltaRatio(measuredM, expectedM, toleranceM float64) (float64, error) {
	if toleranceM <= 0 || math.IsNaN(toleranceM) {
		return 0, ErrBadTolerance
	}
	return math.Abs(measuredM-expectedM) / toleranceM, nil
}

// HeadLevel 水位差维度三档结论。
type HeadLevel string

// normal / suspect / critical。
const (
	HeadNormal   HeadLevel = "normal"
	HeadSuspect  HeadLevel = "suspect"
	HeadCritical HeadLevel = "critical"
)

// HeadVerdict 水位差判定结论。
type HeadOffset struct {
	MeasuredM float64
	ExpectedM float64
	Ratio     float64
	Level     HeadLevel
}

// EvaluateHead 综合比值给出水位差结论：
// ratio>=3 critical；ratio>=2 suspect；其余 normal。
func EvaluateHead(measuredM, expectedM, toleranceM float64) (HeadOffset, error) {
	ratio, err := DeltaRatio(measuredM, expectedM, toleranceM)
	if err != nil {
		return HeadOffset{}, err
	}
	off := HeadOffset{MeasuredM: measuredM, ExpectedM: expectedM, Ratio: ratio}
	switch {
	case ratio >= 3:
		off.Level = HeadCritical
	case ratio >= 2:
		off.Level = HeadSuspect
	default:
		off.Level = HeadNormal
	}
	return off, nil
}

// HeadDetail 格式化水位差结论。
func HeadDetail(o HeadOffset) string {
	return fmt.Sprintf("head %.2fm vs expected %.2fm (ratio %.2f, level %s)",
		o.MeasuredM, o.ExpectedM, o.Ratio, o.Level)
}

// HeadScore 把水位差等级映射为百分制评分。
func HeadScore(level HeadLevel, ratio float64) float64 {
	switch level {
	case HeadCritical:
		return math.Max(5, 40-ratio*5)
	case HeadSuspect:
		return math.Max(40, 75-ratio*10)
	default:
		return 100
	}
}

// PumpScore 泵站维度评分：最近 1 小时排水量越足分越高。
func PumpScore(flowCMS float64, running bool) float64 {
	if !running {
		return 55
	}
	if flowCMS <= 0 {
		return 70
	}
	return math.Min(100, 80+flowCMS*2)
}
