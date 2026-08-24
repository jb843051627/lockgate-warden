package ingest

import (
	"context"

	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/model"
	"github.com/jb843051627/lockgate-warden/internal/store"
)

// EnqueueSnapshotRefresh 投递心跳快照刷新任务：
// 从存储回读最新心跳并写入进程内缓存，失败仅记日志不影响入库链。
func EnqueueSnapshotRefresh(p *Pipeline, st *store.Store, c *cache.Cache, hb model.SensorHeartbeat, batchID int64) {
	if p == nil {
		return
	}
	p.Submit(Task{
		BatchID: batchID,
		Run: func(_ context.Context) error {
			latest, err := st.LatestHeartbeat(hb.SensorID)
			if err != nil {
				return err
			}
			c.PutSensorSnapshot(cache.SensorSnapshot{
				SensorID: latest.SensorID,
				Code:     latest.Code,
				Kind:     string(latest.Kind),
				Value:    latest.Value,
				Quality:  string(latest.Quality),
				BatchID:  hb.BatchID,
			})
			return nil
		},
	})
}
