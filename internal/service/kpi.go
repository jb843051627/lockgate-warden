package service

import (
	"fmt"

	"github.com/jb843051627/lockgate-warden/internal/store"
)

// closeRatio 告警关闭率：周期内无新增告警视为全部关闭（返回 1）。
func closeRatio(closed, opened int64) float64 {
	if opened <= 0 {
		return 1
	}
	return float64(closed) / float64(opened)
}

// WeeklyKPI 周报聚合视图。
type WeeklyKPI struct {
	ChamberID     int64               `json:"chamber_id"`
	Days          int                 `json:"days"`
	Transits      int64               `json:"transits"`
	AvgWaitingMin float64             `json:"avg_waiting_min"`
	CloseRate     float64             `json:"close_rate"`
	IntegrityRate float64             `json:"integrity_rate"`
	Buckets       []store.DailyBucket `json:"buckets"`
}

// LineWeeklyKPI 汇总闸室近 N 天运行 KPI。
func (s *Service) LineWeeklyKPI(chamberID int64, days int) (WeeklyKPI, error) {
	if days <= 0 || days > 30 {
		return WeeklyKPI{}, fmt.Errorf("days must be in [1,30]")
	}
	chamber, err := s.store.GetChamber(chamberID)
	if err != nil {
		return WeeklyKPI{}, err
	}

	buckets, err := s.store.WeeklyBuckets(chamber.ID, days, s.clock.Now())
	if err != nil {
		return WeeklyKPI{}, err
	}

	var totalWaiting float64
	for _, b := range buckets {
		totalWaiting += b.AvgWaitingMin
	}

	closedCount, openedCount, err := s.store.AlertCloseCounts(chamber.ID, s.clock.Now().AddDate(0, 0, -days))
	if err != nil {
		return WeeklyKPI{}, err
	}
	transits, err := s.store.CountTransitsSince(chamber.ID, s.clock.Now().AddDate(0, 0, -days))
	if err != nil {
		return WeeklyKPI{}, err
	}

	kpi := WeeklyKPI{
		ChamberID:     chamber.ID,
		Days:          days,
		Buckets:       buckets,
		Transits:      transits,
		CloseRate:     closeRatio(closedCount, openedCount),
		IntegrityRate: aggregateIntegrity(buckets),
	}
	if len(buckets) > 0 {
		kpi.AvgWaitingMin = totalWaiting / float64(len(buckets))
	}
	return kpi, nil
}

func aggregateIntegrity(buckets []store.DailyBucket) float64 {
	var sum float64
	for _, b := range buckets {
		sum += b.IntegrityRate
	}
	if len(buckets) == 0 {
		return 1
	}
	return sum / float64(len(buckets))
}
