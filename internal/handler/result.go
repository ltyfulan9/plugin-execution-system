package handler

import (
	"net/http"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
)

type ResultHandler struct {
	results    *service.ResultService
	executions ExecutionReader
}

type ExecutionReader interface {
	GetExecution(user model.CurrentUser, id string) (model.Execution, error)
}

func NewResultHandler(rs *service.ResultService, e ExecutionReader) *ResultHandler {
	return &ResultHandler{results: rs, executions: e}
}
func (h *ResultHandler) GetExecutionResults(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	if _, err := h.executions.GetExecution(user, id); err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	items, err := h.results.GetExecutionResults(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), items)
}
func (h *ResultHandler) GetExecutionSummary(w http.ResponseWriter, r *http.Request, id string) {
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
	items, err := h.results.GetExecutionResults(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), h.results.BuildExecutionSummary(e, items))
}
