package store

import "fmt"

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS chambers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			length_m REAL NOT NULL,
			width_m REAL NOT NULL,
			norm_level_up_m REAL NOT NULL,
			norm_level_down_m REAL NOT NULL,
			max_head_diff_m REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS gates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			kind TEXT NOT NULL,
			clear_width_m REAL NOT NULL,
			status TEXT NOT NULL DEFAULT 'sealed',
			misalign_limit_deg REAL NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS gate_sensors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			kind TEXT NOT NULL,
			unit TEXT NOT NULL,
			gate_ref_id INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			expected_value REAL NOT NULL DEFAULT 0,
			tolerance REAL NOT NULL DEFAULT 1,
			soft_min REAL NOT NULL DEFAULT 0,
			soft_max REAL NOT NULL DEFAULT 100,
			hard_min REAL NOT NULL DEFAULT -100,
			hard_max REAL NOT NULL DEFAULT 200
		)`,
		`CREATE TABLE IF NOT EXISTS sensor_heartbeats (
			sensor_id INTEGER PRIMARY KEY REFERENCES gate_sensors(id),
			value REAL NOT NULL,
			quality TEXT NOT NULL,
			seen_at TEXT NOT NULL,
			batch_id INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS vessels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mmsi TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			lora_m REAL NOT NULL,
			beam_m REAL NOT NULL,
			draft_m REAL NOT NULL,
			tonnage REAL NOT NULL,
			hazmat INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS transits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vessel_id INTEGER NOT NULL REFERENCES vessels(id),
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			direction TEXT NOT NULL,
			entered_at TEXT NOT NULL,
			exited_at TEXT,
			waiting_sec INTEGER NOT NULL DEFAULT 0,
			priority INTEGER NOT NULL DEFAULT 2
		)`,
		`CREATE TABLE IF NOT EXISTS schedule_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			vessel_id INTEGER NOT NULL REFERENCES vessels(id),
			slot_start TEXT NOT NULL,
			slot_end TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 2,
			status TEXT NOT NULL DEFAULT 'queued'
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL,
			sensor_id INTEGER NOT NULL DEFAULT 0,
			dedup_key TEXT NOT NULL,
			kind TEXT NOT NULL,
			severity TEXT NOT NULL,
			message TEXT NOT NULL,
			status TEXT NOT NULL,
			occurrences INTEGER NOT NULL DEFAULT 1,
			first_seen_at TEXT NOT NULL,
			latest_seen_at TEXT NOT NULL,
			acked_by TEXT NOT NULL DEFAULT '',
			acked_at TEXT,
			closed_at TEXT,
			close_note TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_dedup ON alerts(dedup_key, status)`,
		`CREATE TABLE IF NOT EXISTS telemetry_batches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			window_start TEXT NOT NULL,
			window_end TEXT NOT NULL,
			point_count INTEGER NOT NULL,
			checksum INTEGER NOT NULL,
			received_at TEXT NOT NULL,
			UNIQUE(chamber_id, checksum)
		)`,
		`CREATE TABLE IF NOT EXISTS telemetry_points (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES telemetry_batches(id),
			sensor_id INTEGER NOT NULL REFERENCES gate_sensors(id),
			seq INTEGER NOT NULL,
			taken_at TEXT NOT NULL,
			value REAL NOT NULL,
			quality TEXT NOT NULL,
			inserted_at TEXT NOT NULL,
			UNIQUE(batch_id, sensor_id, seq)
		)`,
		`CREATE TABLE IF NOT EXISTS level_baselines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			sensor_code TEXT NOT NULL,
			expected_m REAL NOT NULL,
			temp_coeff_m REAL NOT NULL,
			ambient_temp_c REAL NOT NULL DEFAULT 20,
			tolerance_m REAL NOT NULL,
			valid_from TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS assessments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL,
			head_score REAL NOT NULL,
			gate_score REAL NOT NULL,
			pump_score REAL NOT NULL,
			integrity_rate REAL NOT NULL,
			level TEXT NOT NULL,
			frost_active INTEGER NOT NULL DEFAULT 0,
			notes TEXT NOT NULL DEFAULT '',
			assessed_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS maintenance_holds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL REFERENCES chambers(id),
			reason TEXT NOT NULL,
			operator TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'planned',
			activated_at TEXT,
			lifted_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS pump_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL,
			pump_code TEXT NOT NULL,
			action TEXT NOT NULL,
			flow_cms REAL NOT NULL,
			at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ops_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chamber_id INTEGER NOT NULL,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			at TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
