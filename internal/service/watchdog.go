package service

import (
	"log"

	"github.com/jb843051627/lockgate-warden/internal/metrics"
)

// RunWatchdog 巡检一轮：自动关闭过期 warning + 心跳过期扫描。
// 单步失败只记日志不中断下一轮。
func (s *Service) RunWatchdog() {
	now := s.clock.Now()

	closed, err := s.store.AutoCloseStaleWarnings(now.Add(s.params.WarnTTL), now)
	if err != nil {
		log.Printf("watchdog auto-close failed: %v", err)
	} else if closed > 0 {
		if s.metrics != nil {
			s.metrics.Add(metrics.AlertsClosed, closed)
		}
		log.Printf("watchdog auto-closed %d stale warnings", closed)
	}

	stale, err := s.store.ListStaleSensors(now.Add(-s.params.Staleness))
	if err != nil {
		log.Printf("watchdog stale scan failed: %v", err)
		return
	}
	for _, sen := range stale {
		created, err := s.raiseAlert(sen.ChamberID, sen.ID, "sensor_stale", "warning",
			"sensor heartbeat exceeded staleness window")
		if err != nil {
			log.Printf("watchdog raise stale alert failed sensor=%s: %v", sen.Code, err)
			continue
		}
		_ = created
	}
	if s.metrics != nil {
		s.metrics.Inc(metrics.WatchdogTicks)
	}
}
