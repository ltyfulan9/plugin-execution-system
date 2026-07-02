package service

import (
	"time"

	"plugin-execution-system/internal/event"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
)

type ExecutionEventService struct {
	repo repository.ExecutionEventStore
	bus  *event.Bus
}

func NewExecutionEventService(repo repository.ExecutionEventStore, bus *event.Bus) *ExecutionEventService {
	return &ExecutionEventService{repo: repo, bus: bus}
}

func (s *ExecutionEventService) Record(executionID, pluginID string, eventType model.ExecutionEventType, status, message, requestID string, detail map[string]any) {
	if s == nil || s.repo == nil || executionID == "" {
		return
	}
	e := model.ExecutionEvent{TenantID: model.DefaultTenantID, ProjectID: model.DefaultProjectID, ID: newID("evt"), ExecutionID: executionID, PluginID: pluginID, Type: eventType, Status: status, Message: message, Detail: detail, RequestID: requestID, CreatedAt: time.Now().UTC()}
	_ = s.repo.Create(e)
	if s.bus != nil {
		s.bus.Publish(e)
	}
}

func (s *ExecutionEventService) ListByExecutionID(executionID string) ([]model.ExecutionEvent, error) {
	return s.repo.ListByExecutionID(executionID)
}

func (s *ExecutionEventService) Subscribe(executionID string, fn func(model.ExecutionEvent)) func() {
	if s == nil || s.bus == nil {
		return func() {}
	}
	return s.bus.Subscribe(executionID, fn)
}
