package service

import "time"

type WorkerHeartbeat struct {
	WorkerID    string    `json:"worker_id"`
	ExecutionID string    `json:"execution_id"`
	AttemptID   string    `json:"attempt_id"`
	LeaseID     string    `json:"lease_id"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
	LeaseUntil  time.Time `json:"lease_until"`
}
