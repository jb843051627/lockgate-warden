package regression

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
)

func tempFile11(name string) string {
	return filepath.Join(os.TempDir(), "lg11_"+name)
}

func joinTemp11(name string) string {
	dir, _ := os.MkdirTemp("", "lg11")
	return filepath.Join(dir, name)
}

func TestBug11_stale_warning_expire(t *testing.T) {
	st, err0 := store.Open(tempFile11("a.db"))
	if err0 != nil {
		t.Fatal(err0)
	}
	defer st.Close()
	clk := clock.NewManual(time.Now().UTC())
	svc := service.New(st, clk, cache.New(time.Minute), nil, nil, service.Params{WarnTTL: time.Minute})

	fresh := &model.Alert{ChamberID: 1, DedupKey: "C1|wind_restricted", Kind: "wind_restricted",
		Severity: model.SeverityWarning, Message: "just raised"}
	if err := st.InsertAlert(fresh); err != nil {
		t.Fatal(err)
	}

	clk.Advance(2 * time.Minute)
	svc.RunWatchdog()
	got, err := st.GetAlert(fresh.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.AlertClosed {
		t.Fatalf("stale warning must be auto-closed, got %s", got.Status)
	}
}
