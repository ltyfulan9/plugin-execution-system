package model

import "time"

type PluginResultStatus string

const (
	PluginResultSuccess       PluginResultStatus = "Success"
	PluginResultFailed        PluginResultStatus = "Failed"
	PluginResultTimeout       PluginResultStatus = "Timeout"
	PluginResultInvalidOutput PluginResultStatus = "InvalidOutput"
	PluginResultRuntimeError  PluginResultStatus = "RuntimeError"
	PluginResultCanceled      PluginResultStatus = "Canceled"
)

type ExecutionResult struct {
	TenantID      string             `json:"tenant_id"`
	ProjectID     string             `json:"project_id"`
	ID            string             `json:"id"`
	ExecutionID   string             `json:"execution_id"`
	PluginID      string             `json:"plugin_id"`
	PluginName    string             `json:"plugin_name"`
	Status        PluginResultStatus `json:"status"`
	OutputJSON    map[string]any     `json:"output,omitempty"`
	ErrorMessage  string             `json:"error_message,omitempty"`
	StdoutPreview string             `json:"stdout_preview,omitempty"`
	StderrPreview string             `json:"stderr_preview,omitempty"`
	ExitCode      int                `json:"exit_code"`
	DurationMS    int64              `json:"duration_ms"`
	StartedAt     time.Time          `json:"started_at"`
	FinishedAt    time.Time          `json:"finished_at"`
}

type ExecutionSummary struct {
	ProjectID   string          `json:"project_id"`
	ID          string          `json:"id"`
	ExecutionID string          `json:"execution_id"`
	Status      ExecutionStatus `json:"status"`
	Total       int             `json:"total"`
	Success     int             `json:"success"`
	Failed      int             `json:"failed"`
	Timeout     int             `json:"timeout"`
	DurationMS  int64           `json:"duration_ms"`
}

func IsPluginResultSuccess(s PluginResultStatus) bool { return s == PluginResultSuccess }
