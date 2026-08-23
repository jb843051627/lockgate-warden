// Package service 编排全部业务用例：入库链、告警状态机、评估合成、
// 过闸调度、周报与导出。依赖仅限 store/engine/cache/ingest 抽象。
package service

import (
	"time"

	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/engine"
	"github.com/jb843051627/lockgate-warden/internal/ingest"
	"github.com/jb843051627/lockgate-warden/internal/metrics"
	"github.com/jb843051627/lockgate-warden/internal/store"
	"github.com/jb843051627/lockgate-warden/internal/validation"
)

// Params 业务参数集合（去重窗口、心跳过期、自动关闭 TTL）。
type Params struct {
	DedupWindow time.Duration
	Staleness   time.Duration
	WarnTTL     time.Duration
}

func (p Params) withDefaults() Params {
	if p.DedupWindow <= 0 {
		p.DedupWindow = 30 * time.Minute
	}
	if p.Staleness <= 0 {
		p.Staleness = 10 * time.Minute
	}
	if p.WarnTTL <= 0 {
		p.WarnTTL = 6 * time.Hour
	}
	return p
}

// Service 业务门面：聚合全部用例所需的协作者。
type Service struct {
	store    *store.Store
	clock    clock.Clock
	cache    *cache.Cache
	pipeline *ingest.Pipeline
	frost    engine.FrostPolicy
	window   validation.WindowRule
	metrics  *metrics.Metrics
	params   Params
}

// New 构造业务服务。
func New(st *store.Store, clk clock.Clock, c *cache.Cache, pipe *ingest.Pipeline,
	m *metrics.Metrics, p Params) *Service {
	return &Service{
		store:    st,
		clock:    clk,
		cache:    c,
		pipeline: pipe,
		frost:    engine.DefaultFrostPolicy(),
		window:   validation.DefaultWindowRule(),
		metrics:  m,
		params:   p.withDefaults(),
	}
}

// Frost 暴露当前防冻策略（测试注入用）。
func (s *Service) Frost() engine.FrostPolicy { return s.frost }

// SetFrost 覆盖防冻策略。
func (s *Service) SetFrost(p engine.FrostPolicy) { s.frost = p }

// activeFrost 依据注入时钟解析防冻季生效状态。
func (s *Service) activeFrost(now time.Time) engine.FrostPolicy {
	return s.frost.ResolveForTime(now)
}
