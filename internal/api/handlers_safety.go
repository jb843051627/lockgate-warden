package api

import (
	"net/http"
)

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	status := queryString(r, "status")
	limit := queryInt(r, "limit", 100)
	list, err := s.svc.ListAlerts(status, int(limit))
	respond(w, list, err)
}

func (s *Server) handleAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ID int64  `json:"id"`
		By string `json:"by"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	alert, err := s.svc.AckAlert(body.ID, body.By)
	respond(w, alert, err)
}

func (s *Server) handleClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		ID   int64  `json:"id"`
		Note string `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	alert, err := s.svc.CloseAlert(body.ID, body.Note)
	respond(w, alert, err)
}
