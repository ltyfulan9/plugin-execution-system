package model

import "time"

type ExecutionEventType string

const (
	ExecutionEventCreated        ExecutionEventType = "ExecutionCreated"
	ExecutionEventQueued         ExecutionEventType = "ExecutionQueued"
	ExecutionEventStarted        ExecutionEventType = "ExecutionStarted"
	ExecutionEventPluginStarted  ExecutionEventType = "PluginStarted"
	ExecutionEventPluginFinished ExecutionEventType = "PluginFinished"
	ExecutionEventFinished       ExecutionEventType = "ExecutionFinished"
	ExecutionEventCanceled       ExecutionEventType = "ExecutionCanceled"
	ExecutionEventRecovered      ExecutionEventType = "ExecutionRecovered"
	ExecutionEventFailed         ExecutionEventType = "ExecutionFailed"
)

type ExecutionEvent struct {
	TenantID    string             `json:"tenant_id"`
	ProjectID   string             `json:"project_id"`
	ID          string             `json:"id"`
	ExecutionID string             `json:"execution_id"`
	PluginID    string             `json:"plugin_id,omitempty"`
	Type        ExecutionEventType `json:"type"`
	Status      string             `json:"status,omitempty"`
	Message     string             `json:"message,omitempty"`
	Detail      map[string]any     `json:"detail,omitempty"`
	RequestID   string             `json:"request_id,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

func (e ExecutionEvent) Scope() ResourceScope {
	return ResourceScope{TenantID: e.TenantID, ProjectID: e.ProjectID}.Normalize()
}
