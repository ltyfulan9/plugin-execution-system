package service

import (
	"os"
	"path/filepath"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
)

type RegistryService struct {
	manifest   *ManifestService
	plugins    *PluginService
	registry   repository.RegistryStore
	pluginRepo repository.PluginStore
}

func NewRegistryService(m *ManifestService, p *PluginService, r repository.RegistryStore, pr repository.PluginStore) *RegistryService {
	return &RegistryService{manifest: m, plugins: p, registry: r, pluginRepo: pr}
}
func (s *RegistryService) ScanAndSync(root string) (model.RegistrySyncResult, error) {
	return s.ReloadPlugins(root)
}
func (s *RegistryService) ReloadPlugins(root string) (model.RegistrySyncResult, error) {
	res := model.RegistrySyncResult{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return res, err
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		path := ManifestPath(dir)
		m, b, err := s.manifest.LoadManifest(path)
		if err != nil {
			res.Errors = append(res.Errors, e.Name()+": "+err.Error())
			continue
		}
		p := s.manifest.BuildPluginFromManifest(m, dir)
		existed, ok, _ := s.pluginRepo.GetByNameVersion(p.Name, p.Version)
		if err := s.plugins.UpsertFromManifest(p); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		saved, _, _ := s.pluginRepo.GetByNameVersion(p.Name, p.Version)
		seen[saved.ID] = true
		now := time.Now().UTC()
		_ = s.registry.UpsertRegistryRecord(model.PluginRegistryRecord{ID: "registry_" + saved.ID, PluginID: saved.ID, Name: saved.Name, Version: saved.Version, SourcePath: dir, ManifestHash: s.manifest.HashManifest(b), LastSeenAt: now, SyncedAt: now})
		if ok && existed.ID != "" {
			res.Updated++
		} else {
			res.Created++
		}
	}
	plugins, _ := s.pluginRepo.List()
	for _, p := range plugins {
		if p.Status != model.PluginStatusRemoved && !seen[p.ID] {
			p.Status = model.PluginStatusRemoved
			p.UpdatedAt = time.Now().UTC()
			_ = s.pluginRepo.Update(p)
			res.Removed++
		}
	}
	return res, nil
}
