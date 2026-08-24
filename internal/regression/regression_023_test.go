package regression

import (
	"errors"
	"testing"

	"github.com/jb843051627/lockgate-warden/internal/engine"
	"github.com/jb843051627/lockgate-warden/internal/model"
)

func TestBug23_head_tolerance_precheck(t *testing.T) {
	if _, err := engine.DeltaRatio(1, 0, 0); !errors.Is(err, engine.ErrBadTolerance) {
		t.Fatalf("zero tolerance must be rejected with ErrBadTolerance, got %v", err)
	}
	if _, err := engine.EvaluateHead(9.5, 3.0, -1); !errors.Is(err, engine.ErrBadTolerance) {
		t.Fatalf("negative tolerance must be rejected, got %v", err)
	}
	err := (&model.LevelBaseline{ChamberID: 1, SensorCode: "LVL-1", ExpectedM: 3, ToleranceM: 0}).Validate()
	if err == nil {
		t.Fatal("baseline with zero tolerance must fail validation")
	}
}
