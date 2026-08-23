package api

import (
	"net/http"
)

// handleAssessRun POST /api/assessments/run {"chamber_id":1}
func (s *Server) handleAssessRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ChamberID int64 `json:"chamber_id"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	a, err := s.svc.RunAssessment(body.ChamberID)
	respond(w, a, err)
}

// handleKPI GET /api/kpi/weekly?chamber_id=1&days=7
func (s *Server) handleKPI(w http.ResponseWriter, r *http.Request) {
	chamberID := queryInt(r, "chamber_id", 0)
	days := queryInt(r, "days", 7)
	kpi, err := s.svc.LineWeeklyKPI(chamberID, int(days))
	respond(w, kpi, err)
}

// handleExport GET /api/export/chamber.csv?chamber_id=1&limit=5000
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	chamberID := queryInt(r, "chamber_id", 0)
	limit := queryInt(r, "limit", 5000)
	csv, err := s.svc.ExportChamberCSV(chamberID, int(limit))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Write([]byte(csv))
}
