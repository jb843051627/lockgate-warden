package api

import (
	"net/http"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// handleHoldCreate POST /api/holds {"chamber_id":1,"reason":"...","operator":"..."}
func (s *Server) handleHoldCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var h model.MaintenanceHold
	if err := decode(r, &h); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	err := s.svc.CreateHold(&h)
	respond(w, h, err)
}

// handlePump POST /api/pumps {"chamber_id":1,"pump_code":"P1","action":"start","flow_cms":12.5}
func (s *Server) handlePump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ChamberID int64   `json:"chamber_id"`
		PumpCode  string  `json:"pump_code"`
		Action    string  `json:"action"`
		FlowCMS   float64 `json:"flow_cms"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	e, err := s.svc.PumpControl(body.ChamberID, body.PumpCode, body.Action, body.FlowCMS)
	respond(w, e, err)
}
