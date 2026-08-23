package api

import (
	"net/http"
)

// handleRoster GET /api/pilots/roster
func (s *Server) handleRoster(w http.ResponseWriter, r *http.Request) {
	list, err := s.svc.OnDutyRosterSafe()
	respond(w, list, err)
}

// handlePilotAdd POST /api/pilots {"name":"..","license":"A","hazmat_ok":true}
func (s *Server) handlePilotAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		Name     string `json:"name"`
		License  string `json:"license"`
		HazmatOK bool   `json:"hazmat_ok"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	p, err := s.svc.AddPilotEntry(body.Name, body.License, body.HazmatOK)
	respond(w, p, err)
}
