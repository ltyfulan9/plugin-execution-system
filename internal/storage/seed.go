package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"plugin-execution-system/internal/model"
)

func SeedDefaultUsers(store *JSONStore, demoToken, adminToken string) error {
	var users []model.User
	if err := store.Load("users", &users); err != nil {
		return err
	}
	if len(users) > 0 {
		return nil
	}
	now := time.Now().UTC()
	users = []model.User{
		{TenantID: model.DefaultTenantID, ProjectID: model.DefaultProjectID, ID: "user_demo", Username: "demo", Role: model.RoleUser, TokenHash: HashSecret(demoToken), PasswordHash: HashSecret("demo123"), CreatedAt: now, UpdatedAt: now},
		{TenantID: model.DefaultTenantID, ProjectID: model.DefaultProjectID, ID: "user_admin", Username: "admin", Role: model.RoleAdmin, TokenHash: HashSecret(adminToken), PasswordHash: HashSecret("admin123"), CreatedAt: now, UpdatedAt: now},
	}
	return store.Save("users", users)
}

func HashSecret(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
