package api

import (
	"net/http"
)

// handleHourlyLoads GET /api/stats/hourly?chamber_id=1&hours=24
func (s *Server) handleHourlyLoads(w http.ResponseWriter, r *http.Request) {
	chamberID := queryInt(r, "chamber_id", 0)
	hours := queryInt(r, "hours", 24)
	rows, err := s.svc.HourlyLoads(chamberID, int(hours))
	respond(w, rows, err)
}

// handleDwell GET /api/stats/dwell?chamber_id=1&days=7
func (s *Server) handleDwell(w http.ResponseWriter, r *http.Request) {
	chamberID := queryInt(r, "chamber_id", 0)
	days := queryInt(r, "days", 7)
	rep, err := s.svc.DwellReport(chamberID, int(days))
	respond(w, rep, err)
}

// handlePlanLockage POST /api/stats/plan {"mmsi":"...","chamber_id":1}
func (s *Server) handlePlanLockage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		MMSI      string `json:"mmsi"`
		ChamberID int64  `json:"chamber_id"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	plan, err := s.svc.PlanLockage(body.MMSI, body.ChamberID)
	respond(w, plan, err)
}

// handleUptime GET /api/stats/uptime?sensor_id=1&days=7
func (s *Server) handleUptime(w http.ResponseWriter, r *http.Request) {
	sensorID := queryInt(r, "sensor_id", 0)
	days := queryInt(r, "days", 7)
	ratio, err := s.svc.SensorUptimeReport(sensorID, int(days))
	respond(w, map[string]float64{"uptime": ratio}, err)
}
