package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type ExecutionAttemptRepository struct{ store *storage.JSONStore }

func NewExecutionAttemptRepository(store *storage.JSONStore) *ExecutionAttemptRepository {
	return &ExecutionAttemptRepository{store: store}
}

func (r *ExecutionAttemptRepository) all() ([]model.ExecutionAttempt, error) {
	var items []model.ExecutionAttempt
	err := r.store.Load("execution_attempts", &items)
	return items, err
}

func (r *ExecutionAttemptRepository) save(items []model.ExecutionAttempt) error {
	return r.store.Save("execution_attempts", items)
}

func (r *ExecutionAttemptRepository) Create(attempt model.ExecutionAttempt) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	items = append(items, attempt)
	return r.save(items)
}

func (r *ExecutionAttemptRepository) Update(attempt model.ExecutionAttempt) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == attempt.ID {
			items[i] = attempt
			return r.save(items)
		}
	}
	items = append(items, attempt)
	return r.save(items)
}

func (r *ExecutionAttemptRepository) ListByExecutionID(executionID string) ([]model.ExecutionAttempt, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	out := []model.ExecutionAttempt{}
	for _, item := range items {
		if item.ExecutionID == executionID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *ExecutionAttemptRepository) NextAttemptNo(executionID string) (int, error) {
	items, err := r.ListByExecutionID(executionID)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, item := range items {
		if item.AttemptNo > max {
			max = item.AttemptNo
		}
	}
	return max + 1, nil
}
