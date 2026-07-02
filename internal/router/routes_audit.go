package router

import (
	"net/http"
	"strings"

	"plugin-execution-system/internal/handler"
)

func RegisterAuditRoutes(mux *http.ServeMux, h Handlers) {
	mux.HandleFunc("/api/audit/logs", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") && admin(w, r) {
			h.Audit.ListAuditLogs(w, r)
		}
	})
	mux.HandleFunc("/api/audit/executions/", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") && admin(w, r) {
			id := handler.ExecutionIDFromPath(strings.Replace(r.URL.Path, "/api/audit", "/api", 1))
			h.Audit.ListExecutionLogs(w, r, id)
		}
	})
	mux.HandleFunc("/api/audit/plugins/", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") && admin(w, r) {
			id := handler.PluginIDFromPath(strings.Replace(r.URL.Path, "/api/audit", "/api", 1))
			h.Audit.ListPluginLogs(w, r, id)
		}
	})
}
