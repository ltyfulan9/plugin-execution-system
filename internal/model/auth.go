package model

import "time"

type Role string

const (
	RoleUser       Role = "user"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

type User struct {
	TenantID     string    `json:"tenant_id"`
	ProjectID    string    `json:"project_id"`
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Role         Role      `json:"role"`
	TokenHash    string    `json:"token_hash"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CurrentUser struct {
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
	ID        string `json:"id"`
	Username  string `json:"username"`
	Role      Role   `json:"role"`
}

func IsValidRole(role Role) bool {
	return role == RoleUser || role == RoleAdmin || role == RoleSuperAdmin
}
func IsAdminRole(role Role) bool      { return role == RoleAdmin || role == RoleSuperAdmin }
func IsSuperAdminRole(role Role) bool { return role == RoleSuperAdmin }
func (u CurrentUser) Scope() ResourceScope {
	return ResourceScope{TenantID: u.TenantID, ProjectID: u.ProjectID}.Normalize()
}
