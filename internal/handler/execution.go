package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"context"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/validator"
)

type ExecutionUseCase interface {
	CreateExecution(ctx context.Context, user model.CurrentUser, pluginIDs []string, input map[string]any, idemKey, requestID string) (model.Execution, error)
	ListExecutions(user model.CurrentUser) ([]model.Execution, error)
	GetExecution(user model.CurrentUser, id string) (model.Execution, error)
	CancelExecution(user model.CurrentUser, id, requestID string) (model.Execution, error)
}

type ExecutionHandler struct{ executions ExecutionUseCase }

func NewExecutionHandler(e ExecutionUseCase) *ExecutionHandler {
	return &ExecutionHandler{executions: e}
}
func (h *ExecutionHandler) CreateExecution(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	var req validator.CreateExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validator.ValidateCreateExecutionRequest(req) {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeInvalidArgument, "invalid execution request"))
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	e, err := h.executions.CreateExecution(r.Context(), user, req.PluginIDs, req.Input, idem, middleware.RequestIDFromContext(r.Context()))
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Created(w, middleware.RequestIDFromContext(r.Context()), e)
}
func (h *ExecutionHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	items, err := h.executions.ListExecutions(user)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if r.URL.Query().Get("page") != "" || r.URL.Query().Get("page_size") != "" {
		response.Page(w, middleware.RequestIDFromContext(r.Context()), PageItems(items, PageQueryFromRequest(r)))
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), items)
}
func (h *ExecutionHandler) GetExecution(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	e, err := h.executions.GetExecution(user, id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), e)
}
func (h *ExecutionHandler) CancelExecution(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	e, err := h.executions.CancelExecution(user, id, middleware.RequestIDFromContext(r.Context()))
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), e)
}
func ExecutionIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
