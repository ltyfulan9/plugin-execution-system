package model

type PolicyDecision string

const (
	PolicyDecisionAllow PolicyDecision = "allow"
	PolicyDecisionDeny  PolicyDecision = "deny"
)

type PolicySubject struct {
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
	ActorID   string `json:"actor_id"`
	Role      Role   `json:"role"`
}

type PolicyResource struct {
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
	Type      string `json:"type"`
	ID        string `json:"id"`
}

type PolicyEvaluation struct {
	Decision PolicyDecision `json:"decision"`
	Reason   string         `json:"reason,omitempty"`
	PolicyID string         `json:"policy_id,omitempty"`
}
