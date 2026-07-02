package worker

import (
	"context"
	"fmt"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/queue"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/service"
	"plugin-execution-system/internal/storage"
)

// MetadataExecutionHandler is the production durable-worker handler. It uses
// MetadataStore transaction helpers for every critical state transition and does
// not treat process memory as the source of truth.
type MetadataExecutionHandler struct {
	meta      storage.MetadataStore
	execs     repository.ExecutionStore
	plugins   *service.PluginService
	runtime   *service.RuntimeService
	results   *service.ResultService
	workerID  string
	requestID string
}

func NewMetadataExecutionHandler(meta storage.MetadataStore, execs repository.ExecutionStore, plugins *service.PluginService, runtime *service.RuntimeService, results *service.ResultService, workerID string) *MetadataExecutionHandler {
	if workerID == "" {
		workerID = "durable-worker"
	}
	return &MetadataExecutionHandler{meta: meta, execs: execs, plugins: plugins, runtime: runtime, results: results, workerID: workerID, requestID: "durable-worker"}
}

func (h *MetadataExecutionHandler) HandleLeasedExecution(ctx context.Context, ref queue.TaskRef) (err error) {
	e, ok, err := h.execs.GetByID(ref.TaskID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("execution not found: %s", ref.TaskID)
	}
	attempt := model.ExecutionAttempt{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("attempt"), ExecutionID: e.ID, AttemptNo: ref.AttemptNo, WorkerID: h.workerID, LeaseID: ref.LeaseID, Status: model.ExecutionAttemptRunning, StartedAt: time.Now().UTC()}
	if !ref.LeaseUntil.IsZero() {
		attempt.LeaseUntil = &ref.LeaseUntil
	}
	now := time.Now().UTC()
	attempt.HeartbeatAt = &now
	attemptStartEvent := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventStarted, Status: string(model.ExecutionStatusRunning), Message: "durable worker attempt started", RequestID: h.requestID, CreatedAt: now}
	if err := h.meta.AppendAttempt(ctx, attempt, attemptStartEvent); err != nil {
		return err
	}
	if e.Status == model.ExecutionStatusCanceled {
		return nil
	}
	if e.Status != model.ExecutionStatusRunning {
		audit := model.AuditLog{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("audit"), ActorID: h.workerID, ActorType: "worker", Action: model.AuditExecutionStarted, ResourceType: model.AuditResourceExecution, ResourceID: e.ID, Decision: model.AuditDecisionAllow, RequestID: h.requestID, InputHash: e.InputHash, Message: "execution leased by durable worker", CreatedAt: now}
		transitionEvent := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventStarted, Status: string(model.ExecutionStatusRunning), Message: "execution state changed to running", RequestID: h.requestID, CreatedAt: time.Now().UTC()}
		if err := h.meta.TransitionExecutionWithEvent(ctx, e.ID, e.Status, model.ExecutionStatusRunning, transitionEvent, audit); err != nil {
			return err
		}
		e.Status = model.ExecutionStatusRunning
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("durable worker panic: %v", r)
		}
		if err != nil {
			finished := time.Now().UTC()
			attempt.Status = model.ExecutionAttemptFailed
			attempt.ErrorMessage = err.Error()
			attempt.FinishedAt = &finished
			failEvent := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventFailed, Status: string(model.ExecutionStatusFailed), Message: err.Error(), RequestID: h.requestID, CreatedAt: finished}
			_ = h.meta.AppendAttempt(context.Background(), attempt, failEvent)
		}
	}()
	plugins, err := h.plugins.GetExecutablePluginsInScope(e.PluginIDs, e.Scope())
	if err != nil {
		return err
	}
	runResults := h.runtime.RunPlugins(ctx, service.RunRequest{Execution: e, Plugins: plugins, RequestID: h.requestID})
	final := h.results.AggregateExecutionStatus(runResults)
	finished := time.Now().UTC()
	attempt.Status = model.ExecutionAttemptSuccess
	attempt.FinishedAt = &finished
	attemptFinishedEvent := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventFinished, Status: string(final), Message: "durable worker attempt finished", RequestID: h.requestID, CreatedAt: finished}
	finishedEvent := model.ExecutionEvent{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("evt"), ExecutionID: e.ID, Type: model.ExecutionEventFinished, Status: string(final), Message: "execution finalized", RequestID: h.requestID, CreatedAt: finished.Add(time.Nanosecond)}
	audit := model.AuditLog{TenantID: e.TenantID, ProjectID: e.ProjectID, ID: service.NewPublicID("audit"), ActorID: h.workerID, ActorType: "worker", Action: model.AuditExecutionFinished, ResourceType: model.AuditResourceExecution, ResourceID: e.ID, Decision: model.AuditDecisionAllow, RequestID: h.requestID, InputHash: e.InputHash, Message: "execution finished by durable worker", DetailJSON: map[string]any{"status": final}, CreatedAt: finished}
	if err := h.meta.AppendAttempt(ctx, attempt, attemptFinishedEvent); err != nil {
		return err
	}
	return h.meta.AppendResultAndFinalize(ctx, e.ID, runResults, final, finishedEvent, audit)
}
