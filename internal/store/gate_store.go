package store

import (
	"database/sql"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// CreateGate 新建闸门档案。
func (s *Store) CreateGate(g *model.Gate) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO gates(code,chamber_id,kind,clear_width_m,status,misalign_limit_deg,enabled,created_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		g.Code, g.ChamberID, g.Kind, g.ClearWidthM, string(g.Status), g.MisalignLimitDeg,
		boolToInt(g.Enabled), formatTime(now))
	if err != nil {
		return err
	}
	g.ID, err = res.LastInsertId()
	g.CreatedAt = now
	return err
}

// GetGate 按 ID 查询闸门。
func (s *Store) GetGate(id int64) (model.Gate, error) {
	row := s.db.QueryRow(`SELECT id,code,chamber_id,kind,clear_width_m,status,misalign_limit_deg,enabled,created_at
		FROM gates WHERE id=?`, id)
	var (
		g         model.Gate
		status    string
		createdAt string
		enabled   int64
	)
	err := row.Scan(&g.ID, &g.Code, &g.ChamberID, &g.Kind, &g.ClearWidthM, &status,
		&g.MisalignLimitDeg, &enabled, &createdAt)
	if err == sql.ErrNoRows {
		return g, notFound
	}
	if err != nil {
		return g, err
	}
	g.Status = model.GateStatus(status)
	g.Enabled = enabled == 1
	g.CreatedAt = parseTime(createdAt)
	return g, nil
}

// ListGates 列出闸室全部闸门。
func (s *Store) ListGates(chamberID int64) ([]model.Gate, error) {
	rows, err := s.db.Query(`SELECT id FROM gates WHERE chamber_id=? ORDER BY id`, chamberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Gate{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		g, err := s.GetGate(id)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetGateEnabled 启用/停用闸门；不存在时返回 notFound。
func (s *Store) SetGateEnabled(id int64, enabled bool) error {
	res, err := s.db.Exec(`UPDATE gates SET enabled=? WHERE id=?`, boolToInt(enabled), id)
	if err != nil {
		return err
	}
	return requireAffected(res, notFound)
}

// UpdateGateStatus 流转闸门状态机；非法流转影响 0 行返回 conflict。
func (s *Store) UpdateGateStatus(id int64, from, to model.GateStatus, at time.Time) error {
	res, err := s.db.Exec(`UPDATE gates SET status=?, created_at=created_at WHERE id=? AND status=?`,
		string(to), id, string(from))
	if err != nil {
		return err
	}
	_ = at
	if err := requireAffected(res, model.ErrConflict); err != nil {
		return err
	}
	return nil
}

// ListStaleGates 心跳过期扫描：启用的闸位计在 cutoff 后无心跳即视为过期。
// 从未上报过心跳的闸位计同样纳入结果（COALESCE 兜底 NULL）。
func (s *Store) ListStaleSensors(cutoff time.Time) ([]model.GateSensor, error) {
	rows, err := s.db.Query(`
		SELECT gs.id
		FROM gate_sensors gs LEFT JOIN sensor_heartbeats h ON h.sensor_id=gs.id
		WHERE gs.enabled=1 AND gs.kind='position' AND h.seen_at > ?`, formatTime(cutoff))
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
