package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/storage"
)

type captureQueue struct{ ids []string }

func (q *captureQueue) Submit(ctx context.Context, executionID string) error {
	q.ids = append(q.ids, executionID)
	return nil
}

func TestRecoverIncompleteExecutionsRequeuesQueuedAndFailsStaleRunning(t *testing.T) {
	store, err := storage.OpenJSONStore(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(store); err != nil {
		t.Fatal(err)
	}
	repos := repository.NewRepositories(store)
	q := &captureQueue{}
	execSvc := NewExecutionService(repos.Execution, nil, NewIdempotencyService(repos.Execution), NewExecutionStateMachine(), nil, q)
	now := time.Now().UTC()
	queued := model.Execution{ID: "exec_queued", UserID: "u", PluginIDs: []string{"p"}, InputJSON: map[string]any{"x": 1}, Status: model.ExecutionStatusQueued, CreatedAt: now}
	running := model.Execution{ID: "exec_running", UserID: "u", PluginIDs: []string{"p"}, InputJSON: map[string]any{"x": 1}, Status: model.ExecutionStatusRunning, CreatedAt: now}
	if err := repos.Execution.Create(queued); err != nil {
		t.Fatal(err)
	}
	if err := repos.Execution.Create(running); err != nil {
		t.Fatal(err)
	}
	recovered, err := execSvc.RecoverIncompleteExecutions(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 || len(q.ids) != 1 || q.ids[0] != queued.ID {
		t.Fatalf("unexpected recovery result recovered=%d ids=%v", recovered, q.ids)
	}
	after, _, err := repos.Execution.GetByID(running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != model.ExecutionStatusFailed {
		t.Fatalf("stale running should be failed, got %+v", after)
	}
}
