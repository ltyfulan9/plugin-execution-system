package service

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
)

type ExecutionStateMachine struct{}

func NewExecutionStateMachine() *ExecutionStateMachine { return &ExecutionStateMachine{} }
func (m *ExecutionStateMachine) CanTransit(from, to model.ExecutionStatus) bool {
	allowed := map[model.ExecutionStatus][]model.ExecutionStatus{
		model.ExecutionStatusPending: {model.ExecutionStatusQueued, model.ExecutionStatusCanceled, model.ExecutionStatusFailed},
		model.ExecutionStatusQueued:  {model.ExecutionStatusRunning, model.ExecutionStatusCanceled, model.ExecutionStatusFailed},
		model.ExecutionStatusRunning: {model.ExecutionStatusSuccess, model.ExecutionStatusPartialSuccess, model.ExecutionStatusFailed, model.ExecutionStatusTimeout, model.ExecutionStatusCanceled},
	}
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}
func (m *ExecutionStateMachine) ValidateTransition(from, to model.ExecutionStatus) error {
	if !m.CanTransit(from, to) {
		return response.NewAppError(response.CodeExecutionStateInvalid, "invalid execution state transition")
	}
	return nil
}
func (m *ExecutionStateMachine) CanCancel(s model.ExecutionStatus) bool {
	return s == model.ExecutionStatusPending || s == model.ExecutionStatusQueued || s == model.ExecutionStatusRunning
}
func (m *ExecutionStateMachine) CanRun(s model.ExecutionStatus) bool {
	return s == model.ExecutionStatusQueued
}
func (m *ExecutionStateMachine) IsFinal(s model.ExecutionStatus) bool {
	return model.IsFinalExecutionStatus(s)
}
