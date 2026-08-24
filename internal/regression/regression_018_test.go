package regression

import (
	"errors"
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
	"github.com/jb843051627/lockgate-warden/internal/validation"
)

func TestBug18_all_points_outside_rejected(t *testing.T) {
	st, err0 := store.Open(t.TempDir() + "/t18.db")
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	c := &model.Chamber{Code: "C-out18", Name: "outside chamber", LengthM: 200, WidthM: 20,
		NormLevelUpM: 10, NormLevelDownM: 4, MaxHeadDiffM: 5}
	if err := st.CreateChamber(c); err != nil {
		t.Fatal(err)
	}
	sen := &model.GateSensor{ChamberID: c.ID, Code: "L-OUT", Kind: model.KindLevel, Unit: "m",
		Enabled: true, Tolerance: 1, SoftMin: 0, SoftMax: 12, HardMin: -2, HardMax: 20}
	if err := st.CreateSensor(sen); err != nil {
		t.Fatal(err)
	}
	clk := clock.NewManual(time.Now().UTC())
	svc := service.New(st, clk, nil, nil, nil, service.Params{})
	now := clk.Now()

	points := []model.TelemetryPointInput{
		{SensorCode: "L-OUT", Seq: 1, TakenAt: now.Add(-30 * time.Minute), Value: 5},
	}
	in := model.BatchInput{
		ChamberCode: c.Code,
		WindowStart: now.Add(-5 * time.Minute),
		WindowEnd:   now,
		Checksum:    validation.ComputeChecksum(points),
		Points:      points,
	}
	res, err := svc.IngestBatch(in)
	if !errors.Is(err, model.ErrEmptyBatch) {
		t.Fatalf("batch fully outside declared window must be rejected with ErrEmptyBatch, got err=%v res=%+v", err, res)
	}
}
