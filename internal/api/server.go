// Package api 提供 HTTP/JSON 接口与静态页面托管。
package api

import (
	"encoding/json"
	"net/http"

	"github.com/jb843051627/lockgate-warden/internal/service"
)

// Server HTTP 服务聚合。
type Server struct {
	svc *service.Service
	mux *http.ServeMux
}

// NewServer 构造路由。
func NewServer(svc *service.Service) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回根处理器。
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/api/chambers", s.handleChambers)
	s.mux.HandleFunc("/api/gates", s.handleGates)
	s.mux.HandleFunc("/api/gates/command", s.handleGateCommand)
	s.mux.HandleFunc("/api/gates/enabled", s.handleGateEnabled)
	s.mux.HandleFunc("/api/vessels", s.handleVessels)
	s.mux.HandleFunc("/api/transits", s.handleTransits)
	s.mux.HandleFunc("/api/telemetry/batches", s.handleBatches)
	s.mux.HandleFunc("/api/telemetry/batch", s.handleBatchGet)
	s.mux.HandleFunc("/api/telemetry/checksum", s.handleChecksum)
	s.mux.HandleFunc("/api/alerts", s.handleAlerts)
	s.mux.HandleFunc("/api/alerts/ack", s.handleAck)
	s.mux.HandleFunc("/api/alerts/close", s.handleClose)
	s.mux.HandleFunc("/api/assessments/run", s.handleAssessRun)
	s.mux.HandleFunc("/api/kpi/weekly", s.handleKPI)
	s.mux.HandleFunc("/api/export/chamber.csv", s.handleExport)
	s.mux.HandleFunc("/api/stats/hourly", s.handleHourlyLoads)
	s.mux.HandleFunc("/api/stats/dwell", s.handleDwell)
	s.mux.HandleFunc("/api/stats/plan", s.handlePlanLockage)
	s.mux.HandleFunc("/api/stats/uptime", s.handleUptime)
	s.mux.HandleFunc("/api/pilots/roster", s.handleRoster)
	s.mux.HandleFunc("/api/pilots", s.handlePilotAdd)
	s.mux.HandleFunc("/metrics", s.handleMetrics)
	s.mux.Handle("/", http.FileServer(http.Dir("web/static")))
}

func writeErr(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func writeOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
