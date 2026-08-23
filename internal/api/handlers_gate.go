package api

import (
	"fmt"
	"net/http"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

func (s *Server) handleGates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var g model.Gate
		if err := decode(r, &g); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		err := s.svc.CreateGate(&g)
		respond(w, g, err)
	case http.MethodGet:
		chamberID := queryInt(r, "chamber_id", 0)
		list, err := s.svc.ListGates(chamberID)
		respond(w, list, err)
	default:
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
	}
}

// handleGateCommand POST /api/gates/command {"id":1,"to":"opening"}
func (s *Server) handleGateCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ID int64            `json:"id"`
		To model.GateStatus `json:"to"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	gate, err := s.svc.CommandGate(body.ID, body.To)
	respond(w, gate, err)
}

// handleGateEnabled POST /api/gates/enabled {"id":1,"enabled":false}
func (s *Server) handleGateEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ID      int64 `json:"id"`
		Enabled bool  `json:"enabled"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if body.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id is required"))
		return
	}
	err := s.svc.SetGateEnabled(body.ID, body.Enabled)
	respond(w, nil, err)
}
