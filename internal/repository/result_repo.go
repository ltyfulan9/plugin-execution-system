package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type ResultRepository struct{ store *storage.JSONStore }

func NewResultRepository(store *storage.JSONStore) *ResultRepository {
	return &ResultRepository{store: store}
}
func (r *ResultRepository) all() ([]model.ExecutionResult, error) {
	var items []model.ExecutionResult
	err := r.store.Load("execution_results", &items)
	return items, err
}
func (r *ResultRepository) save(items []model.ExecutionResult) error {
	return r.store.Save("execution_results", items)
}
func (r *ResultRepository) Create(res model.ExecutionResult) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	items = append(items, res)
	return r.save(items)
}
func (r *ResultRepository) BatchCreate(results []model.ExecutionResult) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	items = append(items, results...)
	return r.save(items)
}
func (r *ResultRepository) GetByExecutionID(id string) ([]model.ExecutionResult, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	out := []model.ExecutionResult{}
	for _, it := range items {
		if it.ExecutionID == id {
			out = append(out, it)
		}
	}
	return out, nil
}
func (r *ResultRepository) DeleteByExecutionID(id string) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	out := []model.ExecutionResult{}
	for _, it := range items {
		if it.ExecutionID != id {
			out = append(out, it)
		}
	}
	return r.save(out)
}
