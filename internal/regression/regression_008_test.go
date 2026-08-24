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

func insertAlert08(t *testing.T, st *store.Store, dedup string, severity model.AlertSeverity) model.Alert {
	t.Helper()
	a := &model.Alert{ChamberID: 1, SensorID: 0, DedupKey: dedup, Kind: "head_offset",
		Severity: severity, Message: "x"}
	if err := st.InsertAlert(a); err != nil {
		t.Fatal(err)
	}
	return *a
}

func TestBug08_close_alert_guards(t *testing.T) {
	st, err0 := store.Open(filepath.Join(t.TempDir(), "t08.db"))
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	svc := service.New(st, clock.NewManual(time.Now().UTC()), cache.New(time.Minute), nil, nil, service.Params{})

	crit := insertAlert08(t, st, "k-critical-08", model.SeverityCritical)
	warn := insertAlert08(t, st, "k-warning-08", model.SeverityWarning)

	if _, err := svc.CloseAlert(crit.ID, "early"); !errors.Is(err, model.ErrAckRequired) {
		t.Fatalf("critical unacked close must require ack, got %v", err)
	}
	if _, err := svc.CloseAlert(warn.ID, "ok"); err != nil {
		t.Fatalf("warning direct close failed: %v", err)
	}
	if _, err := svc.CloseAlert(warn.ID, "again"); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("double close must conflict, got %v", err)
	}
}
