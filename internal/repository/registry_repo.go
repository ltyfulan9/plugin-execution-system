package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type RegistryRepository struct{ store *storage.JSONStore }

func NewRegistryRepository(store *storage.JSONStore) *RegistryRepository {
	return &RegistryRepository{store: store}
}
func (r *RegistryRepository) all() ([]model.PluginRegistryRecord, error) {
	var items []model.PluginRegistryRecord
	err := r.store.Load("plugin_registry", &items)
	return items, err
}
func (r *RegistryRepository) save(items []model.PluginRegistryRecord) error {
	return r.store.Save("plugin_registry", items)
}
func (r *RegistryRepository) UpsertRegistryRecord(rec model.PluginRegistryRecord) error {
	items, err := r.all()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].PluginID == rec.PluginID {
			items[i] = rec
			return r.save(items)
		}
	}
	items = append(items, rec)
	return r.save(items)
}
func (r *RegistryRepository) GetByPluginID(id string) (model.PluginRegistryRecord, bool, error) {
	items, err := r.all()
	if err != nil {
		return model.PluginRegistryRecord{}, false, err
	}
	for _, it := range items {
		if it.PluginID == id {
			return it, true, nil
		}
	}
	return model.PluginRegistryRecord{}, false, nil
}
func (r *RegistryRepository) ListRegistryRecords() ([]model.PluginRegistryRecord, error) {
	return r.all()
}
