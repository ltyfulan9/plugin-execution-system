package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const traceIDKey ctxKey = "trace_id"

// TraceContext provides an OpenTelemetry-compatible trace_id propagation baseline
// without taking a hard dependency on the OTel SDK. The platform contract is that
// trace_id must flow API -> scheduler -> worker -> runner -> result/audit.
func TraceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := TraceIDFromHeaders(r)
		if traceID == "" {
			traceID = newHexID(16)
		}
		w.Header().Set("Trace-ID", traceID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), traceIDKey, traceID)))
	})
}

func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

func TraceIDFromHeaders(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("Trace-ID")); isSafeTraceID(v) {
		return v
	}
	// W3C traceparent: version-traceid-spanid-flags.
	parts := strings.Split(strings.TrimSpace(r.Header.Get("traceparent")), "-")
	if len(parts) >= 4 && isSafeTraceID(parts[1]) {
		return parts[1]
	}
	return ""
}

func isSafeTraceID(v string) bool {
	if len(v) < 16 || len(v) > 64 {
		return false
	}
	for _, r := range v {
		if !((r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func newHexID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
