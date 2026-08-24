package regression

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
	"github.com/jb843051627/lockgate-warden/internal/validation"
)

func TestBug05_error_wrap_chain(t *testing.T) {
	st, stErr := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if stErr != nil {
		t.Fatal(stErr)
	}
	defer st.Close()
	svc := service.New(st, clock.NewManual(time.Now().UTC()), cache.New(time.Minute), nil, nil, service.Params{})

	now := time.Now().UTC()
	points := []model.TelemetryPointInput{{SensorCode: "GHOST", Seq: 1, TakenAt: now.Add(-time.Minute), Value: 1}}
	_, err := svc.IngestBatch(model.BatchInput{ChamberCode: "NOPE", WindowStart: now.Add(-5 * time.Minute), WindowEnd: now, Points: points, Checksum: validation.ComputeChecksum(points)})
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("ingest unknown chamber must keep ErrNotFound chain, got %v", err)
	}

	_, err = svc.UpsertBaseline(&model.LevelBaseline{
		ChamberID: 1, SensorCode: "GHOST", ExpectedM: 3, ToleranceM: 1,
	})
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("baseline unknown sensor must keep ErrNotFound chain, got %v", err)
	}

}
