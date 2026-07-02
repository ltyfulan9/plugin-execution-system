package model

import "time"

type AuditAction string
type AuditResourceType string

const (
	AuditPluginLoaded        AuditAction = "plugin.loaded"
	AuditPluginStatusChanged AuditAction = "plugin.status_changed"
	AuditExecutionCreated    AuditAction = "execution.created"
	AuditExecutionQueued     AuditAction = "execution.queued"
	AuditExecutionStarted    AuditAction = "execution.started"
	AuditExecutionFinished   AuditAction = "execution.finished"
	AuditExecutionCanceled   AuditAction = "execution.canceled"
	AuditRuntimeError        AuditAction = "runtime.error"

	AuditResourcePlugin    AuditResourceType = "plugin"
	AuditResourceExecution AuditResourceType = "execution"
	AuditResourceSystem    AuditResourceType = "system"
)

type AuditDecision string

const (
	AuditDecisionAllow AuditDecision = "allow"
	AuditDecisionDeny  AuditDecision = "deny"
	AuditDecisionError AuditDecision = "error"
)

type AuditLog struct {
	TenantID     string            `json:"tenant_id"`
	ProjectID    string            `json:"project_id"`
	ID           string            `json:"id"`
	ActorID      string            `json:"actor_id"`
	ActorType    string            `json:"actor_type,omitempty"`
	UserID       string            `json:"user_id,omitempty"` // compatibility alias for older clients
	Action       AuditAction       `json:"action"`
	ResourceType AuditResourceType `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Decision     AuditDecision     `json:"decision"`
	Reason       string            `json:"reason,omitempty"`
	RequestID    string            `json:"request_id"`
	TraceID      string            `json:"trace_id,omitempty"`
	PluginDigest string            `json:"plugin_digest,omitempty"`
	InputHash    string            `json:"input_hash,omitempty"`
	ResultHash   string            `json:"result_hash,omitempty"`
	Message      string            `json:"message"`
	DetailJSON   map[string]any    `json:"detail,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

func IsValidAuditAction(a AuditAction) bool { return a != "" }
