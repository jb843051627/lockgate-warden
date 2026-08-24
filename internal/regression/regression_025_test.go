package regression

import (
	"math"
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/engine"
)

func engineNow() time.Time { return time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC) }

func TestBug25_clearance_rate_zero_span(t *testing.T) {
	now := engineNow()
	single := []engine.ClearanceSample{{At: now, Cumulative: 5}}
	rate, ok := engine.ClearanceRate(single, time.Hour, now)
	if ok && math.IsNaN(rate) {
		t.Fatalf("zero-span window must not yield NaN rate, got %v", rate)
	}
	if len(single) < 2 && ok {
		t.Fatalf("single-sample window must report no data, got (%v,%v)", rate, ok)
	}
	flat := []engine.ClearanceSample{{At: now, Cumulative: 5}, {At: now, Cumulative: 7}}
	flatRate, flatOK := engine.ClearanceRate(flat, time.Hour, now)
	if flatOK || math.IsNaN(flatRate) || math.IsInf(flatRate, 0) {
		t.Fatalf("same-timestamp samples must report finite no-data result, got (%v,%v)", flatRate, flatOK)
	}

	reset := []engine.ClearanceSample{
		{At: now.Add(-time.Hour), Cumulative: 9},
		{At: now, Cumulative: 3},
	}
	r2, ok2 := engine.ClearanceRate(reset, 2*time.Hour, now)
	if ok2 || r2 != 0 || math.IsNaN(r2) {
		t.Fatalf("counter reset must report no valid rate, got (%v,%v)", r2, ok2)
	}
}
