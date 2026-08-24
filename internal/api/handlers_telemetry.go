package api

import (
	"net/http"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

func (s *Server) handleBatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var in model.BatchInput
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := s.svc.IngestBatch(in)
	respond(w, res, err)
}

// handleChecksum POST /api/telemetry/checksum {"points":[...]}
func (s *Server) handleChecksum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
		return
	}
	var body struct {
		Points []model.TelemetryPointInput `json:"points"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeOK(w, s.svc.RecomputeChecksum(body.Points))
}

// handleBatchGet GET /api/telemetry/batches?id=1
func (s *Server) handleBatchGet(w http.ResponseWriter, r *http.Request) {
	id := queryInt(r, "id", 0)
	batch, err := s.svc.GetBatch(id)
	respond(w, batch, err)
}
