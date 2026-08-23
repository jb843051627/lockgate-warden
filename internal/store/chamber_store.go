package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// formatTime / parseTime：SQLite 文本时刻统一 UTC RFC3339。
func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func nullTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t := parseTime(ns.String)
	return &t
}

// CreateChamber 新建闸室档案。
func (s *Store) CreateChamber(c *model.Chamber) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO chambers(code,name,length_m,width_m,norm_level_up_m,norm_level_down_m,max_head_diff_m,status,created_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		c.Code, c.Name, c.LengthM, c.WidthM, c.NormLevelUpM, c.NormLevelDownM, c.MaxHeadDiffM, model.LockOpen, formatTime(now))
	if err != nil {
		return err
	}
	c.ID, err = res.LastInsertId()
	c.CreatedAt = now
	return err
}

// GetChamber 按 ID 查询闸室。
func (s *Store) GetChamber(id int64) (model.Chamber, error) {
	row := s.db.QueryRow(`SELECT id,code,name,length_m,width_m,norm_level_up_m,norm_level_down_m,max_head_diff_m,status,created_at
		FROM chambers WHERE id=?`, id)
	var (
		c         model.Chamber
		status    string
		createdAt string
	)
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.LengthM, &c.WidthM, &c.NormLevelUpM, &c.NormLevelDownM,
		&c.MaxHeadDiffM, &status, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, notFound
	}
	if err != nil {
		return c, err
	}
	c.Status = model.LockStatus(status)
	c.CreatedAt = parseTime(createdAt)
	return c, nil
}

// GetChamberByCode 按编号查询闸室。
func (s *Store) GetChamberByCode(code string) (model.Chamber, error) {
	row := s.db.QueryRow(`SELECT id FROM chambers WHERE code=?`, code)
	var id int64
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Chamber{}, notFound
		}
		return model.Chamber{}, err
	}
	return s.GetChamber(id)
}

// ListChambers 列出全部闸室。
func (s *Store) ListChambers() ([]model.Chamber, error) {
	rows, err := s.db.Query(`SELECT id FROM chambers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Chamber{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		c, err := s.GetChamber(id)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateChamberStatus 同步评估等级到闸室档案。
func (s *Store) UpdateChamberStatus(chamberID int64, status model.LockStatus, at time.Time) error {
	res, err := s.db.Exec(`UPDATE chambers SET status=?, created_at=created_at WHERE id=?`,
		string(status), chamberID)
	if err != nil {
		return err
	}
	_ = at
	return requireAffected(res, notFound)
}
