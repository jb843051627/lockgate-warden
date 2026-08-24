package regression

import (
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/engine"
)

func TestBug29_winter_margin_direction(t *testing.T) {
	p := engine.DefaultFrostPolicy()
	jan := p.ResolveForTime(time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC))
	if !jan.Active {
		t.Fatal("january must activate frost policy")
	}

	in := engine.FrostInput{TempC: -2, HumidityPct: 95, WindMS: 1}
	base := engine.EvaluateFrost(in, 1.0)
	withMargin := jan.AssessFrost(in)
	if withMargin.Score <= base.Score {
		t.Fatalf("winter margin must amplify frost score in season, got %.1f vs base %.1f",
			withMargin.Score, base.Score)
	}
	if !withMargin.WinterMarginApplied {
		t.Fatal("margin flag must be set in season")
	}

	limit := jan.MisalignLimit(2.0)
	if limit >= 2.0 {
		t.Fatalf("frost margin must tighten misalignment limit, got %.2f want <2.0", limit)
	}
}
