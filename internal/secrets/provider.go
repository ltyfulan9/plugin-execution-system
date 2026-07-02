package secrets

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"plugin-execution-system/internal/model"
)

type Reference struct {
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Ref       string `json:"ref"`
}

type Provider interface {
	Resolve(ctx context.Context, ref Reference) ([]byte, error)
}

// InjectionPlan is passed to runners; raw secret values must not be stored in manifests.
type InjectionPlan struct {
	Env   map[string]Reference `json:"env,omitempty"`
	Files map[string]Reference `json:"files,omitempty"`
}

// MemoryProvider is a deterministic test/dev provider. Production deployments
// should back Provider with Vault, cloud secrets manager, KMS-encrypted metadata,
// or an equivalent enterprise secret plane.
type MemoryProvider struct {
	mu      sync.RWMutex
	secrets map[string][]byte
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{secrets: map[string][]byte{}}
}

func (p *MemoryProvider) Put(scope model.ResourceScope, ref string, value []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.secrets[key(scope, ref)] = append([]byte(nil), value...)
}

func (p *MemoryProvider) Resolve(ctx context.Context, ref Reference) ([]byte, error) {
	_ = ctx
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.secrets[key(model.ResourceScope{TenantID: ref.TenantID, ProjectID: ref.ProjectID}, ref.Ref)]
	if !ok {
		return nil, fmt.Errorf("secret reference not found: %s", ref.Ref)
	}
	return append([]byte(nil), v...), nil
}

func ResolveEnv(ctx context.Context, provider Provider, scope model.ResourceScope, refs map[string]string) (map[string]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	if provider == nil {
		return nil, fmt.Errorf("secret provider is not configured")
	}
	scope = scope.Normalize()
	out := make(map[string]string, len(refs))
	for envName, ref := range refs {
		if !safeEnvName(envName) {
			return nil, fmt.Errorf("secret env name %q is invalid", envName)
		}
		b, err := provider.Resolve(ctx, Reference{TenantID: scope.TenantID, ProjectID: scope.ProjectID, Name: envName, Provider: "default", Ref: ref})
		if err != nil {
			return nil, err
		}
		if strings.ContainsAny(string(b), "\x00\r\n") {
			return nil, fmt.Errorf("secret value for %q contains unsafe control characters", envName)
		}
		out[envName] = string(b)
	}
	return out, nil
}

func safeEnvName(name string) bool {
	if name == "" || strings.ContainsAny(name, "=\x00\r\n") || strings.HasPrefix(name, "-") {
		return false
	}
	for _, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func key(scope model.ResourceScope, ref string) string {
	scope = scope.Normalize()
	return scope.TenantID + "/" + scope.ProjectID + "/" + ref
}
