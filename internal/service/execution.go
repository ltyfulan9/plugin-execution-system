package service

import (
	"context"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/observability"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
)

type QueueSubmitter interface {
	Submit(ctx context.Context, executionID string) error
}

type ExecutionService struct {
	repo      repository.ExecutionStore
	pluginSvc *PluginService
	idem      *IdempotencyService
	sm        *ExecutionStateMachine
	audit     *AuditService
	queue     QueueSubmitter
	events    *ExecutionEventService
}

func NewExecutionService(repo repository.ExecutionStore, pluginSvc *PluginService, idem *IdempotencyService, sm *ExecutionStateMachine, audit *AuditService, queue QueueSubmitter) *ExecutionService {
	return &ExecutionService{repo: repo, pluginSvc: pluginSvc, idem: idem, sm: sm, audit: audit, queue: queue}
}

func (s *ExecutionService) WithEventService(events *ExecutionEventService) *ExecutionService {
	s.events = events
	return s
}

func (s *ExecutionService) CreateExecution(ctx context.Context, user model.CurrentUser, pluginIDs []string, input map[string]any, idemKey, requestID string) (model.Execution, error) {
	if len(pluginIDs) == 0 || input == nil {
		return model.Execution{}, response.NewAppError(response.CodeInvalidArgument, "plugin_ids and input are required")
	}
	if _, err := s.pluginSvc.ValidatePluginsExecutable(pluginIDs); err != nil {
		return model.Execution{}, err
	}
	inputHash, err := s.idem.BuildInputHash(input)
	if err != nil {
		return model.Execution{}, err
	}
	pluginHash := s.idem.BuildPluginIDsHash(pluginIDs)
	now := time.Now().UTC()
	scope := user.Scope()
	e := model.Execution{TenantID: scope.TenantID, ProjectID: scope.ProjectID, ID: newID("exec"), UserID: user.ID, PluginIDs: pluginIDs, InputJSON: input, InputHash: inputHash, PluginIDsHash: pluginHash, IdempotencyKey: idemKey, Status: model.ExecutionStatusPending, CreatedAt: now}
	createdOrExisting, created, err := s.repo.CreateWithIdempotency(e)
	if err != nil {
		return model.Execution{}, err
	}
	if !created {
		if err := s.idem.CheckConflict(createdOrExisting, inputHash, pluginHash); err != nil {
			observability.IncIdempotencyConflict()
			return model.Execution{}, err
		}
		observability.IncIdempotencyHit()
		return createdOrExisting, nil
	}
	if s.audit != nil {
		if err := s.audit.RecordScoped(e.Scope(), user.ID, model.AuditExecutionCreated, model.AuditResourceExecution, e.ID, requestID, "", model.AuditDecisionAllow, "", "execution created", map[string]any{"input_hash": e.InputHash}); err != nil {
			return model.Execution{}, err
		}
	}
	if s.events != nil {
		s.events.Record(e.ID, "", model.ExecutionEventCreated, string(e.Status), "execution created", requestID, nil)
	}
	if s.queue != nil {
		// Mark the task as Queued before publishing it to the in-memory queue.
		// Otherwise a fast worker can consume the ID while the task is still Pending
		// and incorrectly reject Pending -> Running.
		if err := s.MarkQueued(e.ID, requestID); err != nil {
			return model.Execution{}, err
		}
		if err := s.queue.Submit(ctx, e.ID); err != nil {
			observability.IncQueueFull()
			_ = s.repo.UpdateStatus(e.ID, model.ExecutionStatusFailed, err.Error())
			return model.Execution{}, response.NewAppError(response.CodeQueueFull, "execution queue is full")
		}
		observability.IncTaskSubmitted()
	}
	return s.GetExecution(user, e.ID)
}

func (s *ExecutionService) GetExecution(user model.CurrentUser, id string) (model.Execution, error) {
	e, ok, err := s.repo.GetByID(id)
	if err != nil {
		return model.Execution{}, err
	}
	if !ok {
		return model.Execution{}, response.NewAppError(response.CodeExecutionNotFound, "execution not found")
	}
	if !model.IsSuperAdminRole(user.Role) && !model.SameScope(user.Scope(), e.Scope()) {
		return model.Execution{}, response.NewAppError(response.CodeForbidden, "cannot view execution")
	}
	if !model.IsAdminRole(user.Role) && e.UserID != user.ID {
		return model.Execution{}, response.NewAppError(response.CodeForbidden, "cannot view execution")
	}
	return e, nil
}
func (s *ExecutionService) GetExecutionInternal(id string) (model.Execution, error) {
	e, ok, err := s.repo.GetByID(id)
	if err != nil {
		return model.Execution{}, err
	}
	if !ok {
		return model.Execution{}, response.NewAppError(response.CodeExecutionNotFound, "execution not found")
	}
	return e, nil
}
func (s *ExecutionService) ListExecutions(user model.CurrentUser) ([]model.Execution, error) {
	if model.IsSuperAdminRole(user.Role) {
		return s.repo.ListAll()
	}
	if model.IsAdminRole(user.Role) {
		return s.repo.ListByScope(user.Scope())
	}
	return s.repo.ListByUserID(user.ID)
}
func (s *ExecutionService) CancelExecution(user model.CurrentUser, id, requestID string) (model.Execution, error) {
	e, err := s.GetExecution(user, id)
	if err != nil {
		return model.Execution{}, err
	}
	if !s.sm.CanCancel(e.Status) {
		return model.Execution{}, response.NewAppError(response.CodeExecutionStateInvalid, "execution cannot be canceled")
	}
	if err := s.repo.UpdateStatus(id, model.ExecutionStatusCanceled, "canceled"); err != nil {
		return model.Execution{}, err
	}
	if s.audit != nil {
		if err := s.audit.RecordScoped(e.Scope(), user.ID, model.AuditExecutionCanceled, model.AuditResourceExecution, id, requestID, "", model.AuditDecisionAllow, "", "execution canceled", nil); err != nil {
			return model.Execution{}, err
		}
	}
	if s.events != nil {
		s.events.Record(id, "", model.ExecutionEventCanceled, string(model.ExecutionStatusCanceled), "execution canceled", requestID, nil)
	}
	return s.GetExecution(user, id)
}
func (s *ExecutionService) MarkQueued(id, requestID string) error {
	e, err := s.GetExecutionInternal(id)
	if err != nil {
		return err
	}
	if err := s.sm.ValidateTransition(e.Status, model.ExecutionStatusQueued); err != nil {
		return err
	}
	err = s.repo.UpdateStatus(id, model.ExecutionStatusQueued, "")
	if err == nil && s.events != nil {
		s.events.Record(id, "", model.ExecutionEventQueued, string(model.ExecutionStatusQueued), "execution queued", requestID, nil)
	}
	return err
}
func (s *ExecutionService) MarkRunning(id, requestID string) error {
	e, err := s.GetExecutionInternal(id)
	if err != nil {
		return err
	}
	if err := s.sm.ValidateTransition(e.Status, model.ExecutionStatusRunning); err != nil {
		return err
	}
	if s.audit != nil {
		if err := s.audit.RecordScoped(e.Scope(), e.UserID, model.AuditExecutionStarted, model.AuditResourceExecution, id, requestID, "", model.AuditDecisionAllow, "", "execution started", map[string]any{"input_hash": e.InputHash}); err != nil {
			return err
		}
	}
	err = s.repo.UpdateStatus(id, model.ExecutionStatusRunning, "")
	if err == nil && s.events != nil {
		s.events.Record(id, "", model.ExecutionEventStarted, string(model.ExecutionStatusRunning), "execution started", requestID, nil)
	}
	return err
}
func (s *ExecutionService) FinishExecution(id string, status model.ExecutionStatus, errMsg, requestID string) error {
	e, err := s.GetExecutionInternal(id)
	if err != nil {
		return err
	}
	if err := s.sm.ValidateTransition(e.Status, status); err != nil {
		return err
	}
	if s.audit != nil {
		if err := s.audit.RecordScoped(e.Scope(), e.UserID, model.AuditExecutionFinished, model.AuditResourceExecution, id, requestID, "", model.AuditDecisionAllow, "", "execution finished", map[string]any{"status": status, "input_hash": e.InputHash}); err != nil {
			return err
		}
	}
	if err := s.repo.UpdateStatus(id, status, errMsg); err != nil {
		return err
	}
	if s.events != nil {
		typ := model.ExecutionEventFinished
		if status == model.ExecutionStatusFailed {
			typ = model.ExecutionEventFailed
		}
		s.events.Record(id, "", typ, string(status), "execution finished", requestID, map[string]any{"error": errMsg})
	}
	observability.IncTaskCompleted(status)
	return nil
}
func (s *ExecutionService) FailExecution(id, errMsg, requestID string) error {
	return s.FinishExecution(id, model.ExecutionStatusFailed, errMsg, requestID)
}

// RecoverIncompleteExecutions rehydrates executions that were persisted but lost from the
// in-memory queue during a process restart. Running tasks cannot be safely resumed in this
// v3 single-node runtime, so they are failed explicitly instead of being left stuck forever.
func (s *ExecutionService) RecoverIncompleteExecutions(ctx context.Context, requestID string) (int, error) {
	items, err := s.repo.ListByStatuses(model.ExecutionStatusPending, model.ExecutionStatusQueued, model.ExecutionStatusRunning)
	if err != nil {
		return 0, err
	}
	recovered := 0
	for _, e := range items {
		switch e.Status {
		case model.ExecutionStatusPending:
			if err := s.MarkQueued(e.ID, requestID); err != nil {
				return recovered, err
			}
			fallthrough
		case model.ExecutionStatusQueued:
			if s.queue == nil {
				continue
			}
			if err := s.queue.Submit(ctx, e.ID); err != nil {
				observability.IncQueueFull()
				return recovered, response.NewAppError(response.CodeQueueFull, "execution queue is full during recovery")
			}
			observability.IncTaskRecovered()
			if s.events != nil {
				s.events.Record(e.ID, "", model.ExecutionEventRecovered, string(model.ExecutionStatusQueued), "execution recovered and requeued", requestID, nil)
			}
			recovered++
		case model.ExecutionStatusRunning:
			_ = s.repo.UpdateStatus(e.ID, model.ExecutionStatusFailed, "recovered stale running execution after process restart")
			observability.IncTaskCompleted(model.ExecutionStatusFailed)
		}
	}
	return recovered, nil
}
