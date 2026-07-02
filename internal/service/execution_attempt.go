package service

import (
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
)

type ExecutionAttemptService struct {
	repo repository.ExecutionAttemptStore
}

func NewExecutionAttemptService(repo repository.ExecutionAttemptStore) *ExecutionAttemptService {
	return &ExecutionAttemptService{repo: repo}
}

func (s *ExecutionAttemptService) StartAttempt(executionID, workerID string) (model.ExecutionAttempt, error) {
	attemptNo, err := s.repo.NextAttemptNo(executionID)
	if err != nil {
		return model.ExecutionAttempt{}, err
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(30 * time.Second)
	attempt := model.ExecutionAttempt{TenantID: model.DefaultTenantID, ProjectID: model.DefaultProjectID, ID: newID("attempt"), ExecutionID: executionID, AttemptNo: attemptNo, WorkerID: workerID, LeaseID: newID("lease"), Status: model.ExecutionAttemptRunning, StartedAt: now, HeartbeatAt: &now, LeaseUntil: &leaseUntil}
	return attempt, s.repo.Create(attempt)
}

func (s *ExecutionAttemptService) Heartbeat(attempt model.ExecutionAttempt, extendBy time.Duration) (model.ExecutionAttempt, error) {
	now := time.Now().UTC()
	leaseUntil := now.Add(extendBy)
	attempt.HeartbeatAt = &now
	attempt.LeaseUntil = &leaseUntil
	return attempt, s.repo.Update(attempt)
}

func (s *ExecutionAttemptService) FinishAttempt(attempt model.ExecutionAttempt, status model.ExecutionAttemptStatus, errMsg string) error {
	now := time.Now().UTC()
	attempt.Status = status
	attempt.ErrorMessage = errMsg
	attempt.FinishedAt = &now
	return s.repo.Update(attempt)
}

func (s *ExecutionAttemptService) ListByExecutionID(executionID string) ([]model.ExecutionAttempt, error) {
	return s.repo.ListByExecutionID(executionID)
}
