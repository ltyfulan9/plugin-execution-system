package service

import (
	"context"
	"strings"
	"testing"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/secrets"
)

func TestRuntimePrepareRejectsInlineEnv(t *testing.T) {
	rt := NewRuntimeService(NewProcessRunner(), 1024)
	_, err := rt.preparePluginForRuntime(context.Background(), model.Plugin{EntryType: "container", Env: map[string]string{"SECRET": "plain"}})
	if err == nil || !strings.Contains(err.Error(), "inline environment") {
		t.Fatalf("expected inline env rejection, got %v", err)
	}
}

func TestRuntimePrepareResolvesSecretRefsForContainer(t *testing.T) {
	provider := secrets.NewMemoryProvider()
	scope := model.ResourceScope{TenantID: "t1", ProjectID: "p1"}
	provider.Put(scope, "secret://api-key", []byte("value"))
	rt := NewRuntimeService(NewProcessRunner(), 1024).WithSecretProvider(provider)
	plugin, err := rt.preparePluginForRuntime(context.Background(), model.Plugin{TenantID: "t1", ProjectID: "p1", EntryType: "container", SecretRefs: map[string]string{"API_KEY": "secret://api-key"}})
	if err != nil || plugin.Env["API_KEY"] != "value" {
		t.Fatalf("unexpected plugin=%+v err=%v", plugin, err)
	}
}

func TestRuntimePrepareRejectsSecretRefsForProcess(t *testing.T) {
	rt := NewRuntimeService(NewProcessRunner(), 1024).WithSecretProvider(secrets.NewMemoryProvider())
	_, err := rt.preparePluginForRuntime(context.Background(), model.Plugin{EntryType: "process", SecretRefs: map[string]string{"API_KEY": "secret://api-key"}})
	if err == nil || !strings.Contains(err.Error(), "container runtime") {
		t.Fatalf("expected process secret injection rejection, got %v", err)
	}
}
