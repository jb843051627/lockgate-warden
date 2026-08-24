package engine

import (
	"fmt"
	"time"
)

// FrostPolicy 季节防冻收紧策略：
// 冬季（或注入时钟落在低温月份）时降低 restricted 判据并给错位裕度留余量。
type FrostPolicy struct {
	Active bool `json:"active"`

	// WinterMonths 防冻关注期月份集合（1-12）。
	WinterMonths []int `json:"winter_months"`

	// BaseRestrictedDiff 正常季 restricted 判据对应的水位差下限（米）。
	BaseRestrictedDiff float64 `json:"base_restricted_diff"`

	// FrostRelief 防冻关注期把 restricted 水位差下调的幅度（米）。
	FrostRelief float64 `json:"frost_relief"`

	// MarginDeg 防冻期附加的闸门错位裕度（度），等效收紧错位判据。
	MarginDeg float64 `json:"margin_deg"`
}

// DefaultFrostPolicy 默认策略：11 月至次年 3 月为防冻关注期，
// restricted 水位差由 2.5m 放宽触发面至 2.0m，错位裕度 0.3 度。
func DefaultFrostPolicy() FrostPolicy {
	return FrostPolicy{
		WinterMonths:       []int{11, 12, 1, 2, 3},
		BaseRestrictedDiff: 2.5,
		FrostRelief:        0.5,
		MarginDeg:          0.3,
	}
}

// inWinter 判断月份是否属于防冻关注期。
func (p FrostPolicy) inWinter(month time.Month) bool {
	for _, m := range p.WinterMonths {
		if int(month) == m {
			return true
		}
	}
	return false
}

// ResolveForTime 依据注入时钟所在月份决定策略是否生效。
func (p FrostPolicy) ResolveForTime(now time.Time) FrostPolicy {
	p.Active = p.inWinter(now.Month())
	return p
}

// BaseHeadThresholds 未叠加防冻收紧的基础水位差判据。
func (p FrostPolicy) BaseHeadThresholds() HeadThresholds {
	return HeadThresholds{RestrictedDiffM: p.BaseRestrictedDiff, CriticalDiffM: 4.0}
}

// HeadThresholds 输出当前生效的水位差判据；
// 防冻期生效时 restricted 差值按 relief 下调，但不得达到 critical。
func (p FrostPolicy) HeadThresholds() HeadThresholds {
	th := HeadThresholds{RestrictedDiffM: p.BaseRestrictedDiff, CriticalDiffM: 4.0}
	if p.Active {
		th = applyFrostRelief(th, p.FrostRelief)
	}
	return th
}

// MisalignLimit 输出叠加防冻裕度后的闸门错位上限；
// 裕度从限值中扣除，等效提前进入受限区。非法基线直接钳 0 交给引擎层报错。
func (p FrostPolicy) MisalignLimit(baseLimitDeg float64) float64 {
	if baseLimitDeg <= 0 {
		return 0
	}
	if !p.Active {
		return baseLimitDeg
	}
	tightened := baseLimitDeg + p.MarginDeg
	if tightened < 0.5 {
		tightened = 0.5
	}
	return tightened
}

// Describe 输出策略摘要用于评估备注。
func (p FrostPolicy) Describe() string {
	if !p.Active {
		return "frost policy idle"
	}
	return fmt.Sprintf("frost policy active: head restricted<=%.1fm, misalign margin %.1fdeg",
		p.BaseRestrictedDiff-p.FrostRelief, p.MarginDeg)
}
