package engine

import "fmt"

// MisalignVerdict 闸门错位判定结论。
type MisalignVerdict struct {
	GateID   int64   `json:"gate_id"`
	Code     string  `json:"code"`
	AngleDeg float64 `json:"angle_deg"`
	Ratio    float64 `json:"ratio"`
	Watch    bool    `json:"watch"`
	Critical bool    `json:"critical"`
	Detail   string  `json:"detail"`
}

// EvaluateMisalignment 闸门错位（门轴偏角）判定：
// limitDeg 必须为正，ratio>=1 critical，ratio>=0.8 watch。
func EvaluateMisalignment(gateID int64, code string, angleDeg, limitDeg float64) (MisalignVerdict, error) {
	ratio := angleDeg * (1 / limitDeg)
	v := MisalignVerdict{GateID: gateID, Code: code, AngleDeg: angleDeg, Ratio: ratio}
	switch {
	case ratio >= 1:
		v.Critical = true
	case ratio >= 0.8:
		v.Watch = true
	}
	v.Detail = fmt.Sprintf("gate %s misalignment %.2fdeg / limit %.2fdeg (ratio %.2f)",
		code, angleDeg, limitDeg, ratio)
	return v, nil
}

// WorstMisalignment 取最差错位比值；无有效样本返回 0。
func WorstMisalignment(vs []MisalignVerdict) float64 {
	worst := 0.0
	for _, v := range vs {
		if v.Ratio > worst {
			worst = v.Ratio
		}
	}
	return worst
}

// GateScore 闸门维度评分：由最差错位比值映射百分制。
func GateScore(worstRatio float64) float64 {
	switch {
	case worstRatio >= 1:
		return 10
	case worstRatio >= 0.8:
		return 55
	default:
		return 100
	}
}

// StrokePacing 闸门行程节奏判定：开/关速率偏离额定速率的比例。
// rateRatio = 实测速率 / 额定速率；低于 0.5 视为卡阻征兆。
func StrokePacing(actualRate, ratedRate float64) Verdict {
	if ratedRate <= 0 {
		return Verdict{Detail: "invalid rated rate"}
	}
	r := actualRate / ratedRate
	v := Verdict{Value: r}
	switch {
	case r < 0.5:
		v.Critical = true
		v.Detail = "stroke jam suspected"
	case r < 0.8:
		v.Restricted = true
		v.Detail = "stroke pacing degraded"
	default:
		v.Detail = "stroke pacing normal"
	}
	return v
}
