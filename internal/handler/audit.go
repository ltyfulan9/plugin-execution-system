package handler

import (
	"net/http"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
)

type AuditHandler struct{ audit *service.AuditService }

func NewAuditHandler(a *service.AuditService) *AuditHandler { return &AuditHandler{audit: a} }
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.audit.ListAuditLogs()
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if r.URL.Query().Get("page") != "" || r.URL.Query().Get("page_size") != "" {
		response.Page(w, middleware.RequestIDFromContext(r.Context()), PageItems(logs, PageQueryFromRequest(r)))
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), logs)
}
func (h *AuditHandler) ListExecutionLogs(w http.ResponseWriter, r *http.Request, id string) {
	logs, err := h.audit.ListByResource(model.AuditResourceExecution, id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), logs)
}
func (h *AuditHandler) ListPluginLogs(w http.ResponseWriter, r *http.Request, id string) {
	logs, err := h.audit.ListByResource(model.AuditResourcePlugin, id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), logs)
}
