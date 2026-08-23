package cache

import "fmt"

// SensorSnapshot 传感器最新读数快照（热路径只读对象）。
type SensorSnapshot struct {
	SensorID int64
	Code     string
	Kind     string
	Value    float64
	Quality  string
	SeenAt   string
	BatchID  int64
}

func sensorKey(id int64) string { return fmt.Sprintf("sensor:%d", id) }

// PutSensorSnapshot 写入传感器快照（按值存储，读写隔离）。
func (c *Cache) PutSensorSnapshot(s SensorSnapshot) {
	stored := s
	c.Set(sensorKey(s.SensorID), &stored)
}

// SensorSnapshotByID 返回快照副本；未命中返回 false。
func (c *Cache) SensorSnapshotByID(id int64) (*SensorSnapshot, bool) {
	raw, ok := c.Get(sensorKey(id))
	if !ok {
		return nil, false
	}
	snap, ok := raw.(*SensorSnapshot)
	if !ok {
		return nil, false
	}
	return snap, true
}

// chamberKey 闸室状态缓存键。
func chamberKey(id int64) string { return fmt.Sprintf("chamber:%d", id) }

// PutChamberStatus 缓存闸室运行等级。
func (c *Cache) PutChamberStatus(chamberID int64, status string) {
	c.Set(chamberKey(chamberID), status)
}

// ChamberStatus 读取缓存的闸室等级。
func (c *Cache) ChamberStatus(chamberID int64) (string, bool) {
	raw, ok := c.Get(chamberKey(chamberID))
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	return s, ok
}
