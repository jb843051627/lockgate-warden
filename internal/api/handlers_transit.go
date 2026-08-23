package api

import (
	"net/http"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

func (s *Server) handleVessels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var v model.Vessel
	if err := decode(r, &v); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	err := s.svc.UpsertVesselEntry(&v)
	respond(w, v, err)
}

// handleTransitBegin POST /api/transits/begin {"mmsi":"123456789","chamber_id":1,"direction":"up"}
func (s *Server) handleTransitBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		MMSI      string `json:"mmsi"`
		ChamberID int64  `json:"chamber_id"`
		Direction string `json:"direction"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	t, err := s.svc.BeginTransit(body.MMSI, body.ChamberID, body.Direction)
	respond(w, t, err)
}

// handleTransitComplete POST /api/transits/complete {"id":1}
func (s *Server) handleTransitComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ID int64 `json:"id"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	t, err := s.svc.CompleteTransit(body.ID)
	respond(w, t, err)
}

func (s *Server) handleTransits(w http.ResponseWriter, r *http.Request) {
	writeErr(w, http.StatusNotFound, &staticError{"use /api/transits/begin or /api/transits/complete"})
}
