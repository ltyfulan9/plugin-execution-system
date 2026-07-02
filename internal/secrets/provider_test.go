package secrets

import (
	"context"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestResolveEnvUsesScope(t *testing.T) {
	p := NewMemoryProvider()
	p.Put(model.ResourceScope{TenantID: "t1", ProjectID: "p1"}, "secret://openai", []byte("sk-test"))
	env, err := ResolveEnv(context.Background(), p, model.ResourceScope{TenantID: "t1", ProjectID: "p1"}, map[string]string{"OPENAI_API_KEY": "secret://openai"})
	if err != nil || env["OPENAI_API_KEY"] != "sk-test" {
		t.Fatalf("unexpected env=%v err=%v", env, err)
	}
	_, err = ResolveEnv(context.Background(), p, model.ResourceScope{TenantID: "t2", ProjectID: "p1"}, map[string]string{"OPENAI_API_KEY": "secret://openai"})
	if err == nil {
		t.Fatalf("expected cross-scope secret lookup to fail")
	}
}

func TestResolveEnvRejectsUnsafeNames(t *testing.T) {
	_, err := ResolveEnv(context.Background(), NewMemoryProvider(), model.NewDefaultScope(), map[string]string{"BAD-NAME": "x"})
	if err == nil {
		t.Fatalf("expected invalid env name")
	}
}
