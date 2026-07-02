package model

import "time"

type ExecutionAttemptStatus string

const (
	ExecutionAttemptRunning  ExecutionAttemptStatus = "Running"
	ExecutionAttemptSuccess  ExecutionAttemptStatus = "Success"
	ExecutionAttemptFailed   ExecutionAttemptStatus = "Failed"
	ExecutionAttemptCanceled ExecutionAttemptStatus = "Canceled"
)

type ExecutionAttempt struct {
	TenantID     string                 `json:"tenant_id"`
	ProjectID    string                 `json:"project_id"`
	ID           string                 `json:"id"`
	ExecutionID  string                 `json:"execution_id"`
	AttemptNo    int                    `json:"attempt_no"`
	WorkerID     string                 `json:"worker_id"`
	LeaseID      string                 `json:"lease_id,omitempty"`
	HeartbeatAt  *time.Time             `json:"heartbeat_at,omitempty"`
	LeaseUntil   *time.Time             `json:"lease_until,omitempty"`
	Status       ExecutionAttemptStatus `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	FinishedAt   *time.Time             `json:"finished_at,omitempty"`
}
