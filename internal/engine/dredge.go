package engine

import (
	"fmt"
	"time"
)

// SiltSample 淤积测厚样本（闸室底板）。
type SiltSample struct {
	At        time.Time
	Thickness float64
}

// SiltRate 估算年淤积速率（米/年）：取首末样本线性拟合。
// 样本不足或时间跨度为 0 返回 (0,false)。
func SiltRate(samples []SiltSample) (float64, bool) {
	if len(samples) < 2 {
		return 0, false
	}
	first := samples[0]
	last := samples[len(samples)-1]
	spanYears := last.At.Sub(first.At).Hours() / 24.0 / 365.0
	if spanYears <= 0 {
		return 0, false
	}
	delta := last.Thickness - first.Thickness
	return delta / spanYears, true
}

// DredgeDue 清淤到期判定：当前厚度超过阈值即需清淤，
// 或按速率外推将在 horizonDays 内超限。
func DredgeDue(currentM, limitM, annualRateM float64, horizonDays int) (bool, string) {
	if currentM >= limitM {
		return true, fmt.Sprintf("thickness %.2fm exceeds limit %.2fm", currentM, limitM)
	}
	if annualRateM <= 0 || horizonDays <= 0 {
		return false, "no growth trend"
	}
	projected := currentM + annualRateM*(float64(horizonDays)/365.0)
	if projected >= limitM {
		return true, fmt.Sprintf("projected %.2fm reaches limit within %dd", projected, horizonDays)
	}
	return false, fmt.Sprintf("ok: %.2fm/%.2fm", currentM, limitM)
}
