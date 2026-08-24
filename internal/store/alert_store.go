package store

import (
	"database/sql"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// FindDedupCandidate 在去重窗口内查找同键活跃告警（open/acked）。
func (s *Store) FindDedupCandidate(dedupKey string, windowStart time.Time) (model.Alert, error) {
	row := s.db.QueryRow(`SELECT id,chamber_id,sensor_id,dedup_key,kind,severity,message,status,occurrences,
		first_seen_at,latest_seen_at,acked_by,acked_at,closed_at,close_note,updated_at
		FROM alerts WHERE dedup_key=? AND status IN ('open','acked') AND latest_seen_at>=?
		ORDER BY id DESC LIMIT 1`, dedupKey, formatTime(windowStart))
	return scanAlert(row.Scan)
}

// InsertAlert 新建告警。
func (s *Store) InsertAlert(a *model.Alert) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO alerts(chamber_id,sensor_id,dedup_key,kind,severity,message,status,occurrences,
		first_seen_at,latest_seen_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		a.ChamberID, a.SensorID, a.DedupKey, a.Kind, string(a.Severity), a.Message, string(model.AlertOpen),
		a.Occurrences, formatTime(now), formatTime(now), formatTime(now))
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	a.Status = model.AlertOpen
	a.FirstSeenAt, a.LatestSeenAt, a.UpdatedAt = now, now, now
	return nil
}

// TouchAlert 命中去重窗口：累加次数并推进 latest_seen_at。
func (s *Store) TouchAlert(id int64, occurrences int64, seenAt time.Time) error {
	res, err := s.db.Exec(`UPDATE alerts SET occurrences=?, latest_seen_at=?, updated_at=? WHERE id=?`,
		occurrences, formatTime(seenAt), formatTime(seenAt), id)
	if err != nil {
		return err
	}
	return requireAffected(res, notFound)
}

// GetAlert 按 ID 查询告警。
func (s *Store) GetAlert(id int64) (model.Alert, error) {
	row := s.db.QueryRow(alertSelect+` WHERE id=?`, id)
	alert, err := scanAlert(row.Scan)
	if err == sql.ErrNoRows {
		return alert, notFound
	}
	return alert, err
}

// ListAlerts 按状态过滤列出告警；status 为空返回全部。
func (s *Store) ListAlerts(status string, limit int) ([]model.Alert, error) {
	if limit <= 0 {
		limit = 200
	}
	query := alertSelect
	args := []any{}
	if status != "" {
		query += ` WHERE status=?`
		args = append(args, status)
	}
	query += ` ORDER BY latest_seen_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Alert{}
	for rows.Next() {
		a, err := scanAlert(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AckAlert open → acked，记录操作人与时刻。
func (s *Store) AckAlert(id int64, by string, at time.Time) error {
	res, err := s.db.Exec(`UPDATE alerts SET status='acked', acked_by=?, acked_at=?, updated_at=?
		WHERE id=? AND status='open'`, by, formatTime(at), formatTime(at), id)
	if err != nil {
		return err
	}
	return requireAffected(res, model.ErrConflict)
}

// CloseAlert 关闭告警；仅允许 open/acked → closed，已 closed 返回冲突。
// 非 critical 允许 open → closed、critical 需先 ack（由 service 校验级别）。
func (s *Store) CloseAlert(id int64, note string, at time.Time) error {
	res, err := s.db.Exec(`UPDATE alerts SET status='closed', close_note=?, closed_at=?, updated_at=?
		WHERE id=? AND status IN ('open','acked')`, note, formatTime(at), formatTime(at), id)
	if err != nil {
		return err
	}
	return requireAffected(res, model.ErrConflict)
}

// AutoCloseStaleWarnings 批量关闭超时的 open warning 告警，返回关闭数量。
func (s *Store) AutoCloseStaleWarnings(cutoff time.Time, at time.Time) (int64, error) {
	res, err := s.db.Exec(`UPDATE alerts SET status='closed', close_note='auto: stale warning', closed_at=?, updated_at=?
		WHERE status='open' AND severity='warning' AND latest_seen_at<?`,
		formatTime(at), formatTime(at), formatTime(cutoff))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CountOpenBySeverity 统计未关闭告警数量（含 open 与 acked）。
func (s *Store) CountOpenBySeverity() (warning int64, critical int64, err error) {
	rows, err := s.db.Query(`SELECT severity, COUNT(*) FROM alerts WHERE status IN ('open','acked') GROUP BY severity`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			sev string
			n   int64
		)
		if err := rows.Scan(&sev, &n); err != nil {
			return 0, 0, err
		}
		switch model.AlertSeverity(sev) {
		case model.SeverityCritical:
			critical += n
		default:
			warning += n
		}
	}
	return warning, critical, rows.Err()
}

const alertSelect = `SELECT id,chamber_id,sensor_id,dedup_key,kind,severity,message,status,occurrences,
	first_seen_at,latest_seen_at,acked_by,acked_at,closed_at,close_note,updated_at FROM alerts`

func scanAlert(scan func(dest ...any) error) (model.Alert, error) {
	var (
		a            model.Alert
		sensorID     sql.NullInt64
		severity     string
		status       string
		firstSeenAt  string
		latestSeenAt string
		ackedAt      sql.NullString
		closedAt     sql.NullString
		updatedAt    string
	)
	if err := scan(&a.ID, &a.ChamberID, &sensorID, &a.DedupKey, &a.Kind, &severity, &a.Message, &status,
		&a.Occurrences, &firstSeenAt, &latestSeenAt, &a.AckedBy, &ackedAt, &closedAt, &a.CloseNote, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return a, notFound
		}
		return a, err
	}
	sev, err := model.ParseAlertSeverity(severity)
	if err != nil {
		return a, err
	}
	a.Severity = sev
	a.Status = model.AlertStatus(status)
	a.SensorID = sensorID.Int64
	a.FirstSeenAt = parseTime(firstSeenAt)
	a.LatestSeenAt = parseTime(latestSeenAt)
	a.AckedAt = nullTimePtr(ackedAt)
	a.ClosedAt = nullTimePtr(closedAt)
	a.UpdatedAt = parseTime(updatedAt)
	return a, nil
}
