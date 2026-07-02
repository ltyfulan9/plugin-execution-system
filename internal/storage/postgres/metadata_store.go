package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"plugin-execution-system/internal/model"
)

type MetadataStore struct{ db *sql.DB }

func NewMetadataStore(db *sql.DB) *MetadataStore { return &MetadataStore{db: db} }

func (s *MetadataStore) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type txKey struct{}

func txFrom(ctx context.Context, db *sql.DB) execer {
	if tx, _ := ctx.Value(txKey{}).(*sql.Tx); tx != nil {
		return tx
	}
	return db
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func jsonb(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// CreateExecutionAndEnqueue is the enterprise-safe create path: current state,
// business event, audit record, and durable queue row are written in one tx.
func (s *MetadataStore) CreateExecutionAndEnqueue(ctx context.Context, e model.Execution, evs []model.ExecutionEvent, audit model.AuditLog, availableAt time.Time) (model.Execution, bool, error) {
	var out model.Execution
	var created bool
	err := s.WithTx(ctx, func(txCtx context.Context) error {
		createdOrExisting, didCreate, err := s.CreateExecutionWithEvent(txCtx, e, evs, audit)
		if err != nil {
			return err
		}
		out = createdOrExisting
		created = didCreate
		if !didCreate {
			return nil
		}
		if availableAt.IsZero() {
			availableAt = time.Now().UTC()
		}
		_, err = txFrom(txCtx, s.db).ExecContext(txCtx, `
			INSERT INTO task_queue (tenant_id, project_id, task_id, status, available_at, created_at, updated_at)
			VALUES ($1, $2, $3, 'ready', $4, now(), now())
			ON CONFLICT (tenant_id, project_id, task_id) DO NOTHING`, e.TenantID, e.ProjectID, e.ID, availableAt)
		return err
	})
	return out, created, err
}

func (s *MetadataStore) CreateExecutionWithEvent(ctx context.Context, e model.Execution, evs []model.ExecutionEvent, audit model.AuditLog) (model.Execution, bool, error) {
	pluginIDs, err := jsonb(e.PluginIDs)
	if err != nil {
		return model.Execution{}, false, err
	}
	inputJSON, err := jsonb(e.InputJSON)
	if err != nil {
		return model.Execution{}, false, err
	}
	var existing model.Execution
	err = txFrom(ctx, s.db).QueryRowContext(ctx, `
		INSERT INTO executions (id, tenant_id, project_id, user_id, plugin_ids_json, input_json, input_hash, plugin_ids_hash, idempotency_key, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (tenant_id, project_id, user_id, idempotency_key) WHERE idempotency_key <> ''
		DO UPDATE SET idempotency_key = executions.idempotency_key
		RETURNING id, tenant_id, project_id, user_id, input_hash, plugin_ids_hash, idempotency_key, status, created_at`, e.ID, e.TenantID, e.ProjectID, e.UserID, pluginIDs, inputJSON, e.InputHash, e.PluginIDsHash, e.IdempotencyKey, e.Status, e.CreatedAt).Scan(&existing.ID, &existing.TenantID, &existing.ProjectID, &existing.UserID, &existing.InputHash, &existing.PluginIDsHash, &existing.IdempotencyKey, &existing.Status, &existing.CreatedAt)
	if err != nil {
		return model.Execution{}, false, err
	}
	created := existing.ID == e.ID
	if created {
		for _, ev := range evs {
			if err := insertEvent(ctx, txFrom(ctx, s.db), ev); err != nil {
				return model.Execution{}, false, err
			}
		}
		if err := insertAudit(ctx, txFrom(ctx, s.db), audit); err != nil {
			return model.Execution{}, false, err
		}
		return e, true, nil
	}
	return existing, false, nil
}

func (s *MetadataStore) TransitionExecutionWithEvent(ctx context.Context, id string, from, to model.ExecutionStatus, ev model.ExecutionEvent, audit model.AuditLog) error {
	return s.WithTx(ctx, func(txCtx context.Context) error {
		res, err := txFrom(txCtx, s.db).ExecContext(txCtx, `UPDATE executions SET status=$3, updated_at=now() WHERE id=$1 AND status=$2`, id, from, to)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		if err := insertEvent(txCtx, txFrom(txCtx, s.db), ev); err != nil {
			return err
		}
		return insertAudit(txCtx, txFrom(txCtx, s.db), audit)
	})
}
func (s *MetadataStore) AppendAttempt(ctx context.Context, attempt model.ExecutionAttempt, ev model.ExecutionEvent) error {
	return s.WithTx(ctx, func(txCtx context.Context) error {
		_, err := txFrom(txCtx, s.db).ExecContext(txCtx, `INSERT INTO task_attempts (id, tenant_id, project_id, execution_id, attempt_no, worker_id, lease_id, status, heartbeat_at, lease_until, started_at, finished_at, error_message) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, attempt.ID, attempt.TenantID, attempt.ProjectID, attempt.ExecutionID, attempt.AttemptNo, attempt.WorkerID, attempt.LeaseID, attempt.Status, attempt.HeartbeatAt, attempt.LeaseUntil, attempt.StartedAt, attempt.FinishedAt, attempt.ErrorMessage)
		if err != nil {
			return err
		}
		return insertEvent(txCtx, txFrom(txCtx, s.db), ev)
	})
}
func (s *MetadataStore) AppendResultAndFinalize(ctx context.Context, executionID string, results []model.ExecutionResult, final model.ExecutionStatus, ev model.ExecutionEvent, audit model.AuditLog) error {
	return s.WithTx(ctx, func(txCtx context.Context) error {
		for _, r := range results {
			out, err := jsonb(r.OutputJSON)
			if err != nil {
				return err
			}
			_, err = txFrom(txCtx, s.db).ExecContext(txCtx, `INSERT INTO execution_results (id, tenant_id, project_id, execution_id, plugin_id, plugin_name, status, output_json, error_message, stdout_preview, stderr_preview, exit_code, duration_ms, started_at, finished_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, r.ID, r.TenantID, r.ProjectID, r.ExecutionID, r.PluginID, r.PluginName, r.Status, out, r.ErrorMessage, r.StdoutPreview, r.StderrPreview, r.ExitCode, r.DurationMS, r.StartedAt, r.FinishedAt)
			if err != nil {
				return err
			}
		}
		if _, err := txFrom(txCtx, s.db).ExecContext(txCtx, `UPDATE executions SET status=$2, finished_at=now(), updated_at=now() WHERE id=$1`, executionID, final); err != nil {
			return err
		}
		if err := insertEvent(txCtx, txFrom(txCtx, s.db), ev); err != nil {
			return err
		}
		return insertAudit(txCtx, txFrom(txCtx, s.db), audit)
	})
}

func insertEvent(ctx context.Context, x execer, e model.ExecutionEvent) error {
	detail, err := jsonb(e.Detail)
	if err != nil {
		return err
	}
	_, err = x.ExecContext(ctx, `INSERT INTO task_events (id, tenant_id, project_id, execution_id, plugin_id, type, status, message, detail_json, request_id, trace_id, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, e.ID, e.TenantID, e.ProjectID, e.ExecutionID, e.PluginID, e.Type, e.Status, e.Message, detail, e.RequestID, "", e.CreatedAt)
	return err
}
func insertAudit(ctx context.Context, x execer, a model.AuditLog) error {
	detail, err := jsonb(a.DetailJSON)
	if err != nil {
		return err
	}
	_, err = x.ExecContext(ctx, `INSERT INTO audit_logs (id, tenant_id, project_id, actor_id, action, resource_type, resource_id, decision, reason, request_id, trace_id, plugin_digest, input_hash, result_hash, message, detail_json, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, a.ID, a.TenantID, a.ProjectID, a.ActorID, a.Action, a.ResourceType, a.ResourceID, a.Decision, a.Reason, a.RequestID, a.TraceID, a.PluginDigest, a.InputHash, a.ResultHash, a.Message, detail, a.CreatedAt)
	return err
}
