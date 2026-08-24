package regression

import (
	"context"
	"testing"

	"github.com/jb843051627/lockgate-warden/internal/ingest"
)

func noop(context.Context) error { return nil }

func TestBug28_pipeline_double_close_safe(t *testing.T) {
	p := ingest.New(4, nil, nil)
	p.Submit(ingest.Task{BatchID: 1, Run: noop})
	p.Close()
	p.Close() // 二次 Close 绝不允许 panic: close of closed channel
}
