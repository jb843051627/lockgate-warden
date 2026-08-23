package api

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/metrics"
)

// statusRecorder 捕获响应码供访问日志使用。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware 访问日志、异常兜底与请求计数。
func Middleware(m *metrics.Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		defer func() {
			if rec2 := recover(); rec2 != nil {
				log.Printf("http panic %s: %v\n%s", r.URL.Path, rec2, debug.Stack())
				writeErr(rec, http.StatusInternalServerError, errInternal)
			}
			log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start))
		}()
		next.ServeHTTP(rec, r)
	})
}

var errInternal = &staticError{"internal error"}

type staticError struct{ msg string }

func (e *staticError) Error() string { return e.msg }
