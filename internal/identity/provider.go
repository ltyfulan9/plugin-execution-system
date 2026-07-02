package identity

import (
	"context"
	"errors"
	"strings"

	"plugin-execution-system/internal/model"
)

// Provider is the enterprise identity boundary. Implementations can back this
// with OIDC, SAML, mTLS service identities, or local bootstrap tokens. The rest
// of the platform consumes only normalized principals and never parses provider-
// specific claims in handler/service code.
type Provider interface {
	Authenticate(ctx context.Context, credential Credential) (Principal, error)
}

type Credential struct {
	BearerToken string
	SAMLAssert  string
	MTLSSubject string
}

type Principal struct {
	TenantID  string     `json:"tenant_id"`
	ProjectID string     `json:"project_id"`
	ActorID   string     `json:"actor_id"`
	Username  string     `json:"username"`
	Role      model.Role `json:"role"`
	Groups    []string   `json:"groups,omitempty"`
	Provider  string     `json:"provider"`
	Subject   string     `json:"subject"`
}

func (p Principal) CurrentUser() model.CurrentUser {
	return model.CurrentUser{TenantID: p.TenantID, ProjectID: p.ProjectID, ID: p.ActorID, Username: p.Username, Role: p.Role}
}

// StaticProvider is restricted to dev/bootstrap scenarios. Production should
// use OIDC/SAML/mTLS providers plus RBAC/ABAC policy evaluation.
type StaticProvider struct{ principals map[string]Principal }

func NewStaticProvider(items map[string]Principal) *StaticProvider {
	return &StaticProvider{principals: items}
}

func (p *StaticProvider) Authenticate(ctx context.Context, cred Credential) (Principal, error) {
	_ = ctx
	token := strings.TrimSpace(cred.BearerToken)
	if token == "" {
		return Principal{}, errors.New("missing bearer token")
	}
	principal, ok := p.principals[token]
	if !ok {
		return Principal{}, errors.New("invalid bearer token")
	}
	if principal.Provider == "" {
		principal.Provider = "static"
	}
	return principal, nil
}
