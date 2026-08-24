package regression

import (
	"math"
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
)

func TestBug26_kpi_close_rate_zero_opened(t *testing.T) {
	st, err0 := store.Open(t.TempDir() + "/t26.db")
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	c := &model.Chamber{Code: "C-kpi26", Name: "kpi chamber", LengthM: 200, WidthM: 20,
		NormLevelUpM: 10, NormLevelDownM: 4, MaxHeadDiffM: 5}
	if err := st.CreateChamber(c); err != nil {
		t.Fatal(err)
	}
	svc := service.New(st, clock.NewManual(time.Now().UTC()), nil, nil, nil, service.Params{})

	rep, err := svc.LineWeeklyKPI(c.ID, 7)
	if err != nil {
		t.Fatalf("weekly kpi: %v", err)
	}
	if math.IsNaN(rep.CloseRate) || math.IsInf(rep.CloseRate, 0) {
		t.Fatalf("close rate with zero opened alerts must stay finite, got %v", rep.CloseRate)
	}
	for _, b := range rep.Buckets {
		if math.IsNaN(b.AlertCloseRate) || math.IsInf(b.AlertCloseRate, 0) ||
			math.IsNaN(b.IntegrityRate) || math.IsInf(b.IntegrityRate, 0) {
			t.Fatalf("day %s rates must stay finite, got close=%v integrity=%v",
				b.Date, b.AlertCloseRate, b.IntegrityRate)
		}
	}
}
