package identity

import (
	"context"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestStaticProviderNormalizesPrincipal(t *testing.T) {
	provider := NewStaticProvider(map[string]Principal{"token": {TenantID: "t", ProjectID: "p", ActorID: "u", Username: "alice", Role: model.RoleAdmin}})
	p, err := provider.Authenticate(context.Background(), Credential{BearerToken: "token"})
	if err != nil {
		t.Fatal(err)
	}
	user := p.CurrentUser()
	if user.TenantID != "t" || user.ProjectID != "p" || user.Role != model.RoleAdmin {
		t.Fatalf("unexpected user: %#v", user)
	}
	if _, err := provider.Authenticate(context.Background(), Credential{BearerToken: "bad"}); err == nil {
		t.Fatal("expected invalid token")
	}
}
