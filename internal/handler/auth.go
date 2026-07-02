package handler

import (
	"encoding/json"
	"net/http"

	"plugin-execution-system/internal/middleware"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
	"plugin-execution-system/internal/validator"
)

type AuthHandler struct{ auth *service.AuthService }

func NewAuthHandler(auth *service.AuthService) *AuthHandler { return &AuthHandler{auth: auth} }
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req validator.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validator.ValidateLoginRequest(req) {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeInvalidArgument, "invalid login request"))
		return
	}
	token, user, err := h.auth.Login(r.Context(), req.Username, req.Password, req.Token)
	if err != nil {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), map[string]any{"token": token, "user": user})
}
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUserFromContext(r.Context())
	if !ok {
		response.Fail(w, middleware.RequestIDFromContext(r.Context()), response.NewAppError(response.CodeUnauthorized, "login required"))
		return
	}
	response.Success(w, middleware.RequestIDFromContext(r.Context()), user)
}
