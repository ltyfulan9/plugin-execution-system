package router

import "net/http"

func RegisterStaticRoutes(mux *http.ServeMux, h Handlers) {
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, "GET") {
			http.ServeFile(w, r, "docs/openapi.json")
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/static/index.html")
	})
}
