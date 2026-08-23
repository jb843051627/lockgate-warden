package store

import (
	"time"
)

// HourlyLoad 单小时入库负载统计。
type HourlyLoad struct {
	Hour    string
	Batches int64
	Points  int64
	Rejects int64
}

// HourlyLoads 取闸室自 since 以来逐小时负载（含全库视角 chamberID=0）。
func (s *Store) HourlyLoads(chamberID int64, since time.Time) ([]HourlyLoad, error) {
	join := ""
	where := "p.taken_at>=?"
	args := []any{formatTime(since)}
	if chamberID > 0 {
		join = " JOIN gate_sensors gs ON gs.id=p.sensor_id "
		where += " AND gs.chamber_id=?"
		args = append(args, chamberID)
	}
	rows, err := s.db.Query(`SELECT substr(p.taken_at,1,13) AS hr, COUNT(DISTINCT p.batch_id), COUNT(*)
		FROM telemetry_points p`+join+`WHERE `+where+` GROUP BY hr ORDER BY hr`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HourlyLoad{}
	for rows.Next() {
		var h HourlyLoad
		if err := rows.Scan(&h.Hour, &h.Batches, &h.Points); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rejRows, err := s.db.Query(`SELECT substr(p.taken_at,1,13) AS hr, COUNT(*)
		FROM telemetry_points p`+join+`WHERE `+where+` AND p.quality='rejected' GROUP BY hr`, args...)
	if err != nil {
		return nil, err
	}
	defer rejRows.Close()
	idx := map[string]int{}
	for i, h := range out {
		idx[h.Hour] = i
	}
	for rejRows.Next() {
		var (
			hr string
			n  int64
		)
		if err := rejRows.Scan(&hr, &n); err != nil {
			return nil, err
		}
		if i, ok := idx[hr]; ok {
			out[i].Rejects = n
		}
	}
	return out, rejRows.Err()
}

// DwellStats 过闸待闸时长分位数（分钟）：p50/p95。
func (s *Store) DwellStats(chamberID int64, since time.Time) (p50 float64, p95 float64, err error) {
	rows, err := s.db.Query(`SELECT waiting_sec FROM transits
		WHERE chamber_id=? AND exited_at IS NOT NULL AND entered_at>=? ORDER BY waiting_sec`,
		chamberID, formatTime(since))
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	var secs []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return 0, 0, err
		}
		secs = append(secs, v)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	if len(secs) == 0 {
		return 0, 0, nil
	}
	pick := func(q float64) float64 {
		i := int(q * float64(len(secs)-1))
		return float64(secs[i]) / 60.0
	}
	return pick(0.50), pick(0.95), nil
}

// SensorUptime 传感器在线率：心跳数 / 应答点数（按 10 分钟粒度粗估）。
func (s *Store) SensorUptime(sensorID int64, since time.Time) (float64, error) {
	var beats int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sensor_heartbeats h
		JOIN telemetry_points p ON p.sensor_id=h.sensor_id
		WHERE h.sensor_id=? AND p.taken_at>=?`, sensorID, formatTime(since)).Scan(&beats)
	if err != nil {
		return 0, err
	}
	expected := int64(time.Since(since).Minutes() / 10)
	if expected <= 0 {
		return 1, nil
	}
	ratio := float64(beats) / float64(expected)
	if ratio > 1 {
		ratio = 1
	}
	return ratio, nil
}
