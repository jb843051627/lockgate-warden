package api

import (
	"net/http"
)

var errMethod = &staticError{"method not allowed"}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{"status": "ok", "service": "lockgate-warden"})
}

// handleMetrics 文本格式指标页。
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write([]byte("# lockgate metrics exposed via /api endpoints\n"))
}

func (s *Server) handleChambers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.svc.ListChambers()
		respond(w, list, err)
	default:
		writeErr(w, http.StatusMethodNotAllowed, errMethod)
	}
}
