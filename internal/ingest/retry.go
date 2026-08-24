package ingest

import (
	"context"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/clock"
)

// Retry 以线性递增退避重试 fn 至多 attempts 次。
// 返回最后一次错误；ctx 取消立即终止并返回 ctx.Err()。
func Retry(ctx context.Context, attempts int, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		// 在执行 fn 前检查 ctx，避免上游取消后仍触发回调。
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = fn(); err == nil {
			return nil
		}
		backoff := time.Duration(i+1) * 20 * time.Millisecond
		if err = clock.Wait(ctx, backoff); err != nil {
			return err
		}
	}
	return err
}
