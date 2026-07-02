package middleware

import (
	"fmt"
	"net/http"

	"plugin-execution-system/internal/response"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				response.Fail(w, RequestIDFromContext(r.Context()), response.NewDetailedError(response.CodeInternalError, "internal error", fmt.Sprint(rec)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
