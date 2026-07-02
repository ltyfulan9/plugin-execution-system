package middleware

import (
	"context"
	"net/http"
	"strings"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/service"
)

const currentUserKey ctxKey = "current_user"

func Auth(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
				if user, err := auth.ValidateToken(token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), currentUserKey, user))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
func CurrentUserFromContext(ctx context.Context) (model.CurrentUser, bool) {
	u, ok := ctx.Value(currentUserKey).(model.CurrentUser)
	return u, ok
}
