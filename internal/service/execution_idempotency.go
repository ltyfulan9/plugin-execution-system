package service

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/validator"
)

type IdempotencyService struct {
	repo repository.ExecutionStore
}

func NewIdempotencyService(repo repository.ExecutionStore) *IdempotencyService {
	return &IdempotencyService{repo: repo}
}
func (s *IdempotencyService) BuildInputHash(input map[string]any) (string, error) {
	return validator.HashJSON(input)
}
func (s *IdempotencyService) BuildPluginIDsHash(ids []string) string {
	return validator.HashStrings(ids)
}
func (s *IdempotencyService) FindExistingExecution(userID, key string) (model.Execution, bool, error) {
	return s.repo.FindByIdempotencyKey(userID, key)
}
func (s *IdempotencyService) CheckConflict(existing model.Execution, inputHash, pluginHash string) error {
	if existing.InputHash != inputHash || existing.PluginIDsHash != pluginHash {
		return response.NewAppError(response.CodeIdempotencyConflict, "same idempotency key with different request")
	}
	return nil
}
