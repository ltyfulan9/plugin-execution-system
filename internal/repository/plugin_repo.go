package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type PluginRepository struct{ store *storage.JSONStore }

func NewPluginRepository(store *storage.JSONStore) *PluginRepository {
	return &PluginRepository{store: store}
}
func (r *PluginRepository) all() ([]model.Plugin, error) {
	var items []model.Plugin
	err := r.store.Load("plugins", &items)
	return items, err
}
func (r *PluginRepository) save(items []model.Plugin) error { return r.store.Save("plugins", items) }
func (r *PluginRepository) Create(p model.Plugin) error {
	p.TenantID = model.ResourceScope{TenantID: p.TenantID, ProjectID: p.ProjectID}.Normalize().TenantID
	p.ProjectID = model.ResourceScope{TenantID: p.TenantID, ProjectID: p.ProjectID}.Normalize().ProjectID
	items, err := r.all()
	if err != nil {
		return err
	}
	items = append(items, p)
	return r.save(items)
}
func (r *PluginRepository) Update(p model.Plugin) error {
	scope := model.ResourceScope{TenantID: p.TenantID, ProjectID: p.ProjectID}.Normalize()
	p.TenantID, p.ProjectID = scope.TenantID, scope.ProjectID
	items, err := r.all()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == p.ID {
			items[i] = p
			return r.save(items)
		}
	}
	items = append(items, p)
	return r.save(items)
}
func (r *PluginRepository) GetByID(id string) (model.Plugin, bool, error) {
	items, err := r.all()
	if err != nil {
		return model.Plugin{}, false, err
	}
	for _, it := range items {
		if it.ID == id {
			return it, true, nil
		}
	}
	return model.Plugin{}, false, nil
}
func (r *PluginRepository) GetByNameVersion(name, version string) (model.Plugin, bool, error) {
	items, err := r.all()
	if err != nil {
		return model.Plugin{}, false, err
	}
	for _, it := range items {
		if it.Name == name && it.Version == version {
			return it, true, nil
		}
	}
	return model.Plugin{}, false, nil
}
func (r *PluginRepository) List() ([]model.Plugin, error) { return r.all() }
func (r *PluginRepository) UpdateStatus(id string, status model.PluginStatus) error {
	p, ok, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	p.Status = status
	return r.Update(p)
}
func (r *PluginRepository) UpdateError(id, msg string) error {
	p, ok, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	p.ErrorMessage = msg
	p.Status = model.PluginStatusError
	return r.Update(p)
}
func (r *PluginRepository) MarkRemoved(id string) error {
	return r.UpdateStatus(id, model.PluginStatusRemoved)
}
