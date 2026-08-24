package service

import (
	"fmt"
	"sort"
	"strings"
)

// GateFaultCode 闸门故障码（PLC 透传）。
type GateFaultCode string

// 常见故障码集合。
const (
	FaultOvercurrent GateFaultCode = "E01"
	FaultOvertravel  GateFaultCode = "E02"
	FaultSensorLost  GateFaultCode = "E03"
	FaultPowerLow    GateFaultCode = "E04"
)

// FaultEntry 故障登记条目。
type FaultEntry struct {
	GateID  int64
	Code    GateFaultCode
	Detail  string
	AtUnix  int64
	Cleared bool
}

var faultLog = map[int64][]FaultEntry{}

// ReportFault 登记 PLC 故障：同一闸门同故障码未清除前只更新明细。
func (s *Service) ReportFault(gateID int64, code GateFaultCode, detail string) error {
	if gateID <= 0 {
		return fmt.Errorf("gate id is required")
	}
	switch code {
	case FaultOvercurrent, FaultOvertravel, FaultSensorLost, FaultPowerLow:
	default:
		return fmt.Errorf("unknown fault code %q", code)
	}
	for i := range faultLog[gateID] {
		f := &faultLog[gateID][i]
		if f.Code == code && !f.Cleared {
			f.Detail = detail
			f.AtUnix = s.clock.Now().Unix()
			return nil
		}
	}
	faultLog[gateID] = append(faultLog[gateID], FaultEntry{
		GateID: gateID, Code: code, Detail: detail, AtUnix: s.clock.Now().Unix(),
	})
	return nil
}

// ClearFault 清除故障；不存在未清除同码故障返回 not found 语义错误。
func (s *Service) ClearFault(gateID int64, code GateFaultCode) error {
	list := faultLog[gateID]
	for i := range list {
		if list[i].Code == code && !list[i].Cleared {
			list[i].Cleared = true
			return nil
		}
	}
	return fmt.Errorf("fault %s on gate %d: %w", code, gateID, errNotFoundPlain)
}

var errNotFoundPlain = fmt.Errorf("not found")

// OpenFaults 汇总全部未清除故障，按时间倒序。
func (s *Service) OpenFaults() []FaultEntry {
	out := []FaultEntry{}
	for _, list := range faultLog {
		for _, f := range list {
			if !f.Cleared {
				out = append(out, f)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AtUnix > out[j].AtUnix })
	return out
}

// FormatFaultLine 单行故障摘要（前端表格用）。
func FormatFaultLine(f FaultEntry) string {
	return strings.Join([]string{string(f.Code), fmt.Sprintf("gate=%d", f.GateID), f.Detail}, " | ")
}
