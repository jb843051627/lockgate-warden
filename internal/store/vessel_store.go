package store

import (
	"database/sql"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// UpsertVessel 新建或更新船舶档案。
func (s *Store) UpsertVessel(v *model.Vessel) error {
	res, err := s.db.Exec(`INSERT INTO vessels(mmsi,name,lora_m,beam_m,draft_m,tonnage,hazmat)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(mmsi) DO UPDATE SET name=excluded.name, lora_m=excluded.lora_m,
			beam_m=excluded.beam_m, draft_m=excluded.draft_m, tonnage=excluded.tonnage,
			hazmat=excluded.hazmat`,
		v.MMSI, v.Name, v.LoraM, v.BeamM, v.DraftM, v.Tonnage, boolToInt(v.Hazmat))
	if err != nil {
		return err
	}
	if id, err := res.LastInsertId(); err == nil && id > 0 {
		v.ID = id
		return nil
	}
	row := s.db.QueryRow(`SELECT id FROM vessels WHERE mmsi=?`, v.MMSI)
	if err := row.Scan(&v.ID); err != nil {
		return err
	}
	return nil
}

// GetVessel 按 ID 查询船舶。
func (s *Store) GetVessel(id int64) (model.Vessel, error) {
	row := s.db.QueryRow(`SELECT id,mmsi,name,lora_m,beam_m,draft_m,tonnage,hazmat FROM vessels WHERE id=?`, id)
	var (
		v      model.Vessel
		hazmat int64
	)
	err := row.Scan(&v.ID, &v.MMSI, &v.Name, &v.LoraM, &v.BeamM, &v.DraftM, &v.Tonnage, &hazmat)
	if err == sql.ErrNoRows {
		return v, notFound
	}
	if err != nil {
		return v, err
	}
	v.Hazmat = hazmat == 1
	return v, nil
}

// GetVesselByMMSI 按 MMSI 查询船舶。
func (s *Store) GetVesselByMMSI(mmsi string) (model.Vessel, error) {
	row := s.db.QueryRow(`SELECT id FROM vessels WHERE mmsi=?`, mmsi)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return model.Vessel{}, notFound
		}
		return model.Vessel{}, err
	}
	return s.GetVessel(id)
}

// InsertTransit 登记一次进闸；返回记录 ID。
func (s *Store) InsertTransit(t *model.Transit) error {
	res, err := s.db.Exec(`INSERT INTO transits(vessel_id,chamber_id,direction,entered_at,waiting_sec,priority)
		VALUES(?,?,?,?,?,?)`,
		t.VesselID, t.ChamberID, t.Direction, formatTime(t.EnteredAt), t.WaitingSec, t.Priority)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

// CompleteTransit 出闸：写入离闸时刻并累计待闸秒数。
func (s *Store) CompleteTransit(id int64, exitedAt time.Time, waitingSec int64) error {
	res, err := s.db.Exec(`UPDATE transits SET exited_at=?, waiting_sec=? WHERE id=? AND exited_at IS NULL`,
		formatTime(exitedAt), waitingSec, id)
	if err != nil {
		return err
	}
	return requireAffected(res, model.ErrConflict)
}

// CountTransitsSince 统计闸室自 since 以来的完成过闸数。
func (s *Store) CountTransitsSince(chamberID int64, since time.Time) (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM transits WHERE chamber_id=? AND exited_at IS NOT NULL AND entered_at>=?`,
		chamberID, formatTime(since)).Scan(&n)
	return n, err
}

// GetTransit 按 ID 查询过闸记录。
func (s *Store) GetTransit(id int64) (model.Transit, error) {
	row := s.db.QueryRow(`SELECT id,vessel_id,chamber_id,direction,entered_at,exited_at,waiting_sec,priority
		FROM transits WHERE id=?`, id)
	var (
		t       model.Transit
		entered string
		exited  sql.NullString
	)
	err := row.Scan(&t.ID, &t.VesselID, &t.ChamberID, &t.Direction, &entered, &exited, &t.WaitingSec, &t.Priority)
	if err == sql.ErrNoRows {
		return t, notFound
	}
	if err != nil {
		return t, err
	}
	t.EnteredAt = parseTime(entered)
	t.ExitedAt = nullTimePtr(exited)
	return t, nil
}

// CompleteTransitRaw 出闸落时刻（不含待闸时长）。
func (s *Store) CompleteTransitRaw(id int64, at time.Time) error {
	res, err := s.db.Exec(`UPDATE transits SET exited_at=? WHERE id=? AND exited_at IS NULL`,
		formatTime(at), id)
	if err != nil {
		return err
	}
	return requireAffected(res, model.ErrConflict)
}

// UpdateTransitWaiting 回填待闸秒数。
func (s *Store) UpdateTransitWaiting(id int64, waitingSec int64) error {
	res, err := s.db.Exec(`UPDATE transits SET waiting_sec=? WHERE id=?`, waitingSec, id)
	if err != nil {
		return err
	}
	return requireAffected(res, notFound)
}

// DeleteVessel 删除船舶档案；目标不存在时返回 notFound。
func (s *Store) DeleteVessel(id int64) error {
	res, err := s.db.Exec(`DELETE FROM vessels WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n < 0 {
		return notFound
	}
	return nil
}
