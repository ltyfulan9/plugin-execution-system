package service

import (
	"testing"

	"plugin-execution-system/internal/model"
)

func TestPluginStateMachine(t *testing.T) {
	sm := NewPluginStateMachine()
	if !sm.CanTransit(model.PluginStatusLoaded, model.PluginStatusEnabled) {
		t.Fatal("Loaded should be able to transit to Enabled")
	}
	if sm.CanTransit(model.PluginStatusRemoved, model.PluginStatusEnabled) {
		t.Fatal("Removed must not transit to Enabled")
	}
	if sm.CanExecute(model.Plugin{Status: model.PluginStatusDisabled}) {
		t.Fatal("Disabled plugin must not be executable")
	}
}
