package service

import (
	"fmt"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/engine"
)

// StatsHourly 单小时负载视图（透传 store 统计）。
type StatsHourly = struct {
	Hour    string
	Batches int64
	Points  int64
	Rejects int64
}

// HourlyLoads 查询闸室逐小时入库负载。
func (s *Service) HourlyLoads(chamberID int64, hours int) ([]StatsHourly, error) {
	if hours <= 0 || hours > 168 {
		return nil, fmt.Errorf("hours must be in [1,168]")
	}
	rows, err := s.store.HourlyLoads(chamberID, s.clock.Now().Add(-time.Duration(hours)*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("hourly loads: %w", err)
	}
	out := make([]StatsHourly, 0, len(rows))
	for _, r := range rows {
		out = append(out, StatsHourly{Hour: r.Hour, Batches: r.Batches, Points: r.Points, Rejects: r.Rejects})
	}
	return out, nil
}

// DwellReport 待闸时长分位报告。
type DwellReport struct {
	P50Min float64 `json:"p50_min"`
	P95Min float64 `json:"p95_min"`
}

// DwellReport 查询近 N 天待闸分位。
func (s *Service) DwellReport(chamberID int64, days int) (DwellReport, error) {
	if days <= 0 || days > 30 {
		return DwellReport{}, fmt.Errorf("days must be in [1,30]")
	}
	p50, p95, err := s.store.DwellStats(chamberID, s.clock.Now().AddDate(0, 0, -days))
	if err != nil {
		return DwellReport{}, err
	}
	return DwellReport{P50Min: p50, P95Min: p95}, nil
}

// PlanLockage 过闸计划：可行性校验 + 调平时长估算。
func (s *Service) PlanLockage(mmsi string, chamberID int64) (*engine.TransitPlan, error) {
	v, err := s.store.GetVesselByMMSI(mmsi)
	if err != nil {
		return nil, fmt.Errorf("vessel %s: %w", mmsi, err)
	}
	c, err := s.store.GetChamber(chamberID)
	if err != nil {
		return nil, fmt.Errorf("chamber %d: %w", chamberID, err)
	}
	head := c.MaxHeadDiffM
	plan, err := engine.PlanTransit(v, c, head, 6.0)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// WeatherGate 气象限航判定汇总：能见度 + 雷暴 + 流速三重门槛。
func (s *Service) WeatherGate(visibilityM float64, lightning int, gustMS, flowCMS float64) (bool, string) {
	lim := engine.DefaultVisibilityLimits()
	vv := engine.EvaluateVisibility(visibilityM, lim)
	risk, riskDetail := engine.ThunderstormRisk(lightning, gustMS)
	flow := engine.FlowLimit(flowCMS, 3.5)
	deny := vv.Critical || risk >= 60 || flow.Critical
	detail := vv.Detail + "; " + riskDetail + "; " + flow.Detail
	return deny, detail
}

// SensorUptimeReport 传感器在线率报告。
func (s *Service) SensorUptimeReport(sensorID int64, days int) (float64, error) {
	if days <= 0 || days > 30 {
		return 0, fmt.Errorf("days must be in [1,30]")
	}
	ratio, err := s.store.SensorUptime(sensorID, s.clock.Now().AddDate(0, 0, -days))
	if err != nil {
		return 0, err
	}
	return ratio, nil
}

// GatePacingReport 闸门动作速率报告：取近 24h 闸位计样本计算速率，
// 样本不足时由引擎层判定为无有效数据。
func (s *Service) GatePacingReport(sensorID int64) (float64, bool, error) {
	pts, err := s.store.RecentSensorPoints(sensorID, s.clock.Now().Add(-24*time.Hour), 500)
	if err != nil {
		return 0, false, err
	}
	if len(pts) < 2 {
		return 0, false, nil
	}
	samples := make([]engine.ClearanceSample, 0, len(pts))
	for _, p := range pts {
		samples = append(samples, engine.ClearanceSample{At: p.TakenAt, Cumulative: p.Value})
	}
	rate, ok := engine.ClearanceRate(samples, 24*time.Hour, s.clock.Now())
	return rate, ok, nil
}
