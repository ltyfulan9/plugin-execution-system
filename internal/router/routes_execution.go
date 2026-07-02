package router

import (
	"net/http"
	"strings"

	"plugin-execution-system/internal/handler"
)

func RegisterExecutionRoutes(mux *http.ServeMux, h Handlers) {
	mux.HandleFunc("/api/executions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h.Execution.CreateExecution(w, r)
			return
		}
		if method(w, r, "GET") {
			h.Execution.ListExecutions(w, r)
		}
	})
	mux.HandleFunc("/api/executions/", func(w http.ResponseWriter, r *http.Request) {
		id := handler.ExecutionIDFromPath(r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/cancel") {
			if method(w, r, "POST") {
				h.Execution.CancelExecution(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/results") {
			if method(w, r, "GET") {
				h.Result.GetExecutionResults(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/summary") {
			if method(w, r, "GET") {
				h.Result.GetExecutionSummary(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/events/stream") {
			if method(w, r, "GET") && h.Observe != nil {
				h.Observe.StreamEvents(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/events") {
			if method(w, r, "GET") && h.Observe != nil {
				h.Observe.ListEvents(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/attempts") {
			if method(w, r, "GET") && h.Observe != nil {
				h.Observe.ListAttempts(w, r, id)
			}
			return
		}
		if method(w, r, "GET") {
			h.Execution.GetExecution(w, r, id)
		}
	})
}
