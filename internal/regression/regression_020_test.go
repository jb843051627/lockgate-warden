package regression

import (
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
	"github.com/jb843051627/lockgate-warden/internal/validation"
)

func TestBug20_batch_checksum_idempotent_id(t *testing.T) {
	st, err0 := store.Open(t.TempDir() + "/t20.db")
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	c := &model.Chamber{Code: "C-idem20", Name: "idem chamber", LengthM: 200, WidthM: 20,
		NormLevelUpM: 10, NormLevelDownM: 4, MaxHeadDiffM: 5}
	if err := st.CreateChamber(c); err != nil {
		t.Fatal(err)
	}
	sen := &model.GateSensor{ChamberID: c.ID, Code: "L-IDEM", Kind: model.KindLevel, Unit: "m",
		Enabled: true, Tolerance: 1, SoftMin: 0, SoftMax: 12, HardMin: -2, HardMax: 20}
	if err := st.CreateSensor(sen); err != nil {
		t.Fatal(err)
	}
	svc := service.New(st, clock.NewManual(time.Now().UTC()), cache.New(time.Minute), nil, nil, service.Params{})
	now := time.Now().UTC()

	points := []model.TelemetryPointInput{
		{SensorCode: "L-IDEM", Seq: 1, TakenAt: now.Add(-30 * time.Second), Value: 6},
	}
	in := model.BatchInput{
		ChamberCode: c.Code,
		WindowStart: now.Add(-5 * time.Minute),
		WindowEnd:   now,
		Checksum:    validation.ComputeChecksum(points),
		Points:      points,
	}
	first, err := svc.IngestBatch(in)
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := svc.IngestBatch(in)
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if second.BatchID != first.BatchID {
		t.Fatalf("identical checksum resubmission must reuse batch id %d, got %d", first.BatchID, second.BatchID)
	}
	if second.BatchID <= 0 {
		t.Fatalf("batch id must stay positive after resubmission, got %d", second.BatchID)
	}
}
