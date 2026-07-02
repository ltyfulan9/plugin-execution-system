package service

import (
	"context"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/observability"
	policyengine "plugin-execution-system/internal/policy"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/storage"
)

// EnterpriseExecutionService is the production execution service. Unlike the
// local ExecutionService, task creation is atomic: current state, event log,
// audit log, and durable queue row are committed by MetadataStore in one
// transaction. It is the path used by Postgres/HA metadata store deployments.
type EnterpriseExecutionService struct {
	meta      storage.MetadataStore
	queries   repository.ExecutionStore
	pluginSvc *PluginService
	idem      *IdempotencyService
	sm        *ExecutionStateMachine
	policy    policyengine.Engine
}

func NewEnterpriseExecutionService(meta storage.MetadataStore, queries repository.ExecutionStore, pluginSvc *PluginService, idem *IdempotencyService, sm *ExecutionStateMachine) *EnterpriseExecutionService {
	return &EnterpriseExecutionService{meta: meta, queries: queries, pluginSvc: pluginSvc, idem: idem, sm: sm, policy: policyengine.NewRBACEngine()}
}

func (s *EnterpriseExecutionService) WithPolicyEngine(engine policyengine.Engine) *EnterpriseExecutionService {
	if engine != nil {
		s.policy = engine
	}
	return s
}

func (s *EnterpriseExecutionService) CreateExecution(ctx context.Context, user model.CurrentUser, pluginIDs []string, input map[string]any, idemKey, requestID string) (model.Execution, error) {
	scope := user.Scope()
	if err := s.enforcePolicy(ctx, user, policyengine.ActionExecutionCreate, model.PolicyResource{TenantID: scope.TenantID, ProjectID: scope.ProjectID, Type: "project", ID: scope.ProjectID}); err != nil {
		return model.Execution{}, err
	}
	if len(pluginIDs) == 0 || input == nil {
		return model.Execution{}, response.NewAppError(response.CodeInvalidArgument, "plugin_ids and input are required")
	}
	if _, err := s.pluginSvc.ValidatePluginsExecutableInScope(pluginIDs, scope); err != nil {
		return model.Execution{}, err
	}
	inputHash, err := s.idem.BuildInputHash(input)
	if err != nil {
		return model.Execution{}, err
	}
	pluginHash := s.idem.BuildPluginIDsHash(pluginIDs)
	now := time.Now().UTC()
	e := model.Execution{TenantID: scope.TenantID, ProjectID: scope.ProjectID, ID: newID("exec"), UserID: user.ID, PluginIDs: pluginIDs, InputJSON: input, InputHash: inputHash, PluginIDsHash: pluginHash, IdempotencyKey: idemKey, Status: model.ExecutionStatusQueued, CreatedAt: now, QueuedAt: &now}
	eventCreated := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: newID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventCreated, Status: string(model.ExecutionStatusPending), Message: "execution created", RequestID: requestID, CreatedAt: now}
	eventQueued := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: newID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventQueued, Status: string(model.ExecutionStatusQueued), Message: "execution queued", RequestID: requestID, CreatedAt: now.Add(time.Nanosecond)}
	audit := model.AuditLog{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: newID("audit"), ActorID: user.ID, Action: model.AuditExecutionCreated, ResourceType: model.AuditResourceExecution, ResourceID: e.ID, Decision: model.AuditDecisionAllow, RequestID: requestID, InputHash: inputHash, Message: "execution created and queued", DetailJSON: map[string]any{"plugin_ids_hash": pluginHash}, CreatedAt: now}
	createdOrExisting, created, err := s.meta.CreateExecutionAndEnqueue(ctx, e, []model.ExecutionEvent{eventCreated, eventQueued}, audit, now)
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
	observability.IncTaskSubmitted()
	return createdOrExisting, nil
}

func (s *EnterpriseExecutionService) GetExecution(user model.CurrentUser, id string) (model.Execution, error) {
	e, ok, err := s.queries.GetByID(id)
	if err != nil {
		return model.Execution{}, err
	}
	if !ok {
		return model.Execution{}, response.NewAppError(response.CodeExecutionNotFound, "execution not found")
	}
	if err := s.enforcePolicy(context.Background(), user, policyengine.ActionExecutionRead, model.PolicyResource{TenantID: e.TenantID, ProjectID: e.ProjectID, Type: "execution", ID: e.ID}); err != nil {
		return model.Execution{}, err
	}
	if !model.IsSuperAdminRole(user.Role) && !model.SameScope(user.Scope(), e.Scope()) {
		return model.Execution{}, response.NewAppError(response.CodeForbidden, "cannot view execution")
	}
	if !model.IsAdminRole(user.Role) && e.UserID != user.ID {
		return model.Execution{}, response.NewAppError(response.CodeForbidden, "cannot view execution")
	}
	return e, nil
}
func (s *EnterpriseExecutionService) GetExecutionInternal(id string) (model.Execution, error) {
	e, ok, err := s.queries.GetByID(id)
	if err != nil {
		return model.Execution{}, err
	}
	if !ok {
		return model.Execution{}, response.NewAppError(response.CodeExecutionNotFound, "execution not found")
	}
	return e, nil
}
func (s *EnterpriseExecutionService) ListExecutions(user model.CurrentUser) ([]model.Execution, error) {
	if model.IsSuperAdminRole(user.Role) {
		return s.queries.ListAll()
	}
	if model.IsAdminRole(user.Role) {
		return s.queries.ListByScope(user.Scope())
	}
	return s.queries.ListByUserID(user.ID)
}
func (s *EnterpriseExecutionService) CancelExecution(user model.CurrentUser, id, requestID string) (model.Execution, error) {
	e, err := s.GetExecution(user, id)
	if err != nil {
		return model.Execution{}, err
	}
	if err := s.enforcePolicy(context.Background(), user, policyengine.ActionExecutionCancel, model.PolicyResource{TenantID: e.TenantID, ProjectID: e.ProjectID, Type: "execution", ID: e.ID}); err != nil {
		return model.Execution{}, err
	}
	if !s.sm.CanCancel(e.Status) {
		return model.Execution{}, response.NewAppError(response.CodeExecutionStateInvalid, "execution cannot be canceled")
	}
	now := time.Now().UTC()
	event := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: newID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventCanceled, Status: string(model.ExecutionStatusCanceled), Message: "execution canceled", RequestID: requestID, CreatedAt: now}
	audit := model.AuditLog{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: newID("audit"), ActorID: user.ID, Action: model.AuditExecutionCanceled, ResourceType: model.AuditResourceExecution, ResourceID: e.ID, Decision: model.AuditDecisionAllow, RequestID: requestID, InputHash: e.InputHash, Message: "execution canceled", CreatedAt: now}
	if err := s.meta.TransitionExecutionWithEvent(ctxOrBackground(nil), e.ID, e.Status, model.ExecutionStatusCanceled, event, audit); err != nil {
		return model.Execution{}, err
	}
	return s.GetExecution(user, id)
}

func (s *EnterpriseExecutionService) enforcePolicy(ctx context.Context, user model.CurrentUser, action string, resource model.PolicyResource) error {
	if s.policy == nil {
		return nil
	}
	decision, err := s.policy.Evaluate(ctx, policyengine.Input{Subject: model.PolicySubject{TenantID: user.TenantID, ProjectID: user.ProjectID, ActorID: user.ID, Role: user.Role}, Action: action, Resource: resource})
	if err != nil {
		return err
	}
	if decision.Decision != model.PolicyDecisionAllow {
		return response.NewDetailedError(response.CodeForbidden, "policy denied request", decision.Reason)
	}
	return nil
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}
