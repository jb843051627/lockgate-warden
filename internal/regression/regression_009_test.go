package regression

import (
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

func ingestWind(t *testing.T, svc *service.Service, clk *clock.Manual, chamber string, code string, value float64) {
	t.Helper()
	now := clk.Now()
	points := []model.TelemetryPointInput{
		{SensorCode: code, Seq: 1, TakenAt: now.Add(-30 * time.Second), Value: value},
	}
	in := model.BatchInput{
		ChamberCode: chamber,
		WindowStart: now.Add(-5 * time.Minute),
		WindowEnd:   now,
		Checksum:    validation.ComputeChecksum(points),
		Points:      points,
	}
	if _, err := svc.IngestBatch(in); err != nil {
		t.Fatalf("ingest batch for %s: %v", code, err)
	}
}

func TestBug09_dedup_key_sensor_dim(t *testing.T) {
	st, err0 := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	c := &model.Chamber{Code: "C-dedup", Name: "dedup", LengthM: 200, WidthM: 20,
		NormLevelUpM: 10, NormLevelDownM: 4, MaxHeadDiffM: 5}
	if err := st.CreateChamber(c); err != nil {
		t.Fatal(err)
	}
	mk := func(code string) *model.GateSensor {
		return &model.GateSensor{ChamberID: c.ID, Code: code, Kind: model.KindWind, Unit: "m/s",
			Enabled: true, Tolerance: 1, SoftMin: 0, SoftMax: 25, HardMin: -10, HardMax: 60}
	}
	for _, sen := range []*model.GateSensor{mk("W-A"), mk("W-B")} {
		if err := st.CreateSensor(sen); err != nil {
			t.Fatal(err)
		}
	}
	clk := clock.NewManual(time.Now().UTC())
	svc := service.New(st, clk, cache.New(time.Minute), nil, nil, service.Params{})

	ingestWind(t, svc, clk, c.Code, "W-A", 29)
	ingestWind(t, svc, clk, c.Code, "W-B", 29)

	opens, err := st.ListAlerts("open", 100)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, al := range opens {
		if al.Kind == "wind_critical" && al.SensorID > 0 {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("two sensors must raise two independent alerts, got %d", n)
	}
}
