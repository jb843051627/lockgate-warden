package regression

import (
	"math"
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
)

func TestBug27_integrity_rate_zero_total(t *testing.T) {
	var tally model.QualityTally
	if rate := tally.IntegrityRate(); math.IsNaN(rate) || math.IsInf(rate, 0) {
		t.Fatalf("empty tally integrity rate must stay finite (want 1), got %v", rate)
	}

	st, err0 := store.Open(t.TempDir() + "/t27.db")
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	c := &model.Chamber{Code: "C-ir27", Name: "integrity chamber", LengthM: 200, WidthM: 20,
		NormLevelUpM: 10, NormLevelDownM: 4, MaxHeadDiffM: 5}
	if err := st.CreateChamber(c); err != nil {
		t.Fatal(err)
	}
	svc := service.New(st, clock.NewManual(time.Now().UTC()), cache.New(time.Minute), nil, nil, service.Params{})

	rep, err := svc.RunAssessment(c.ID)
	if err != nil {
		t.Fatalf("assessment on empty chamber: %v", err)
	}
	if math.IsNaN(rep.IntegrityRate) || math.IsInf(rep.IntegrityRate, 0) {
		t.Fatalf("assessment integrity rate on zero data must stay finite, got %v", rep.IntegrityRate)
	}
}
