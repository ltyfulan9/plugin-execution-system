package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
)

type ExecutionObserveHandler struct {
	executions ExecutionReader
	events     *service.ExecutionEventService
	attempts   *service.ExecutionAttemptService
}

func NewExecutionObserveHandler(e ExecutionReader, events *service.ExecutionEventService, attempts *service.ExecutionAttemptService) *ExecutionObserveHandler {
	return &ExecutionObserveHandler{executions: e, events: events, attempts: attempts}
}

func (h *ExecutionObserveHandler) ListEvents(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	if _, err := h.executions.GetExecution(user, id); err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	items, err := h.events.ListByExecutionID(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), items)
}

func (h *ExecutionObserveHandler) ListAttempts(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	if _, err := h.executions.GetExecution(user, id); err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	items, err := h.attempts.ListByExecutionID(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), items)
}

func (h *ExecutionObserveHandler) StreamEvents(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	if _, err := h.executions.GetExecution(user, id); err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeInternalError, "streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	past, _ := h.events.ListByExecutionID(id)
	for _, evt := range past {
		writeSSE(w, evt)
	}
	flusher.Flush()

	ch := make(chan model.ExecutionEvent, 16)
	unsubscribe := h.events.Subscribe(id, func(evt model.ExecutionEvent) {
		select {
		case ch <- evt:
		default:
		}
	})
	defer unsubscribe()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-ch:
			writeSSE(w, evt)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, evt model.ExecutionEvent) {
	b, _ := json.Marshal(evt)
	_, _ = fmt.Fprintf(w, "event: %s\n", evt.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
}
