package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type ExecutionEventRepository struct{ store *storage.JSONStore }

func NewExecutionEventRepository(store *storage.JSONStore) *ExecutionEventRepository {
	return &ExecutionEventRepository{store: store}
}

func (r *ExecutionEventRepository) all() ([]model.ExecutionEvent, error) {
	var items []model.ExecutionEvent
	err := r.store.Load("execution_events", &items)
	return items, err
}

func (r *ExecutionEventRepository) save(items []model.ExecutionEvent) error {
	return r.store.Save("execution_events", items)
}

func (r *ExecutionEventRepository) Create(event model.ExecutionEvent) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	items = append(items, event)
	return r.save(items)
}

func (r *ExecutionEventRepository) ListByExecutionID(executionID string) ([]model.ExecutionEvent, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	out := []model.ExecutionEvent{}
	for _, item := range items {
		if item.ExecutionID == executionID {
			out = append(out, item)
		}
	}
	return out, nil
}
