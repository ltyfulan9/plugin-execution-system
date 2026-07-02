package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type AuditRepository struct{ store *storage.JSONStore }

func NewAuditRepository(store *storage.JSONStore) *AuditRepository {
	return &AuditRepository{store: store}
}
func (r *AuditRepository) all() ([]model.AuditLog, error) {
	var items []model.AuditLog
	err := r.store.Load("audit_logs", &items)
	return items, err
}
func (r *AuditRepository) save(items []model.AuditLog) error {
	return r.store.Save("audit_logs", items)
}
func (r *AuditRepository) Create(log model.AuditLog) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	items = append(items, log)
	return r.save(items)
}
func (r *AuditRepository) List() ([]model.AuditLog, error) { return r.all() }
func (r *AuditRepository) ListByResource(resourceType model.AuditResourceType, resourceID string) ([]model.AuditLog, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	out := []model.AuditLog{}
	for _, it := range items {
		if it.ResourceType == resourceType && it.ResourceID == resourceID {
			out = append(out, it)
		}
	}
	return out, nil
}
func (r *AuditRepository) ListByRequestID(requestID string) ([]model.AuditLog, error) {
	items, err := r.all()
	if err != nil {
		return nil, err
	}
	out := []model.AuditLog{}
	for _, it := range items {
		if it.RequestID == requestID {
			out = append(out, it)
		}
	}
	return out, nil
}
