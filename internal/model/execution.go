package model

import "time"

type Execution struct {
	TenantID       string          `json:"tenant_id"`
	ProjectID      string          `json:"project_id"`
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	PluginIDs      []string        `json:"plugin_ids"`
	InputJSON      map[string]any  `json:"input"`
	InputHash      string          `json:"input_hash"`
	PluginIDsHash  string          `json:"plugin_ids_hash"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	Status         ExecutionStatus `json:"status"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	QueuedAt       *time.Time      `json:"queued_at,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
}

type ExecutionInput struct {
	PluginIDs []string       `json:"plugin_ids"`
	Input     map[string]any `json:"input"`
}

type ExecutionTarget struct {
	PluginID string `json:"plugin_id"`
}

func (e Execution) HasPlugins() bool { return len(e.PluginIDs) > 0 }
func (e Execution) HasInput() bool   { return e.InputJSON != nil }

func (e Execution) Scope() ResourceScope {
	return ResourceScope{TenantID: e.TenantID, ProjectID: e.ProjectID}.Normalize()
}
