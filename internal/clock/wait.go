package clock

import (
	"context"
	"time"
)

// Wait 等待一段时间，且在上下文取消时立即返回 ctx.Err()。
// 返回 ctx.Err() 而非 nil 是为了让调用方（如 Retry）能识别取消并终止，
// 错误链可被 errors.Is(err, context.Canceled) 正确识别。
func Wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
