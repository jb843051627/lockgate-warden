package validation

import (
	"fmt"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// CheckSensorOwnership 归属与启用校验：
// 传感器必须存在、必须挂在批次声明闸室上、且处于启用状态。
// 顺序遵循入库链约定：先归属后启用，错误语义互不遮蔽。
func CheckSensorOwnership(sensor model.GateSensor, chamberID int64) error {
	if sensor.ID == 0 {
		return fmt.Errorf("%w: unknown sensor", model.ErrOrphanSensor)
	}
	if sensor.ChamberID != chamberID {
		return fmt.Errorf("%w: sensor %s on chamber %d, batch on chamber %d",
			model.ErrOrphanSensor, sensor.Code, sensor.ChamberID, chamberID)
	}
	if !sensor.Enabled {
		return fmt.Errorf("%w: %s", model.ErrDisabledSensor, sensor.Code)
	}
	return nil
}

// FilterBatchPoints 逐点执行窗口内校验，返回通过的点与被剔除的说明。
// 剔除不整批拒绝，便于运维定位个别漂移点。
func FilterBatchPoints(points []model.TelemetryPointInput, windowStart, windowEnd time.Time) (kept []model.TelemetryPointInput, dropped []string) {
	kept = make([]model.TelemetryPointInput, 0, len(points))
	for _, p := range points {
		if PointWithinBatch(p.TakenAt, windowStart, windowEnd) {
			kept = append(kept, p)
			continue
		}
		dropped = append(dropped, fmt.Sprintf("%s#%d outside declared window", p.SensorCode, p.Seq))
	}
	return kept, dropped
}
