package regression

import (
	"testing"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/cache"
)

func TestBug03_snapshot_pointer_isolation(t *testing.T) {
	c := cache.New(time.Minute)
	c.PutSensorSnapshot(cache.SensorSnapshot{SensorID: 7, Code: "P-07", Value: 12.5})
	got, ok := c.SensorSnapshotByID(7)
	if !ok || got == nil {
		t.Fatal("snapshot miss on first read")
	}
	got.Value = 99
	again, ok := c.SensorSnapshotByID(7)
	if !ok {
		t.Fatal("snapshot miss on re-read")
	}
	if again.Value != 12.5 {
		t.Fatalf("cached snapshot mutated through returned pointer: got %.2f want 12.50", again.Value)
	}
}
