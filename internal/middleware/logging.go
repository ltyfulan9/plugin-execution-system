package middleware

import (
	"net/http"
	"time"

	"plugin-execution-system/internal/logging"
	"plugin-execution-system/internal/observability"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: 200}
		start := time.Now()
		next.ServeHTTP(sw, r)
		latency := time.Since(start)
		observability.IncHTTP(r.Method, r.URL.Path, sw.status)
		logging.Info("http_request", logging.Fields{
			"request_id":  RequestIDFromContext(r.Context()),
			"trace_id":    TraceIDFromContext(r.Context()),
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      sw.status,
			"latency_ms":  latency.Milliseconds(),
			"remote_addr": r.RemoteAddr,
		})
	})
}
