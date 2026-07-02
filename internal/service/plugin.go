package service

import (
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
)

type PluginService struct {
	repo repository.PluginStore
	sm   *PluginStateMachine
}

func NewPluginService(repo repository.PluginStore, sm *PluginStateMachine) *PluginService {
	return &PluginService{repo: repo, sm: sm}
}
func (s *PluginService) ListPlugins() ([]model.Plugin, error) { return s.repo.List() }
func (s *PluginService) GetPlugin(id string) (model.Plugin, error) {
	p, ok, err := s.repo.GetByID(id)
	if err != nil {
		return model.Plugin{}, err
	}
	if !ok {
		return model.Plugin{}, response.NewAppError(response.CodePluginNotFound, "plugin not found")
	}
	return p, nil
}
func (s *PluginService) EnablePlugin(id string) (model.Plugin, error) {
	p, err := s.GetPlugin(id)
	if err != nil {
		return model.Plugin{}, err
	}
	if !s.sm.CanEnable(p) {
		return model.Plugin{}, response.NewAppError(response.CodePluginStateInvalid, "plugin cannot be enabled")
	}
	p.Status = model.PluginStatusEnabled
	p.UpdatedAt = time.Now().UTC()
	return p, s.repo.Update(p)
}
func (s *PluginService) DisablePlugin(id string) (model.Plugin, error) {
	p, err := s.GetPlugin(id)
	if err != nil {
		return model.Plugin{}, err
	}
	if !s.sm.CanDisable(p) {
		return model.Plugin{}, response.NewAppError(response.CodePluginStateInvalid, "plugin cannot be disabled")
	}
	p.Status = model.PluginStatusDisabled
	p.UpdatedAt = time.Now().UTC()
	return p, s.repo.Update(p)
}
func (s *PluginService) ValidatePluginsExecutable(ids []string) ([]model.Plugin, error) {
	return s.ValidatePluginsExecutableInScope(ids, model.ResourceScope{})
}

func (s *PluginService) ValidatePluginsExecutableInScope(ids []string, scope model.ResourceScope) ([]model.Plugin, error) {
	scope = scope.Normalize()
	checkScope := !scope.IsZero()
	out := []model.Plugin{}
	for _, id := range ids {
		p, err := s.GetPlugin(id)
		if err != nil {
			return nil, err
		}
		if checkScope && !model.SameScope(scope, p.Scope()) {
			return nil, response.NewAppError(response.CodeForbidden, "plugin is outside current tenant/project scope")
		}
		if !s.sm.CanExecute(p) {
			return nil, response.NewDetailedError(response.CodePluginDisabled, "plugin is not executable", p.Name)
		}
		out = append(out, p)
	}
	return out, nil
}
func (s *PluginService) GetExecutablePlugins(ids []string) ([]model.Plugin, error) {
	return s.ValidatePluginsExecutable(ids)
}
func (s *PluginService) GetExecutablePluginsInScope(ids []string, scope model.ResourceScope) ([]model.Plugin, error) {
	return s.ValidatePluginsExecutableInScope(ids, scope)
}
func (s *PluginService) UpsertFromManifest(p model.Plugin) error {
	now := time.Now().UTC()
	existing, ok, err := s.repo.GetByNameVersion(p.Name, p.Version)
	if err != nil {
		return err
	}
	if ok {
		p.ID = existing.ID
		p.Status = existing.Status
		if p.Status == "" || p.Status == model.PluginStatusRemoved || p.Status == model.PluginStatusError {
			p.Status = model.PluginStatusLoaded
		}
		p.CreatedAt = existing.CreatedAt
		p.UpdatedAt = now
		return s.repo.Update(p)
	}
	p.ID = newID("plugin")
	p.Status = model.PluginStatusLoaded
	p.CreatedAt = now
	p.UpdatedAt = now
	return s.repo.Create(p)
}
