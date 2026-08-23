package store

import (
	"database/sql"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// CreateSensor 新建传感器档案。
func (s *Store) CreateSensor(sen *model.GateSensor) error {
	res, err := s.db.Exec(`INSERT INTO gate_sensors(code,chamber_id,kind,unit,gate_ref_id,enabled,expected_value,tolerance,soft_min,soft_max,hard_min,hard_max)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		sen.Code, sen.ChamberID, string(sen.Kind), sen.Unit, sen.GateRefID, boolToInt(sen.Enabled),
		sen.ExpectedValue, sen.Tolerance, sen.SoftMin, sen.SoftMax, sen.HardMin, sen.HardMax)
	if err != nil {
		return err
	}
	sen.ID, err = res.LastInsertId()
	return err
}

// GetSensor 按 ID 查询传感器。
func (s *Store) GetSensor(id int64) (model.GateSensor, error) {
	row := s.db.QueryRow(`SELECT id,code,chamber_id,kind,unit,gate_ref_id,enabled,expected_value,tolerance,soft_min,soft_max,hard_min,hard_max
		FROM gate_sensors WHERE id=?`, id)
	var (
		sen     model.GateSensor
		kind    string
		enabled int64
	)
	err := row.Scan(&sen.ID, &sen.Code, &sen.ChamberID, &kind, &sen.Unit, &sen.GateRefID,
		&enabled, &sen.ExpectedValue, &sen.Tolerance, &sen.SoftMin, &sen.SoftMax, &sen.HardMin, &sen.HardMax)
	if err == sql.ErrNoRows {
		return sen, notFound
	}
	if err != nil {
		return sen, err
	}
	sen.Kind = model.SensorKind(kind)
	sen.Enabled = enabled == 1
	return sen, nil
}

// GetSensorByCode 按编码查询传感器。
func (s *Store) GetSensorByCode(code string) (model.GateSensor, error) {
	row := s.db.QueryRow(`SELECT id FROM gate_sensors WHERE code=?`, code)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return model.GateSensor{}, notFound
		}
		return model.GateSensor{}, err
	}
	return s.GetSensor(id)
}

// ListSensors 列出闸室全部传感器。
func (s *Store) ListSensors(chamberID int64) ([]model.GateSensor, error) {
	rows, err := s.db.Query(`SELECT id FROM gate_sensors WHERE chamber_id=? ORDER BY id`, chamberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.GateSensor{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sen, err := s.GetSensor(id)
		if err != nil {
			return nil, err
		}
		out = append(out, sen)
	}
	return out, rows.Err()
}

// SetSensorEnabled 启用/停用传感器；不存在时返回 notFound。
func (s *Store) SetSensorEnabled(id int64, enabled bool) error {
	res, err := s.db.Exec(`UPDATE gate_sensors SET enabled=? WHERE id=?`, boolToInt(enabled), id)
	if err != nil {
		return err
	}
	return requireAffected(res, notFound)
}

// TouchHeartbeat 推进传感器心跳（upsert）。
func (s *Store) TouchHeartbeat(sensorID int64, value float64, quality model.Quality, at time.Time, batchID int64) error {
	_, err := s.db.Exec(`INSERT INTO sensor_heartbeats(sensor_id,value,quality,seen_at,batch_id)
		VALUES(?,?,?,?,?)
		ON CONFLICT(sensor_id) DO UPDATE SET value=excluded.value, quality=excluded.quality,
			seen_at=excluded.seen_at, batch_id=excluded.batch_id`,
		sensorID, value, string(quality), formatTime(at), batchID)
	return err
}

// LatestHeartbeat 读取传感器最新心跳。
func (s *Store) LatestHeartbeat(sensorID int64) (model.SensorHeartbeat, error) {
	row := s.db.QueryRow(`SELECT h.sensor_id, gs.code, gs.kind, h.value, h.quality, h.seen_at, h.batch_id
		FROM sensor_heartbeats h JOIN gate_sensors gs ON gs.id=h.sensor_id
		WHERE h.sensor_id=?`, sensorID)
	var (
		hb     model.SensorHeartbeat
		kind   string
		seenAt string
	)
	err := row.Scan(&hb.SensorID, &hb.Code, &kind, &hb.Value, &hb.Quality, &seenAt, &hb.BatchID)
	if err == sql.ErrNoRows {
		return hb, notFound
	}
	if err != nil {
		return hb, err
	}
	hb.Kind = model.SensorKind(kind)
	hb.Quality = model.Quality(hb.Quality)
	hb.SeenAt = parseTime(seenAt)
	return hb, nil
}
