package router

import "net/http"

func RegisterAuthRoutes(mux *http.ServeMux, h Handlers) {
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "POST") {
			h.Auth.Login(w, r)
		}
	})
	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") {
			h.Auth.Me(w, r)
		}
	})
}
