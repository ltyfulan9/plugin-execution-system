package router

import "net/http"

func RegisterHealthRoutes(mux *http.ServeMux, h Handlers) {
	for _, path := range []string{"/api/health", "/api/v1/health", "/livez"} {
		p := path
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			if method(w, r, "GET") {
				h.Health.Livez(w, r)
			}
		})
	}
	for _, path := range []string{"/api/ready", "/api/v1/ready", "/readyz"} {
		p := path
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			if method(w, r, "GET") {
				h.Health.Readyz(w, r)
			}
		})
	}
	mux.HandleFunc("/workerz", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") {
			h.Health.Workerz(w, r)
		}
	})
	mux.HandleFunc("/dependencyz", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") {
			h.Health.Dependencyz(w, r)
		}
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") {
			h.Health.Metrics(w, r)
		}
	})
}
