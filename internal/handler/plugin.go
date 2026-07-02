package handler

import (
	"net/http"
	"strings"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
)

type PluginHandler struct {
	plugin    *service.PluginService
	registry  *service.RegistryService
	pluginDir string
	audit     *service.AuditService
}

func NewPluginHandler(p *service.PluginService, r *service.RegistryService, pluginDir string, audit *service.AuditService) *PluginHandler {
	return &PluginHandler{plugin: p, registry: r, pluginDir: pluginDir, audit: audit}
}
func (h *PluginHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	items, err := h.plugin.ListPlugins()
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
func (h *PluginHandler) GetPlugin(w http.ResponseWriter, r *http.Request, id string) {
	p, err := h.plugin.GetPlugin(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), p)
}
func (h *PluginHandler) ReloadPlugins(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUserFromContext(r.Context())
	res, err := h.registry.ReloadPlugins(h.pluginDir)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.audit.Record(user.ID, model.AuditPluginLoaded, model.AuditResourceSystem, "registry", middleware.RequestIDFromContext(r.Context()), "plugins reloaded", map[string]any{"created": res.Created, "updated": res.Updated, "removed": res.Removed})
	response.Success(w, middleware.RequestIDFromContext(r.Context()), res)
}
func (h *PluginHandler) EnablePlugin(w http.ResponseWriter, r *http.Request, id string) {
	user, _ := middleware.CurrentUserFromContext(r.Context())
	p, err := h.plugin.EnablePlugin(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.audit.Record(user.ID, model.AuditPluginStatusChanged, model.AuditResourcePlugin, id, middleware.RequestIDFromContext(r.Context()), "plugin enabled", nil)
	response.Success(w, middleware.RequestIDFromContext(r.Context()), p)
}
func (h *PluginHandler) DisablePlugin(w http.ResponseWriter, r *http.Request, id string) {
	user, _ := middleware.CurrentUserFromContext(r.Context())
	p, err := h.plugin.DisablePlugin(id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.audit.Record(user.ID, model.AuditPluginStatusChanged, model.AuditResourcePlugin, id, middleware.RequestIDFromContext(r.Context()), "plugin disabled", nil)
	response.Success(w, middleware.RequestIDFromContext(r.Context()), p)
}
func PluginIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
