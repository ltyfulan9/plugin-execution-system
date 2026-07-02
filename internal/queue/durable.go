package queue

import (
	"context"
	"time"
)

type TaskRef struct {
	TenantID   string
	ProjectID  string
	TaskID     string
	LeaseID    string
	AttemptNo  int
	VisibleAt  time.Time
	LeaseUntil time.Time
	EnqueuedAt time.Time
}

type EnqueueOptions struct {
	TenantID       string
	ProjectID      string
	TaskID         string
	IdempotencyKey string
	AvailableAt    time.Time
}

type LeaseOptions struct {
	WorkerID          string
	MaxItems          int
	LeaseDuration     time.Duration
	VisibilityTimeout time.Duration
}

type NackOptions struct {
	Retryable   bool
	Backoff     time.Duration
	Reason      string
	DLQ         bool
	MaxAttempts int
	NextVisible time.Time
}

// DurableExecutionQueue is the production contract. Implementations must use a
// durable metadata store and must not rely on process memory as the source of truth.
type DurableExecutionQueue interface {
	Enqueue(ctx context.Context, opts EnqueueOptions) error
	LeaseNext(ctx context.Context, opts LeaseOptions) ([]TaskRef, error)
	Heartbeat(ctx context.Context, taskID, leaseID, workerID string, extendBy time.Duration) error
	Ack(ctx context.Context, taskID, leaseID, workerID string) error
	Nack(ctx context.Context, taskID, leaseID, workerID string, opts NackOptions) error
	ReclaimExpiredLeases(ctx context.Context, now time.Time, limit int) (int, error)
	MoveToDLQ(ctx context.Context, taskID, reason string) error
	Depth(ctx context.Context, tenantID, projectID string) (int64, error)
}
