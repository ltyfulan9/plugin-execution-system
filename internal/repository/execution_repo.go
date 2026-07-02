package repository

import (
	"sync"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type ExecutionRepository struct {
	store *storage.JSONStore
	mu    sync.Mutex
}

func NewExecutionRepository(store *storage.JSONStore) *ExecutionRepository {
	return &ExecutionRepository{store: store}
}
func (r *ExecutionRepository) all() ([]model.Execution, error) {
	var items []model.Execution
	err := r.store.Load("executions", &items)
	return items, err
}
func (r *ExecutionRepository) save(items []model.Execution) error {
	return r.store.Save("executions", items)
}
func (r *ExecutionRepository) Create(e model.Execution) error {
	_, _, err := r.CreateWithIdempotency(e)
	return err
}

func (r *ExecutionRepository) CreateWithIdempotency(e model.Execution) (model.Execution, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items, err := r.all()
	if err != nil {
		return model.Execution{}, false, err
	}
	if e.IdempotencyKey != "" {
		for _, it := range items {
			if model.SameScope(it.Scope(), e.Scope()) && it.UserID == e.UserID && it.IdempotencyKey == e.IdempotencyKey {
				return it, false, nil
			}
		}
	}
	items = append(items, e)
	return e, true, r.save(items)
}
func (r *ExecutionRepository) Update(e model.Execution) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == e.ID {
			items[i] = e
			return r.save(items)
		}
	}
	items = append(items, e)
	return r.save(items)
}
func (r *ExecutionRepository) GetByID(id string) (model.Execution, bool, error) {
	items, err := r.all()
	if err != nil {
		return model.Execution{}, false, err
	}
	for _, it := range items {
		if it.ID == id {
			return it, true, nil
		}
	}
	return model.Execution{}, false, nil
}
func (r *ExecutionRepository) ListByUserID(userID string) ([]model.Execution, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	out := []model.Execution{}
	for _, it := range items {
		if it.UserID == userID {
			out = append(out, it)
		}
	}
	return out, nil
}
func (r *ExecutionRepository) ListAll() ([]model.Execution, error) { return r.all() }
func (r *ExecutionRepository) UpdateStatus(id string, status model.ExecutionStatus, errMsg string) error {
	e, ok, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	e.Status = status
	e.ErrorMessage = errMsg
	now := time.Now().UTC()
	if status == model.ExecutionStatusQueued {
		e.QueuedAt = &now
	}
	if status == model.ExecutionStatusRunning {
		e.StartedAt = &now
	}
	if model.IsFinalExecutionStatus(status) {
		e.FinishedAt = &now
	}
	return r.Update(e)
}
func (r *ExecutionRepository) FindByIdempotencyKey(userID, key string) (model.Execution, bool, error) {
	items, err := r.all()
	if err != nil {
		return model.Execution{}, false, err
	}
	for _, it := range items {
		if it.UserID == userID && it.IdempotencyKey == key && key != "" {
			return it, true, nil
		}
	}
	return model.Execution{}, false, nil
}

func (r *ExecutionRepository) ListByStatuses(statuses ...model.ExecutionStatus) ([]model.Execution, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	allowed := make(map[model.ExecutionStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}
	out := []model.Execution{}
	for _, it := range items {
		if _, ok := allowed[it.Status]; ok {
			out = append(out, it)
		}
	}
	return out, nil
}

func (r *ExecutionRepository) ListByScope(scope model.ResourceScope) ([]model.Execution, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	scope = scope.Normalize()
	out := []model.Execution{}
	for _, it := range items {
		if model.SameScope(scope, it.Scope()) {
			out = append(out, it)
		}
	}
	return out, nil
}
