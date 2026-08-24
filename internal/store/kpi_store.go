package store

import (
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// DailyThroughput 单日过闸统计桶。
type DailyBucket struct {
	Date           string
	Transits       int64
	AvgWaitingMin  float64
	AlertCloseRate float64
	IntegrityRate  float64
}

// WeeklyBuckets 取最近 days 天的逐日统计（日期升序，缺数据日补空桶）。
func (s *Store) WeeklyBuckets(chamberID int64, days int, now time.Time) ([]DailyBucket, error) {
	out := make([]DailyBucket, 0, days)
	start := now.AddDate(0, 0, -days+1)
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := dayStart.Add(24 * time.Hour)
		b := DailyBucket{Date: dayStart.Format("2006-01-02")}
		var (
			closed int64
			opened int64
			tally  model.QualityTally
		)
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM transits
			WHERE chamber_id=? AND exited_at IS NOT NULL AND entered_at>=? AND entered_at<?`,
			chamberID, formatTime(dayStart), formatTime(dayEnd)).Scan(&b.Transits); err != nil {
			return nil, err
		}
		if err := s.db.QueryRow(`SELECT COALESCE(AVG(waiting_sec)/60.0,0) FROM transits
			WHERE chamber_id=? AND exited_at IS NOT NULL AND entered_at>=? AND entered_at<?`,
			chamberID, formatTime(dayStart), formatTime(dayEnd)).Scan(&b.AvgWaitingMin); err != nil {
			return nil, err
		}
		if err := s.db.QueryRow(`SELECT
			COALESCE(SUM(CASE WHEN status='closed' THEN 1 ELSE 0 END),0),
			COUNT(*)
			FROM alerts WHERE chamber_id=? AND first_seen_at>=? AND first_seen_at<?`,
			chamberID, formatTime(dayStart), formatTime(dayEnd)).Scan(&closed, &opened); err != nil {
			return nil, err
		}
		if opened > 0 {
			b.AlertCloseRate = float64(closed) / float64(opened)
		} else {
			b.AlertCloseRate = 1
		}
		rows, err := s.db.Query(`SELECT p.quality, COUNT(*) FROM telemetry_points p
			JOIN gate_sensors gs ON gs.id=p.sensor_id
			WHERE gs.chamber_id=? AND p.taken_at>=? AND p.taken_at<? GROUP BY p.quality`,
			chamberID, formatTime(dayStart), formatTime(dayEnd))
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var (
				quality string
				count   int64
			)
			if err := rows.Scan(&quality, &count); err != nil {
				rows.Close()
				return nil, err
			}
			switch model.Quality(quality) {
			case model.QualityGood:
				tally.Good += count
			case model.QualitySuspect:
				tally.Suspect += count
			default:
				tally.Rejected += count
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		b.IntegrityRate = tally.IntegrityRate()
		out = append(out, b)
	}
	return out, nil
}

// TotalOpenAlerts 全库未关闭告警数。
func (s *Store) TotalOpenAlerts() (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE status IN ('open','acked')`).Scan(&n)
	return n, err
}

// AlertCloseCounts 统计周期内新增告警的关闭情况：返回 (closed, opened)。
func (s *Store) AlertCloseCounts(chamberID int64, since time.Time) (closed int64, opened int64, err error) {
	err = s.db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN status='closed' THEN 1 ELSE 0 END),0),
		COUNT(*)
		FROM alerts WHERE chamber_id=? AND first_seen_at>=?`,
		chamberID, formatTime(since)).Scan(&closed, &opened)
	return closed, opened, err
}

// CountChambersSince 统计自 since 以来的活跃闸室数。
func (s *Store) CountChambersSince(since time.Time) (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(DISTINCT chamber_id) FROM telemetry_batches WHERE received_at>=?`,
		formatTime(since)).Scan(&n)
	return n, err
}
