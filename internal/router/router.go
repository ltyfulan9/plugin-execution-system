package router

import (
	"expvar"
	"net/http"
	"strings"

	"plugin-execution-system/internal/handler"
	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
)

type Handlers struct {
	Auth      *handler.AuthHandler
	Plugin    *handler.PluginHandler
	Execution *handler.ExecutionHandler
	Result    *handler.ResultHandler
	Observe   *handler.ExecutionObserveHandler
	Audit     *handler.AuditHandler
	Health    *handler.HealthHandler
	Webhook   *handler.WebhookHandler
	AuthSvc   *service.AuthService
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	RegisterRoutes(mux, h)
	mux.Handle("/debug/vars", expvar.Handler())
	var root http.Handler = versionAlias(mux)
	root = middleware.Auth(h.AuthSvc)(root)
	root = middleware.Logging(root)
	root = middleware.Recovery(root)
	root = middleware.TraceContext(root)
	root = middleware.RequestID(root)
	return root
}

func RegisterRoutes(mux *http.ServeMux, h Handlers) {
	RegisterHealthRoutes(mux, h)
	RegisterAuthRoutes(mux, h)
	RegisterPluginRoutes(mux, h)
	RegisterExecutionRoutes(mux, h)
	RegisterResultRoutes(mux, h)
	RegisterAuditRoutes(mux, h)
	RegisterWebhookRoutes(mux, h)
	RegisterStaticRoutes(mux, h)
}

func method(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method != want {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeInvalidArgument, "method not allowed"))
		return false
	}
	return true
}
func admin(w http.ResponseWriter, r *http.Request) bool {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok || user.Role != "admin" {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeForbidden, "admin required"))
		return false
	}
	return true
}
func suffix(path, s string) bool { return strings.HasSuffix(strings.Trim(path, "/"), s) }

func versionAlias(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			clone := r.Clone(r.Context())
			clone.URL.Path = "/api/" + strings.TrimPrefix(r.URL.Path, "/api/v1/")
			next.ServeHTTP(w, clone)
			return
		}
		next.ServeHTTP(w, r)
	})
}
