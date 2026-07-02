package policy

import (
	"context"
	"fmt"
	"strings"

	"plugin-execution-system/internal/model"
)

const (
	ActionExecutionCreate = "execution:create"
	ActionExecutionRead   = "execution:read"
	ActionExecutionCancel = "execution:cancel"
	ActionPluginRead      = "plugin:read"
	ActionPluginManage    = "plugin:manage"
	ActionAuditRead       = "audit:read"
	ActionWebhookManage   = "webhook:manage"
	ActionSecretUse       = "secret:use"
	ActionArtifactRead    = "artifact:read"
)

type Input struct {
	Subject  model.PolicySubject  `json:"subject"`
	Action   string               `json:"action"`
	Resource model.PolicyResource `json:"resource"`
	Context  map[string]any       `json:"context,omitempty"`
}

type Engine interface {
	Evaluate(ctx context.Context, input Input) (model.PolicyEvaluation, error)
}

// AllowLocalEngine exists only for local development and unit tests. Production
// deployments should wire RBACEngine, OPA/Rego, or an equivalent policy engine.
type AllowLocalEngine struct{}

func (AllowLocalEngine) Evaluate(ctx context.Context, input Input) (model.PolicyEvaluation, error) {
	return model.PolicyEvaluation{Decision: model.PolicyDecisionAllow, Reason: "local allow engine; production should wire RBAC/OPA/Rego or equivalent"}, nil
}

// RBACEngine is a small deterministic enterprise baseline policy engine. It is
// intentionally boring: scope is checked first, then role/action permissions are
// checked. A future OPA adapter can implement the same Engine interface.
type RBACEngine struct {
	permissions map[model.Role]map[string]struct{}
}

func NewRBACEngine() *RBACEngine {
	return &RBACEngine{permissions: map[model.Role]map[string]struct{}{
		model.RoleUser:       set(ActionPluginRead, ActionExecutionCreate, ActionExecutionRead, ActionExecutionCancel, ActionArtifactRead),
		model.RoleAdmin:      set(ActionPluginRead, ActionPluginManage, ActionExecutionCreate, ActionExecutionRead, ActionExecutionCancel, ActionAuditRead, ActionWebhookManage, ActionSecretUse, ActionArtifactRead),
		model.RoleSuperAdmin: set("*"),
	}}
}

func (e *RBACEngine) Evaluate(ctx context.Context, input Input) (model.PolicyEvaluation, error) {
	_ = ctx
	input.Action = strings.TrimSpace(input.Action)
	if input.Action == "" {
		return deny("missing action"), nil
	}
	if input.Subject.ActorID == "" {
		return deny("missing actor"), nil
	}
	if input.Subject.Role != model.RoleSuperAdmin {
		subScope := model.ResourceScope{TenantID: input.Subject.TenantID, ProjectID: input.Subject.ProjectID}.Normalize()
		resScope := model.ResourceScope{TenantID: input.Resource.TenantID, ProjectID: input.Resource.ProjectID}.Normalize()
		if !model.SameScope(subScope, resScope) {
			return deny(fmt.Sprintf("scope mismatch: actor=%s/%s resource=%s/%s", subScope.TenantID, subScope.ProjectID, resScope.TenantID, resScope.ProjectID)), nil
		}
	}
	allowed := e.permissions[input.Subject.Role]
	if _, ok := allowed["*"]; ok {
		return allow("super admin wildcard"), nil
	}
	if _, ok := allowed[input.Action]; ok {
		return allow("role permission matched"), nil
	}
	return deny("role permission denied"), nil
}

func set(items ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}
func allow(reason string) model.PolicyEvaluation {
	return model.PolicyEvaluation{Decision: model.PolicyDecisionAllow, Reason: reason, PolicyID: "builtin-rbac"}
}
func deny(reason string) model.PolicyEvaluation {
	return model.PolicyEvaluation{Decision: model.PolicyDecisionDeny, Reason: reason, PolicyID: "builtin-rbac"}
}
