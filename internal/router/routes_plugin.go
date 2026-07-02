package router

import (
	"net/http"
	"strings"

	"plugin-execution-system/internal/handler"
)

func RegisterPluginRoutes(mux *http.ServeMux, h Handlers) {
	mux.HandleFunc("/api/plugins", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") {
			h.Plugin.ListPlugins(w, r)
		}
	})
	mux.HandleFunc("/api/plugins/reload", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "POST") && admin(w, r) {
			h.Plugin.ReloadPlugins(w, r)
		}
	})
	mux.HandleFunc("/api/plugins/", func(w http.ResponseWriter, r *http.Request) {
		id := handler.PluginIDFromPath(r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/enable") {
			if method(w, r, "POST") && admin(w, r) {
				h.Plugin.EnablePlugin(w, r, id)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/disable") {
			if method(w, r, "POST") && admin(w, r) {
				h.Plugin.DisablePlugin(w, r, id)
			}
			return
		}
		if method(w, r, "GET") {
			h.Plugin.GetPlugin(w, r, id)
		}
	})
}
