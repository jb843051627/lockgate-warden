package store

import (
	"database/sql"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// CreateScheduleEntry 新建排档条目。
func (s *Store) CreateScheduleEntry(e *model.ScheduleEntry) error {
	res, err := s.db.Exec(`INSERT INTO schedule_entries(chamber_id,vessel_id,slot_start,slot_end,priority,status)
		VALUES(?,?,?,?,?,?)`,
		e.ChamberID, e.VesselID, formatTime(e.SlotStart), formatTime(e.SlotEnd), e.Priority, e.Status)
	if err != nil {
		return err
	}
	e.ID, err = res.LastInsertId()
	return err
}

// GetScheduleEntry 按 ID 查询排档。
func (s *Store) GetScheduleEntry(id int64) (model.ScheduleEntry, error) {
	row := s.db.QueryRow(`SELECT id,chamber_id,vessel_id,slot_start,slot_end,priority,status
		FROM schedule_entries WHERE id=?`, id)
	var (
		e     model.ScheduleEntry
		start string
		end   string
	)
	err := row.Scan(&e.ID, &e.ChamberID, &e.VesselID, &start, &end, &e.Priority, &e.Status)
	if err == sql.ErrNoRows {
		return e, notFound
	}
	if err != nil {
		return e, err
	}
	e.SlotStart = parseTime(start)
	e.SlotEnd = parseTime(end)
	return e, nil
}

// ListQueuedByChamber 列出闸室待执行排档，按优先级与时间排序。
func (s *Store) ListQueuedByChamber(chamberID int64, limit int) ([]model.ScheduleEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id,chamber_id,vessel_id,slot_start,slot_end,priority,status
		FROM schedule_entries WHERE chamber_id=? AND status='queued'
		ORDER BY priority, slot_start LIMIT ?`, chamberID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ScheduleEntry{}
	for rows.Next() {
		var (
			e     model.ScheduleEntry
			start string
			end   string
		)
		if err := rows.Scan(&e.ID, &e.ChamberID, &e.VesselID, &start, &end, &e.Priority, &e.Status); err != nil {
			return nil, err
		}
		e.SlotStart = parseTime(start)
		e.SlotEnd = parseTime(end)
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkScheduleDone 排档完成出队。
func (s *Store) MarkScheduleDone(id int64, at time.Time) error {
	res, err := s.db.Exec(`UPDATE schedule_entries SET status='done' WHERE id=? AND status IN ('queued','admitted')`, id)
	if err != nil {
		return err
	}
	_ = at
	return requireAffected(res, model.ErrConflict)
}
