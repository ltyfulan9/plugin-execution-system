package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type AuthRepository struct{ store *storage.JSONStore }

func NewAuthRepository(store *storage.JSONStore) *AuthRepository {
	return &AuthRepository{store: store}
}

func (r *AuthRepository) all() ([]model.User, error) {
	var users []model.User
	err := r.store.Load("users", &users)
	return users, err
}
func (r *AuthRepository) save(users []model.User) error { return r.store.Save("users", users) }
func (r *AuthRepository) CreateUser(u model.User) error {
	users, err := r.all()
	if err != nil {
		return err
	}
	users = append(users, u)
	return r.save(users)
}
func (r *AuthRepository) GetUserByID(id string) (model.User, bool, error) {
	users, err := r.all()
	if err != nil {
		return model.User{}, false, err
	}
	for _, u := range users {
		if u.ID == id {
			return u, true, nil
		}
	}
	return model.User{}, false, nil
}
func (r *AuthRepository) GetUserByTokenHash(hash string) (model.User, bool, error) {
	users, err := r.all()
	if err != nil {
		return model.User{}, false, err
	}
	for _, u := range users {
		if u.TokenHash == hash {
			return u, true, nil
		}
	}
	return model.User{}, false, nil
}
func (r *AuthRepository) GetUserByUsername(username string) (model.User, bool, error) {
	users, err := r.all()
	if err != nil {
		return model.User{}, false, err
	}
	for _, u := range users {
		if u.Username == username {
			return u, true, nil
		}
	}
	return model.User{}, false, nil
}
func (r *AuthRepository) ListUsers() ([]model.User, error) { return r.all() }
