package regression

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jb843051627/lockgate-warden/internal/metrics"
)

func TestBug02_metrics_add_snapshot_alias(t *testing.T) {
	m := metrics.New()
	m.Inc(metrics.BatchesAccepted)
	const workers = 32
	const rounds = 100
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			for j := 0; j < rounds; j++ {
				m.Add("adhoc_probe_counter", 1)
				m.Add(fmt.Sprintf("probe_%d_%d", n, j), 1)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	snap := m.Snapshot()
	if snap[metrics.BatchesAccepted] != 1 {
		t.Fatalf("declared counter drifted: got %d want 1", snap[metrics.BatchesAccepted])
	}
	if snap["adhoc_probe_counter"] != workers*rounds {
		t.Fatalf("lost ad-hoc updates: got %d want %d", snap["adhoc_probe_counter"], workers*rounds)
	}
}
