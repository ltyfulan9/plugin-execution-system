package queue

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrPostgresQueueNotImplemented = errors.New("postgres queue adapter requires a registered postgres database/sql driver and production wiring")

type PostgresQueue struct{ db *sql.DB }

func NewPostgresQueue(db *sql.DB) *PostgresQueue { return &PostgresQueue{db: db} }

func (q *PostgresQueue) Enqueue(ctx context.Context, opts EnqueueOptions) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO task_queue (tenant_id, project_id, task_id, status, available_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'ready', COALESCE(NULLIF($4, TIMESTAMPTZ '0001-01-01'), now()), now(), now())
		ON CONFLICT (tenant_id, project_id, task_id) DO NOTHING`, opts.TenantID, opts.ProjectID, opts.TaskID, opts.AvailableAt)
	return err
}

func (q *PostgresQueue) LeaseNext(ctx context.Context, opts LeaseOptions) ([]TaskRef, error) {
	if opts.MaxItems <= 0 {
		opts.MaxItems = 1
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 30 * time.Second
	}
	rows, err := q.db.QueryContext(ctx, `
		WITH picked AS (
			SELECT id FROM task_queue
			WHERE status = 'ready' AND available_at <= now()
			ORDER BY available_at ASC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE task_queue q
		SET status='leased', worker_id=$2, lease_id=gen_random_uuid()::text,
		    lease_until=now() + ($3 * interval '1 second'),
		    attempt_no=attempt_no + 1, updated_at=now()
		FROM picked WHERE q.id = picked.id
		RETURNING q.tenant_id, q.project_id, q.task_id, q.lease_id, q.attempt_no, q.available_at, q.lease_until, q.created_at`, opts.MaxItems, opts.WorkerID, int(opts.LeaseDuration.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TaskRef{}
	for rows.Next() {
		var t TaskRef
		if err := rows.Scan(&t.TenantID, &t.ProjectID, &t.TaskID, &t.LeaseID, &t.AttemptNo, &t.VisibleAt, &t.LeaseUntil, &t.EnqueuedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (q *PostgresQueue) Heartbeat(ctx context.Context, taskID, leaseID, workerID string, extendBy time.Duration) error {
	res, err := q.db.ExecContext(ctx, `UPDATE task_queue SET lease_until=now()+($4 * interval '1 second'), heartbeat_at=now(), updated_at=now() WHERE task_id=$1 AND lease_id=$2 AND worker_id=$3 AND status='leased'`, taskID, leaseID, workerID, int(extendBy.Seconds()))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (q *PostgresQueue) Ack(ctx context.Context, taskID, leaseID, workerID string) error {
	_, err := q.db.ExecContext(ctx, `UPDATE task_queue SET status='acked', updated_at=now() WHERE task_id=$1 AND lease_id=$2 AND worker_id=$3`, taskID, leaseID, workerID)
	return err
}
func (q *PostgresQueue) Nack(ctx context.Context, taskID, leaseID, workerID string, opts NackOptions) error {
	status := "ready"
	if opts.DLQ || !opts.Retryable {
		status = "dlq"
	}
	next := opts.NextVisible
	if next.IsZero() {
		next = time.Now().UTC().Add(opts.Backoff)
	}
	_, err := q.db.ExecContext(ctx, `
		UPDATE task_queue
		SET status = CASE
			WHEN $7::int > 0 AND attempt_no >= $7::int THEN 'dlq'
			ELSE $4
		END,
		available_at=$5, last_error=$6, lease_id='', lease_until=NULL, worker_id='', updated_at=now()
		WHERE task_id=$1 AND lease_id=$2 AND worker_id=$3`, taskID, leaseID, workerID, status, next, opts.Reason, opts.MaxAttempts)
	return err
}
func (q *PostgresQueue) ReclaimExpiredLeases(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	res, err := q.db.ExecContext(ctx, `
		WITH expired AS (
			SELECT id FROM task_queue WHERE status='leased' AND lease_until < $1 ORDER BY lease_until ASC FOR UPDATE SKIP LOCKED LIMIT $2
		)
		UPDATE task_queue q SET status='ready', available_at=now(), lease_id='', worker_id='', lease_until=NULL, updated_at=now()
		FROM expired WHERE q.id=expired.id`, now, limit)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
func (q *PostgresQueue) MoveToDLQ(ctx context.Context, taskID, reason string) error {
	_, err := q.db.ExecContext(ctx, `UPDATE task_queue SET status='dlq', last_error=$2, updated_at=now() WHERE task_id=$1`, taskID, reason)
	return err
}
func (q *PostgresQueue) Depth(ctx context.Context, tenantID, projectID string) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT count(*) FROM task_queue WHERE tenant_id=$1 AND project_id=$2 AND status IN ('ready','leased')`, tenantID, projectID).Scan(&n)
	return n, err
}
