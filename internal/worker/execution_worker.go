package worker

import (
	"context"
	"fmt"
	"os"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/service"
)

type ExecutionWorker struct {
	executions *service.ExecutionService
	plugins    *service.PluginService
	runtime    *service.RuntimeService
	results    *service.ResultService
	audit      *service.AuditService
	events     *service.ExecutionEventService
	attempts   *service.ExecutionAttemptService
	workerID   string
}

func NewExecutionWorker(e *service.ExecutionService, p *service.PluginService, r *service.RuntimeService, rs *service.ResultService, a *service.AuditService) *ExecutionWorker {
	return NewExecutionWorkerWithEvents(e, p, r, rs, a, nil, nil)
}

func NewExecutionWorkerWithEvents(e *service.ExecutionService, p *service.PluginService, r *service.RuntimeService, rs *service.ResultService, a *service.AuditService, events *service.ExecutionEventService, attempts *service.ExecutionAttemptService) *ExecutionWorker {
	workerID := os.Getenv("PES_WORKER_ID")
	if workerID == "" {
		workerID = "local-worker"
	}
	return &ExecutionWorker{executions: e, plugins: p, runtime: r, results: rs, audit: a, events: events, attempts: attempts, workerID: workerID}
}

func (w *ExecutionWorker) HandleExecution(ctx context.Context, executionID string) {
	var attempt model.ExecutionAttempt
	if w.attempts != nil {
		started, err := w.attempts.StartAttempt(executionID, w.workerID)
		if err == nil {
			attempt = started
		}
	}
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("worker panic: %v", r)
			_ = w.executions.FailExecution(executionID, errMsg, "worker")
			if w.attempts != nil && attempt.ID != "" {
				_ = w.attempts.FinishAttempt(attempt, model.ExecutionAttemptFailed, errMsg)
			}
		}
	}()
	if err := w.runExecutionSafely(ctx, executionID); err != nil {
		_ = w.executions.FailExecution(executionID, err.Error(), "worker")
		if w.attempts != nil && attempt.ID != "" {
			_ = w.attempts.FinishAttempt(attempt, model.ExecutionAttemptFailed, err.Error())
		}
		return
	}
	if w.attempts != nil && attempt.ID != "" {
		_ = w.attempts.FinishAttempt(attempt, model.ExecutionAttemptSuccess, "")
	}
}

func (w *ExecutionWorker) runExecutionSafely(ctx context.Context, executionID string) error {
	e, err := w.executions.GetExecutionInternal(executionID)
	if err != nil {
		return err
	}
	if e.Status == model.ExecutionStatusCanceled {
		return nil
	}
	if err := w.executions.MarkRunning(executionID, "worker"); err != nil {
		return err
	}
	plugins, err := w.plugins.GetExecutablePlugins(e.PluginIDs)
	if err != nil {
		return err
	}
	results := w.runtime.RunPlugins(ctx, service.RunRequest{Execution: e, Plugins: plugins, RequestID: "worker", Events: w.events})
	if err := w.results.BatchSaveResults(results); err != nil {
		return err
	}
	final := w.results.AggregateExecutionStatus(results)
	return w.executions.FinishExecution(e.ID, final, "", "worker")
}
