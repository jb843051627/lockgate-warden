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

func freshWindEnv(t *testing.T) (*store.Store, *service.Service, *clock.Manual, *model.Chamber) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	c := &model.Chamber{Code: "C-w", Name: "wind chamber", LengthM: 200, WidthM: 20,
		NormLevelUpM: 10, NormLevelDownM: 4, MaxHeadDiffM: 5}
	if err := st.CreateChamber(c); err != nil {
		t.Fatal(err)
	}
	sen := &model.GateSensor{ChamberID: c.ID, Code: "W-X", Kind: model.KindWind, Unit: "m/s",
		Enabled: true, Tolerance: 1, SoftMin: 0, SoftMax: 25, HardMin: -10, HardMax: 60}
	if err := st.CreateSensor(sen); err != nil {
		t.Fatal(err)
	}
	clk := clock.NewManual(time.Now().UTC())
	svc := service.New(st, clk, cache.New(time.Minute), nil, nil, service.Params{})
	return st, svc, clk, c
}

func windIngest(t *testing.T, svc *service.Service, clk *clock.Manual, chamber string, seq int64, value float64) {
	t.Helper()
	now := clk.Now()
	points := []model.TelemetryPointInput{
		{SensorCode: "W-X", Seq: seq, TakenAt: now.Add(-30 * time.Second), Value: value},
	}
	in := model.BatchInput{
		ChamberCode: chamber,
		WindowStart: now.Add(-5 * time.Minute),
		WindowEnd:   now,
		Checksum:    validation.ComputeChecksum(points),
		Points:      points,
	}
	if _, err := svc.IngestBatch(in); err != nil {
		t.Fatalf("ingest: %v", err)
	}
}

func TestBug10_dedup_closed_entry(t *testing.T) {
	st, svc, clk, c := freshWindEnv(t)

	windIngest(t, svc, clk, c.Code, 1, 29)
	opens, err := st.ListAlerts("open", 100)
	if err != nil || len(opens) != 1 {
		t.Fatalf("setup: expected 1 open alert, got %d (err=%v)", len(opens), err)
	}
	firstID := opens[0].ID

	// 先 ack 再关闭（critical 告警），推进时钟越过去重窗口，再触发同类事件。
	if _, err := svc.AckAlert(firstID, "op"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CloseAlert(firstID, "done"); err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * time.Hour)
	windIngest(t, svc, clk, c.Code, 2, 29)

	opens, err = st.ListAlerts("open", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(opens) != 1 || opens[0].ID == firstID {
		t.Fatalf("closed alert beyond window must stay closed; want one fresh alert, got %+v", opens)
	}
}
