package engine

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// ErrInfeasible 过闸不可行哨兵：净空/吃水/水位差任一不满足。
var ErrInfeasible = errors.New("lockage infeasible")

// FillEstimate 闸室调平（充/泄水）时间估算输入。
type FillEstimate struct {
	// HeadDiffM 当前上下游水位差（米）。
	HeadDiffM float64
	// PumpCMS 泵站辅助流量（立方米/秒），重力自流时为 0。
	PumpCMS float64
	// ChamberAreaM2 闸室水面面积。
	ChamberAreaM2 float64
	// SluiceCMS 输水廊道设计流量。
	SluiceCMS float64
}

// Validate 参数合法性：面积与廊道流量必须为正。
func (f FillEstimate) Validate() error {
	if f.ChamberAreaM2 <= 0 {
		return fmt.Errorf("chamber area must be positive")
	}
	if f.SluiceCMS <= 0 {
		return fmt.Errorf("sluice flow must be positive")
	}
	if f.HeadDiffM < 0 {
		return fmt.Errorf("head diff must not be negative")
	}
	return nil
}

// Minutes 调平所需分钟数：
// 有效流量 = 廊道流量 + 泵站流量；水位差越大耗时越长（线性近似）。
func (f FillEstimate) Minutes() (float64, error) {
	if err := f.Validate(); err != nil {
		return 0, err
	}
	effective := f.SluiceCMS + f.PumpCMS
	volume := f.HeadDiffM * f.ChamberAreaM2
	seconds := volume / effective
	return seconds / 60.0, nil
}

// TransitPlan 单船过闸计划（可行性 + 时长估算）。
type TransitPlan struct {
	Vessel      model.Vessel
	Chamber     model.Chamber
	FillMinutes float64
	TotalMin    float64
}

// PlanTransit 评估船舶过闸可行性并估算总时长：
// 进闸航行(按船长/限速) + 调平 + 出闸航行；任一净空不满足返回 ErrInfeasible。
func PlanTransit(v model.Vessel, c model.Chamber, headDiffM, sluiceCMS float64) (*TransitPlan, error) {
	if !v.FitsChamber(&c) {
		return nil, fmt.Errorf("%w: vessel %s does not fit chamber %s", ErrInfeasible, v.Name, c.Code)
	}
	// 吃水校验：闸室槛上最小水深按正常下游水位估算，富裕水深取 0.5m。
	minDepth := v.DraftM + 0.5
	if minDepth > c.NormLevelDownM {
		return nil, fmt.Errorf("%w: draft %.2fm exceeds available depth %.2fm",
			ErrInfeasible, v.DraftM, c.NormLevelDownM)
	}
	fe := FillEstimate{
		HeadDiffM:     math.Min(headDiffM, c.MaxHeadDiffM),
		ChamberAreaM2: c.LengthM * c.WidthM,
		SluiceCMS:     sluiceCMS,
	}
	fill, err := fe.Minutes()
	if err != nil {
		return nil, err
	}
	// 进/出闸航行速度统一取 1.5 m/s。
	sail := (c.LengthM + v.LoraM) / 1.5 / 60.0
	total := sail*2 + fill
	return &TransitPlan{Vessel: v, Chamber: c, FillMinutes: fill, TotalMin: total}, nil
}

// BatchFeasibility 一批船舶的排档可行性汇总。
type BatchFeasibility struct {
	Admitted   []string
	Rejected   []string
	TotalMin   float64
	SeriesTime time.Time
}

// PlanBatch 按优先级依次装载，闸室容量装满后其余拒绝。
// 装载面积利用率上限 85%（安全裕度）。
func PlanBatch(vessels []model.Vessel, c model.Chamber, now time.Time) *BatchFeasibility {
	bf := &BatchFeasibility{SeriesTime: now}
	const capacityRatio = 0.85
	area := c.LengthM * c.WidthM * capacityRatio
	used := 0.0
	for _, v := range vessels {
		need := v.LoraM * v.BeamM
		if used+need > area {
			bf.Rejected = append(bf.Rejected, v.MMSI)
			continue
		}
		used += need
		bf.Admitted = append(bf.Admitted, v.MMSI)
	}
	if len(bf.Admitted) > 0 {
		longest := vessels[0]
		for _, v := range bf.AdmittedList(vessels) {
			if v.LoraM > longest.LoraM {
				longest = v
			}
		}
		_ = longest
	}
	bf.TotalMin = float64(len(bf.Admitted)) * 12.0
	return bf
}

// AdmittedList 按 MMSI 集合过滤出已接纳的船舶对象。
func (bf *BatchFeasibility) AdmittedList(all []model.Vessel) []model.Vessel {
	set := make(map[string]bool, len(bf.Admitted))
	for _, m := range bf.Admitted {
		set[m] = true
	}
	out := make([]model.Vessel, 0, len(set))
	for _, v := range all {
		if set[v.MMSI] {
			out = append(out, v)
		}
	}
	return out
}
