package router

import (
	"net/http"
	"strings"

	"plugin-execution-system/internal/handler"
)

func RegisterWebhookRoutes(mux *http.ServeMux, h Handlers) {
	mux.HandleFunc("/api/webhooks", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if r.Method == "POST" {
			h.Webhook.CreateWebhook(w, r)
			return
		}
		if method(w, r, "GET") {
			h.Webhook.ListWebhooks(w, r)
		}
	})
	mux.HandleFunc("/api/webhooks/deliveries", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") && admin(w, r) {
			h.Webhook.ListDeliveries(w, r, "")
		}
	})
	mux.HandleFunc("/api/webhooks/", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		id := handler.WebhookIDFromPath(r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/enable") {
			if method(w, r, "POST") {
				h.Webhook.EnableWebhook(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/disable") {
			if method(w, r, "POST") {
				h.Webhook.DisableWebhook(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/deliveries") {
			if method(w, r, "GET") {
				h.Webhook.ListDeliveries(w, r, id)
			}
			return
		}
		if r.Method == "DELETE" {
			h.Webhook.DeleteWebhook(w, r, id)
			return
		}
		if method(w, r, "GET") {
			h.Webhook.GetWebhook(w, r, id)
		}
	})
}
