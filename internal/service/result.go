package service

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
)

type ResultService struct{ repo repository.ResultStore }

func NewResultService(repo repository.ResultStore) *ResultService {
	return &ResultService{repo: repo}
}
func (s *ResultService) SavePluginResult(r model.ExecutionResult) error { return s.repo.Create(r) }
func (s *ResultService) BatchSaveResults(rs []model.ExecutionResult) error {
	return s.repo.BatchCreate(rs)
}
func (s *ResultService) GetExecutionResults(executionID string) ([]model.ExecutionResult, error) {
	return s.repo.GetByExecutionID(executionID)
}
func (s *ResultService) BuildExecutionSummary(e model.Execution, results []model.ExecutionResult) model.ExecutionSummary {
	var ok, failed, timeout int
	var dur int64
	for _, r := range results {
		dur += r.DurationMS
		switch r.Status {
		case model.PluginResultSuccess:
			ok++
		case model.PluginResultTimeout:
			timeout++
		default:
			failed++
		}
	}
	return model.ExecutionSummary{ExecutionID: e.ID, Status: e.Status, Total: len(results), Success: ok, Failed: failed, Timeout: timeout, DurationMS: dur}
}
func (s *ResultService) AggregateExecutionStatus(results []model.ExecutionResult) model.ExecutionStatus {
	if len(results) == 0 {
		return model.ExecutionStatusFailed
	}
	success := 0
	timeout := 0
	for _, r := range results {
		if r.Status == model.PluginResultSuccess {
			success++
		}
		if r.Status == model.PluginResultTimeout {
			timeout++
		}
	}
	if success == len(results) {
		return model.ExecutionStatusSuccess
	}
	if success > 0 {
		return model.ExecutionStatusPartialSuccess
	}
	if timeout == len(results) {
		return model.ExecutionStatusTimeout
	}
	return model.ExecutionStatusFailed
}
