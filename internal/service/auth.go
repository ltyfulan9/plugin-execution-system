package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
)

type AuthService struct{ repo repository.AuthStore }

func NewAuthService(repo repository.AuthStore) *AuthService { return &AuthService{repo: repo} }
func hashSecret(s string) string                            { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func (s *AuthService) Login(ctx context.Context, username, password, token string) (string, model.CurrentUser, error) {
	if token != "" {
		u, ok, err := s.repo.GetUserByTokenHash(hashSecret(token))
		if err != nil {
			return "", model.CurrentUser{}, err
		}
		if !ok {
			return "", model.CurrentUser{}, response.NewAppError(response.CodeUnauthorized, "invalid token")
		}
		return token, toCurrentUser(u), nil
	}
	u, ok, err := s.repo.GetUserByUsername(username)
	if err != nil {
		return "", model.CurrentUser{}, err
	}
	if !ok || u.PasswordHash != hashSecret(password) {
		return "", model.CurrentUser{}, response.NewAppError(response.CodeUnauthorized, "invalid username or password")
	}
	return "", toCurrentUser(u), nil
}

func (s *AuthService) ValidateToken(token string) (model.CurrentUser, error) {
	u, ok, err := s.repo.GetUserByTokenHash(hashSecret(token))
	if err != nil {
		return model.CurrentUser{}, err
	}
	if !ok {
		return model.CurrentUser{}, response.NewAppError(response.CodeUnauthorized, "invalid token")
	}
	return toCurrentUser(u), nil
}
func (s *AuthService) RequireAdmin(user model.CurrentUser) error {
	if !model.IsAdminRole(user.Role) {
		return response.NewAppError(response.CodeForbidden, "admin required")
	}
	return nil
}
func (s *AuthService) CanViewExecution(user model.CurrentUser, e model.Execution) bool {
	return model.IsSuperAdminRole(user.Role) || (model.IsAdminRole(user.Role) && model.SameScope(user.Scope(), e.Scope())) || (e.UserID == user.ID && model.SameScope(user.Scope(), e.Scope()))
}
func (s *AuthService) CanCancelExecution(user model.CurrentUser, e model.Execution) bool {
	return model.IsSuperAdminRole(user.Role) || (model.IsAdminRole(user.Role) && model.SameScope(user.Scope(), e.Scope())) || (e.UserID == user.ID && model.SameScope(user.Scope(), e.Scope()))
}
func (s *AuthService) CanManagePlugin(user model.CurrentUser) bool {
	return model.IsAdminRole(user.Role)
}
func (s *AuthService) CanViewAuditLog(user model.CurrentUser) bool {
	return model.IsAdminRole(user.Role)
}
func toCurrentUser(u model.User) model.CurrentUser {
	return model.CurrentUser{TenantID: u.TenantID, ProjectID: u.ProjectID, ID: u.ID, Username: u.Username, Role: u.Role}
}
