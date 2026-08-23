package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// RegistrySummary 船舶档案登记摘要。
type RegistrySummary struct {
	Vessels int
	Hazmat  int
}

// UpsertVesselEntry 登记船舶（供 API 层调用，含字段清洗）。
func (s *Service) UpsertVesselEntry(v *model.Vessel) error {
	v.MMSI = strings.TrimSpace(v.MMSI)
	v.Name = strings.TrimSpace(v.Name)
	if len(v.MMSI) != 9 {
		return fmt.Errorf("mmsi must be 9 digits")
	}
	if v.LoraM <= 0 || v.BeamM <= 0 || v.DraftM <= 0 {
		return fmt.Errorf("vessel dimensions must be positive")
	}
	return s.store.UpsertVessel(v)
}

// VesselFits 判断船舶是否满足闸室净空。
func (s *Service) VesselFits(mmsi string, chamberID int64) (bool, error) {
	v, err := s.store.GetVesselByMMSI(mmsi)
	if err != nil {
		return false, fmt.Errorf("vessel %s: %w", mmsi, err)
	}
	c, err := s.store.GetChamber(chamberID)
	if err != nil {
		return false, fmt.Errorf("chamber %d: %w", chamberID, err)
	}
	return v.FitsChamber(&c), nil
}

// WaitingEstimate 估算排档等待时长：槽位起点距当前时刻的秒数。
func (s *Service) WaitingEstimate(entry model.ScheduleEntry, now time.Time) int64 {
	wait := entry.SlotStart.Sub(now).Seconds()
	if wait < 0 {
		return 0
	}
	return int64(wait)
}

// FormatSlot 渲染排档时间槽（UTC 文本）。
func FormatSlot(start, end time.Time) string {
	return fmt.Sprintf("%s~%s",
		start.UTC().Format("1504"), end.UTC().Format("1504"))
}
