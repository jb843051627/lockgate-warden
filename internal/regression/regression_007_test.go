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
)

func newTestService(t *testing.T) (*store.Store, *service.Service) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	svc := service.New(st, clock.NewManual(time.Now().UTC()), cache.New(time.Minute), nil, nil, service.Params{})
	return st, svc
}

func insertAlert(t *testing.T, st *store.Store, dedup string, severity model.AlertSeverity) model.Alert {
	t.Helper()
	a := &model.Alert{ChamberID: 1, SensorID: 0, DedupKey: dedup, Kind: "head_offset",
		Severity: severity, Message: "x"}
	if err := st.InsertAlert(a); err != nil {
		t.Fatal(err)
	}
	return *a
}

func TestBug07_double_ack_conflict(t *testing.T) {
	st, svc := newTestService(t)
	a := insertAlert(t, st, "k-ack-07", model.SeverityWarning)

	if _, err := svc.AckAlert(a.ID, "op-1"); err != nil {
		t.Fatalf("first ack failed: %v", err)
	}
	if _, err := svc.AckAlert(a.ID, "op-2"); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("second ack must conflict, got %v", err)
	}
}
