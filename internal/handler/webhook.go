package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
)

type WebhookHandler struct{ webhooks *service.WebhookService }

func NewWebhookHandler(webhooks *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhooks: webhooks}
}

func currentWebhookUser(w http.ResponseWriter, r *http.Request) (model.CurrentUser, bool) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return model.CurrentUser{}, false
	}
	return user, true
}

type createWebhookRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events,omitempty"`
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeInvalidArgument, "invalid webhook request"))
		return
	}
	endpoint, secret, err := h.webhooks.CreateEndpoint(user, service.CreateWebhookInput{Name: req.Name, URL: req.URL, Secret: req.Secret, Events: req.Events})
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]any{"webhook": redactWebhook(endpoint), "secret": secret})
}

func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	user, ok := currentWebhookUser(w, r)
	if !ok {
		return
	}
	items, err := h.webhooks.ListEndpointsForUser(user)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), redactWebhooks(items))
}

func (h *WebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := currentWebhookUser(w, r)
	if !ok {
		return
	}
	item, err := h.webhooks.GetEndpointForUser(user, id)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), redactWebhook(item))
}

func (h *WebhookHandler) EnableWebhook(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := currentWebhookUser(w, r)
	if !ok {
		return
	}
	item, err := h.webhooks.SetEndpointStatusForUser(user, id, model.WebhookStatusEnabled)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), redactWebhook(item))
}

func (h *WebhookHandler) DisableWebhook(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := currentWebhookUser(w, r)
	if !ok {
		return
	}
	item, err := h.webhooks.SetEndpointStatusForUser(user, id, model.WebhookStatusDisabled)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), redactWebhook(item))
}

func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := currentWebhookUser(w, r)
	if !ok {
		return
	}
	if err := h.webhooks.DeleteEndpointForUser(user, id); err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]bool{"deleted": true})
}

func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request, webhookID string) {
	user, ok := currentWebhookUser(w, r)
	if !ok {
		return
	}
	items, err := h.webhooks.ListDeliveriesForUser(user, webhookID)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), items)
}

func redactWebhook(endpoint model.WebhookEndpoint) model.WebhookEndpoint {
	endpoint.Secret = ""
	return endpoint
}

func redactWebhooks(items []model.WebhookEndpoint) []model.WebhookEndpoint {
	out := make([]model.WebhookEndpoint, 0, len(items))
	for _, item := range items {
		out = append(out, redactWebhook(item))
	}
	return out
}

func WebhookIDFromPath(path string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "webhooks" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
