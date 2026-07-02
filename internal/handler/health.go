package handler

import (
	"net/http"
	"os"
	"time"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/observability"
	"plugin-execution-system/internal/response"
)

type HealthHandler struct{ pluginDir string }

func NewHealthHandler(pluginDir string) *HealthHandler                   { return &HealthHandler{pluginDir: pluginDir} }
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request)   { h.Livez(w, r) }
func (h *HealthHandler) HealthV1(w http.ResponseWriter, r *http.Request) { h.Livez(w, r) }
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request)    { h.Readyz(w, r) }

func (h *HealthHandler) Livez(w http.ResponseWriter, r *http.Request) {
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]any{"status": "ok", "time": time.Now().UTC()})
}
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	_, err := os.Stat(h.pluginDir)
	ready := err == nil
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]any{"ready": ready, "plugin_dir": h.pluginDir, "checks": map[string]any{"plugin_dir": ready}})
}
func (h *HealthHandler) Workerz(w http.ResponseWriter, r *http.Request) {
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]any{"ready": true, "lease_model": "durable-queue-contract", "heartbeat_required": true})
}
func (h *HealthHandler) Dependencyz(w http.ResponseWriter, r *http.Request) {
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]any{"metadata_store": "postgres-production-contract", "object_storage": "artifact-contract", "policy_engine": "opa-compatible-contract"})
}

func (h *HealthHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(observability.PrometheusText()))
}
