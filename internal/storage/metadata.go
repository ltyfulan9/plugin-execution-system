package storage

import (
	"context"
	"time"

	"plugin-execution-system/internal/model"
)

type TxContext interface{ context.Context }

// MetadataStore is the enterprise source-of-truth contract. Production
// implementations must provide transactional state changes and append-only events.
//
// Important: task state and queue state must be updated in the same metadata
// transaction whenever possible. This prevents the classic split-brain failure
// where executions are committed but never enqueued, or queue rows exist without
// a matching current-state row.
type MetadataStore interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// CreateExecutionAndEnqueue is the production creation path. It must atomically
	// write current state, append a creation event, append audit, and insert the
	// durable queue row. Idempotency must be enforced by a database unique constraint.
	CreateExecutionAndEnqueue(ctx context.Context, e model.Execution, events []model.ExecutionEvent, audit model.AuditLog, availableAt time.Time) (model.Execution, bool, error)

	// Legacy lower-level primitives are kept for adapters and tests, but new
	// production code should prefer CreateExecutionAndEnqueue / transition methods.
	CreateExecutionWithEvent(ctx context.Context, e model.Execution, events []model.ExecutionEvent, audit model.AuditLog) (model.Execution, bool, error)
	TransitionExecutionWithEvent(ctx context.Context, id string, from, to model.ExecutionStatus, event model.ExecutionEvent, audit model.AuditLog) error
	AppendAttempt(ctx context.Context, attempt model.ExecutionAttempt, event model.ExecutionEvent) error
	AppendResultAndFinalize(ctx context.Context, executionID string, results []model.ExecutionResult, final model.ExecutionStatus, event model.ExecutionEvent, audit model.AuditLog) error
}
