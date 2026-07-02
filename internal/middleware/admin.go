package middleware

import (
	"net/http"

	"plugin-execution-system/internal/response"
)

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUserFromContext(r.Context())
		if !ok || user.Role != "admin" {
			response.Fail(w, RequestIDFromContext(r.Context()), response.NewAppError(response.CodeForbidden, "admin required"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
