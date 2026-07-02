package service

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
)

type PluginStateMachine struct{}

func NewPluginStateMachine() *PluginStateMachine { return &PluginStateMachine{} }

func (m *PluginStateMachine) CanTransit(from, to model.PluginStatus) bool {
	allowed := map[model.PluginStatus][]model.PluginStatus{
		model.PluginStatusDiscovered: {model.PluginStatusLoaded, model.PluginStatusError},
		model.PluginStatusLoaded:     {model.PluginStatusEnabled, model.PluginStatusDisabled, model.PluginStatusError, model.PluginStatusRemoved},
		model.PluginStatusEnabled:    {model.PluginStatusDisabled, model.PluginStatusError},
		model.PluginStatusDisabled:   {model.PluginStatusEnabled, model.PluginStatusError, model.PluginStatusRemoved},
		model.PluginStatusError:      {model.PluginStatusLoaded, model.PluginStatusRemoved},
		model.PluginStatusRemoved:    {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

func (m *PluginStateMachine) ValidateTransition(from, to model.PluginStatus) error {
	if !m.CanTransit(from, to) {
		return response.NewAppError(response.CodePluginStateInvalid, "invalid plugin state transition")
	}
	return nil
}
func (m *PluginStateMachine) CanEnable(p model.Plugin) bool {
	return p.Status == model.PluginStatusLoaded || p.Status == model.PluginStatusDisabled
}
func (m *PluginStateMachine) CanDisable(p model.Plugin) bool {
	return p.Status == model.PluginStatusEnabled
}
func (m *PluginStateMachine) CanExecute(p model.Plugin) bool {
	return p.Status == model.PluginStatusEnabled
}
