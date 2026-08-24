// Package metrics 维护进程内计数器与仪表，并以文本格式在 /metrics 暴露。
package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// 预定义指标名，避免散落魔法字符串。
const (
	BatchesAccepted   = "lockgate_batches_accepted_total"
	BatchesRejected   = "lockgate_batches_rejected_total"
	PointsInserted    = "lockgate_points_inserted_total"
	PointsDuplicate   = "lockgate_points_duplicate_total"
	AlertsRaised      = "lockgate_alerts_raised_total"
	AlertsClosed      = "lockgate_alerts_closed_total"
	HoldActivations   = "lockgate_hold_activations_total"
	AssessmentsRun    = "lockgate_assessments_run_total"
	IngestQueueDepth  = "lockgate_ingest_queue_depth"
	CachePurgedItems  = "lockgate_cache_purged_items_total"
	WatchdogTicks     = "lockgate_watchdog_ticks_total"
	TransitsCompleted = "lockgate_transits_completed_total"
)

// Metrics 全部指标集中在一个结构里，按名字寻址。
// samples 用 sync.Map 承载，使 ad-hoc 计数器的声明路径在并发下
// 既能 LoadOrStore 去重（不会因重复创建 sample 而丢计数），
// 又避免对普通 map 的并发读写触发 fatal error: concurrent map writes。
type Metrics struct {
	samples sync.Map // map[string]*sample
}

type sample struct {
	counter atomic.Int64
	gauge   atomic.Int64
	isGauge bool
	help    string
}

// New 构造并注册预定义指标。
func New() *Metrics {
	m := &Metrics{}
	m.declare(BatchesAccepted, "telemetry batches accepted")
	m.declare(BatchesRejected, "telemetry batches rejected")
	m.declare(PointsInserted, "telemetry points inserted")
	m.declare(PointsDuplicate, "telemetry points deduplicated by INSERT OR IGNORE")
	m.declare(AlertsRaised, "alerts raised or refreshed")
	m.declare(AlertsClosed, "alerts closed")
	m.declare(HoldActivations, "maintenance holds activated")
	m.declare(AssessmentsRun, "chamber assessments executed")
	m.declareGauge(IngestQueueDepth, "pending ingest post-processing tasks")
	m.declare(CachePurgedItems, "cache entries purged by janitor")
	m.declare(WatchdogTicks, "watchdog scan rounds")
	m.declare(TransitsCompleted, "vessel transits completed")
	return m
}

func (m *Metrics) declare(name, help string) {
	m.samples.Store(name, &sample{help: help})
}

func (m *Metrics) declareGauge(name, help string) {
	m.samples.Store(name, &sample{help: help, isGauge: true})
}

// load 返回指定名字的 sample，缺失时按 factory 现场声明。
// 用 LoadOrStore 保证并发下同一名字只会落到同一个 sample，
// 随后所有累加都走无锁原子路径，既不丢计数也不竞争全局锁。
func (m *Metrics) load(name string, factory func() *sample) *sample {
	if v, ok := m.samples.Load(name); ok {
		return v.(*sample)
	}
	s := factory()
	actual, _ := m.samples.LoadOrStore(name, s)
	return actual.(*sample)
}

// Inc 计数器加一。
func (m *Metrics) Inc(name string) { m.Add(name, 1) }

// Add 计数器累加。
func (m *Metrics) Add(name string, delta int64) {
	s := m.load(name, func() *sample { return &sample{help: "ad-hoc counter"} })
	s.counter.Add(delta)
}

// SetGauge 设置仪表绝对值（如队列深度）。
func (m *Metrics) SetGauge(name string, value int64) {
	s := m.load(name, func() *sample { return &sample{help: "ad-hoc gauge", isGauge: true} })
	s.gauge.Store(value)
}

// Snapshot 返回排序后的名称→值映射，便于测试与导出。
func (m *Metrics) Snapshot() map[string]int64 {
	names := make([]string, 0, 16)
	m.samples.Range(func(key, _ any) bool {
		names = append(names, key.(string))
		return true
	})
	sort.Strings(names)
	out := make(map[string]int64, len(names))
	for _, name := range names {
		s, _ := m.samples.Load(name)
		sm := s.(*sample)
		if sm.isGauge {
			out[name] = sm.gauge.Load()
		} else {
			out[name] = sm.counter.Load()
		}
	}
	return out
}

// Render 输出文本格式指标页（# HELP/# TYPE 行 + 样本行）。
func (m *Metrics) Render() string {
	type row struct {
		name  string
		help  string
		gauge bool
	}
	rows := make([]row, 0, 16)
	m.samples.Range(func(key, value any) bool {
		s := value.(*sample)
		rows = append(rows, row{name: key.(string), help: s.help, gauge: s.isGauge})
		return true
	})
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "# HELP %s %s\n", r.name, r.help)
		kind := "counter"
		if r.gauge {
			kind = "gauge"
		}
		fmt.Fprintf(&b, "# TYPE %s %s\n", r.name, kind)
	}
	for _, r := range rows {
		s, _ := m.samples.Load(r.name)
		sm := s.(*sample)
		var v int64
		if sm.isGauge {
			v = sm.gauge.Load()
		} else {
			v = sm.counter.Load()
		}
		fmt.Fprintf(&b, "%s %d\n", r.name, v)
	}
	return b.String()
}
