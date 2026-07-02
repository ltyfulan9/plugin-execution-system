package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"plugin-execution-system/internal/model"
)

// NewPostgresRepositories wires all service-facing repositories to Postgres.
// This is the production metadata-store path. The local-json repository remains
// only a dev/test adapter.
func NewPostgresRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Auth:      &postgresAuthRepository{db: db},
		Plugin:    &postgresPluginRepository{db: db},
		Registry:  &postgresRegistryRepository{db: db},
		Execution: &postgresExecutionRepository{db: db},
		Result:    &postgresResultRepository{db: db},
		Audit:     &postgresAuditRepository{db: db},
		Event:     &postgresEventRepository{db: db},
		Attempt:   &postgresAttemptRepository{db: db},
		Webhook:   &postgresWebhookRepository{db: db},
	}
}

func pgJSON(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}
func pgJSONArray(v any) ([]byte, error) {
	if v == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(v)
}
func pgUnmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}

// Auth.
type postgresAuthRepository struct{ db *sql.DB }

func (r *postgresAuthRepository) CreateUser(u model.User) error {
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO users (id, tenant_id, project_id, username, role, token_hash, password_hash, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, u.ID, u.TenantID, u.ProjectID, u.Username, u.Role, u.TokenHash, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	return err
}
func (r *postgresAuthRepository) GetUserByID(id string) (model.User, bool, error) {
	return r.getUser(`id=$1`, id)
}
func (r *postgresAuthRepository) GetUserByTokenHash(hash string) (model.User, bool, error) {
	return r.getUser(`token_hash=$1`, hash)
}
func (r *postgresAuthRepository) GetUserByUsername(username string) (model.User, bool, error) {
	return r.getUser(`username=$1`, username)
}
func (r *postgresAuthRepository) getUser(where string, arg any) (model.User, bool, error) {
	row := r.db.QueryRowContext(context.Background(), `SELECT id, tenant_id, project_id, username, role, token_hash, COALESCE(password_hash,''), created_at, updated_at FROM users WHERE `+where+` LIMIT 1`, arg)
	var u model.User
	err := row.Scan(&u.ID, &u.TenantID, &u.ProjectID, &u.Username, &u.Role, &u.TokenHash, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.User{}, false, nil
	}
	return u, err == nil, err
}
func (r *postgresAuthRepository) ListUsers() ([]model.User, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT id, tenant_id, project_id, username, role, token_hash, COALESCE(password_hash,''), created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.ProjectID, &u.Username, &u.Role, &u.TokenHash, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Plugin.
type postgresPluginRepository struct{ db *sql.DB }

func (r *postgresPluginRepository) Create(p model.Plugin) error { return r.upsert(p) }
func (r *postgresPluginRepository) Update(p model.Plugin) error { return r.upsert(p) }
func (r *postgresPluginRepository) upsert(p model.Plugin) error {
	args, _ := pgJSONArray(p.Args)
	caps, _ := pgJSONArray(p.Capabilities)
	permissions, _ := pgJSON(map[string]any{"network": p.NetworkPolicy, "env": p.Env})
	resources, _ := pgJSON(map[string]any{"memory": p.MemoryLimit, "cpu": p.CPULimit, "pids": p.PIDsLimit})
	secretRefs, _ := pgJSON(p.SecretRefs)
	provenance, _ := pgJSON(map[string]any{})
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO plugins (id, tenant_id, project_id, name, version, api_version, runtime_type, protocol, command, args_json, image, work_dir, capabilities, permissions, resources, secret_refs, checksum, signature, provenance, status, error_message, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,'',$18,$19,$20,$21,$22) ON CONFLICT (tenant_id, project_id, name, version) DO UPDATE SET api_version=EXCLUDED.api_version, runtime_type=EXCLUDED.runtime_type, protocol=EXCLUDED.protocol, command=EXCLUDED.command, args_json=EXCLUDED.args_json, image=EXCLUDED.image, work_dir=EXCLUDED.work_dir, capabilities=EXCLUDED.capabilities, permissions=EXCLUDED.permissions, resources=EXCLUDED.resources, secret_refs=EXCLUDED.secret_refs, checksum=EXCLUDED.checksum, status=EXCLUDED.status, error_message=EXCLUDED.error_message, updated_at=EXCLUDED.updated_at`, p.ID, p.TenantID, p.ProjectID, p.Name, p.Version, p.APIVersion, p.EntryType, p.Protocol, p.Command, args, p.Image, p.WorkDir, caps, permissions, resources, secretRefs, p.Checksum, provenance, p.Status, p.ErrorMessage, p.CreatedAt, p.UpdatedAt)
	return err
}
func (r *postgresPluginRepository) GetByID(id string) (model.Plugin, bool, error) {
	return r.get(`id=$1`, id)
}
func (r *postgresPluginRepository) GetByNameVersion(name, version string) (model.Plugin, bool, error) {
	return r.get(`name=$1 AND version=$2`, name, version)
}
func (r *postgresPluginRepository) get(where string, args ...any) (model.Plugin, bool, error) {
	row := r.db.QueryRowContext(context.Background(), pluginSelectSQL+` WHERE `+where+` LIMIT 1`, args...)
	p, err := scanPlugin(row)
	if err == sql.ErrNoRows {
		return model.Plugin{}, false, nil
	}
	return p, err == nil, err
}

const pluginSelectSQL = `SELECT id, tenant_id, project_id, name, version, COALESCE(description,''), runtime_type, COALESCE(command,''), args_json, COALESCE(image,''), COALESCE(work_dir,''), status, COALESCE(error_message,''), api_version, protocol, capabilities, COALESCE(checksum,''), secret_refs, created_at, updated_at FROM plugins`

func scanPlugin(row interface{ Scan(...any) error }) (model.Plugin, error) {
	var p model.Plugin
	var argsB, capsB, secretRefsB []byte
	err := row.Scan(&p.ID, &p.TenantID, &p.ProjectID, &p.Name, &p.Version, &p.Description, &p.EntryType, &p.Command, &argsB, &p.Image, &p.WorkDir, &p.Status, &p.ErrorMessage, &p.APIVersion, &p.Protocol, &capsB, &p.Checksum, &secretRefsB, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return model.Plugin{}, err
	}
	_ = pgUnmarshal(argsB, &p.Args)
	_ = pgUnmarshal(capsB, &p.Capabilities)
	_ = pgUnmarshal(secretRefsB, &p.SecretRefs)
	return p, nil
}
func (r *postgresPluginRepository) List() ([]model.Plugin, error) {
	rows, err := r.db.QueryContext(context.Background(), pluginSelectSQL+` ORDER BY tenant_id, project_id, name, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Plugin{}
	for rows.Next() {
		p, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *postgresPluginRepository) UpdateStatus(id string, status model.PluginStatus) error {
	_, err := r.db.ExecContext(context.Background(), `UPDATE plugins SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}
func (r *postgresPluginRepository) UpdateError(id, msg string) error {
	_, err := r.db.ExecContext(context.Background(), `UPDATE plugins SET status=$2,error_message=$3,updated_at=now() WHERE id=$1`, id, model.PluginStatusError, msg)
	return err
}
func (r *postgresPluginRepository) MarkRemoved(id string) error {
	return r.UpdateStatus(id, model.PluginStatusRemoved)
}

// Registry.
type postgresRegistryRepository struct{ db *sql.DB }

func (r *postgresRegistryRepository) UpsertRegistryRecord(rec model.PluginRegistryRecord) error {
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO plugin_registry (plugin_id,name,version,source_path,manifest_hash,last_seen_at,synced_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (plugin_id) DO UPDATE SET source_path=EXCLUDED.source_path, manifest_hash=EXCLUDED.manifest_hash, last_seen_at=EXCLUDED.last_seen_at, synced_at=EXCLUDED.synced_at`, rec.PluginID, rec.Name, rec.Version, rec.SourcePath, rec.ManifestHash, rec.LastSeenAt, rec.SyncedAt)
	return err
}
func (r *postgresRegistryRepository) GetByPluginID(id string) (model.PluginRegistryRecord, bool, error) {
	row := r.db.QueryRowContext(context.Background(), `SELECT plugin_id,name,version,source_path,manifest_hash,last_seen_at,synced_at FROM plugin_registry WHERE plugin_id=$1`, id)
	var rec model.PluginRegistryRecord
	err := row.Scan(&rec.PluginID, &rec.Name, &rec.Version, &rec.SourcePath, &rec.ManifestHash, &rec.LastSeenAt, &rec.SyncedAt)
	if err == sql.ErrNoRows {
		return model.PluginRegistryRecord{}, false, nil
	}
	return rec, err == nil, err
}
func (r *postgresRegistryRepository) ListRegistryRecords() ([]model.PluginRegistryRecord, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT plugin_id,name,version,source_path,manifest_hash,last_seen_at,synced_at FROM plugin_registry ORDER BY name,version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PluginRegistryRecord{}
	for rows.Next() {
		var rec model.PluginRegistryRecord
		if err := rows.Scan(&rec.PluginID, &rec.Name, &rec.Version, &rec.SourcePath, &rec.ManifestHash, &rec.LastSeenAt, &rec.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// Execution.
type postgresExecutionRepository struct{ db *sql.DB }

func (r *postgresExecutionRepository) Create(e model.Execution) error {
	_, _, err := r.CreateWithIdempotency(e)
	return err
}
func (r *postgresExecutionRepository) CreateWithIdempotency(e model.Execution) (model.Execution, bool, error) {
	pluginIDs, _ := pgJSONArray(e.PluginIDs)
	input, _ := pgJSON(e.InputJSON)
	var out model.Execution
	var pluginB, inputB []byte
	err := r.db.QueryRowContext(context.Background(), `INSERT INTO executions (id, tenant_id, project_id, user_id, plugin_ids_json, input_json, input_hash, plugin_ids_hash, idempotency_key, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11) ON CONFLICT (tenant_id, project_id, user_id, idempotency_key) WHERE idempotency_key <> '' DO UPDATE SET idempotency_key=executions.idempotency_key RETURNING id,tenant_id,project_id,user_id,plugin_ids_json,input_json,input_hash,plugin_ids_hash,idempotency_key,status,COALESCE(error_message,''),created_at,queued_at,started_at,finished_at`, e.ID, e.TenantID, e.ProjectID, e.UserID, pluginIDs, input, e.InputHash, e.PluginIDsHash, e.IdempotencyKey, e.Status, e.CreatedAt).Scan(&out.ID, &out.TenantID, &out.ProjectID, &out.UserID, &pluginB, &inputB, &out.InputHash, &out.PluginIDsHash, &out.IdempotencyKey, &out.Status, &out.ErrorMessage, &out.CreatedAt, &out.QueuedAt, &out.StartedAt, &out.FinishedAt)
	if err != nil {
		return model.Execution{}, false, err
	}
	_ = pgUnmarshal(pluginB, &out.PluginIDs)
	_ = pgUnmarshal(inputB, &out.InputJSON)
	return out, out.ID == e.ID, nil
}
func (r *postgresExecutionRepository) Update(e model.Execution) error {
	pluginIDs, _ := pgJSONArray(e.PluginIDs)
	input, _ := pgJSON(e.InputJSON)
	_, err := r.db.ExecContext(context.Background(), `UPDATE executions SET plugin_ids_json=$2,input_json=$3,input_hash=$4,plugin_ids_hash=$5,idempotency_key=$6,status=$7,error_message=$8,queued_at=$9,started_at=$10,finished_at=$11,updated_at=now() WHERE id=$1`, e.ID, pluginIDs, input, e.InputHash, e.PluginIDsHash, e.IdempotencyKey, e.Status, e.ErrorMessage, e.QueuedAt, e.StartedAt, e.FinishedAt)
	return err
}
func (r *postgresExecutionRepository) GetByID(id string) (model.Execution, bool, error) {
	return r.get(`id=$1`, id)
}
func (r *postgresExecutionRepository) FindByIdempotencyKey(userID, key string) (model.Execution, bool, error) {
	return r.get(`user_id=$1 AND idempotency_key=$2`, userID, key)
}
func (r *postgresExecutionRepository) get(where string, args ...any) (model.Execution, bool, error) {
	row := r.db.QueryRowContext(context.Background(), executionSelectSQL+` WHERE `+where+` LIMIT 1`, args...)
	e, err := scanExecution(row)
	if err == sql.ErrNoRows {
		return model.Execution{}, false, nil
	}
	return e, err == nil, err
}

const executionSelectSQL = `SELECT id,tenant_id,project_id,user_id,plugin_ids_json,input_json,input_hash,plugin_ids_hash,idempotency_key,status,COALESCE(error_message,''),created_at,queued_at,started_at,finished_at FROM executions`

func scanExecution(row interface{ Scan(...any) error }) (model.Execution, error) {
	var e model.Execution
	var pluginB, inputB []byte
	err := row.Scan(&e.ID, &e.TenantID, &e.ProjectID, &e.UserID, &pluginB, &inputB, &e.InputHash, &e.PluginIDsHash, &e.IdempotencyKey, &e.Status, &e.ErrorMessage, &e.CreatedAt, &e.QueuedAt, &e.StartedAt, &e.FinishedAt)
	if err != nil {
		return model.Execution{}, err
	}
	_ = pgUnmarshal(pluginB, &e.PluginIDs)
	_ = pgUnmarshal(inputB, &e.InputJSON)
	return e, nil
}
func (r *postgresExecutionRepository) ListByUserID(userID string) ([]model.Execution, error) {
	return r.list(`user_id=$1`, userID)
}
func (r *postgresExecutionRepository) ListAll() ([]model.Execution, error) { return r.list(`true`) }
func (r *postgresExecutionRepository) ListByScope(scope model.ResourceScope) ([]model.Execution, error) {
	scope = scope.Normalize()
	return r.list(`tenant_id=$1 AND project_id=$2`, scope.TenantID, scope.ProjectID)
}
func (r *postgresExecutionRepository) ListByStatuses(statuses ...model.ExecutionStatus) ([]model.Execution, error) {
	if len(statuses) == 0 {
		return r.ListAll()
	}
	placeholders := make([]string, len(statuses))
	args := make([]any, len(statuses))
	for i, s := range statuses {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}
	return r.list(`status IN (`+strings.Join(placeholders, ",")+")", args...)
}
func (r *postgresExecutionRepository) list(where string, args ...any) ([]model.Execution, error) {
	rows, err := r.db.QueryContext(context.Background(), executionSelectSQL+` WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Execution{}
	for rows.Next() {
		e, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r *postgresExecutionRepository) UpdateStatus(id string, status model.ExecutionStatus, errMsg string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(context.Background(), `UPDATE executions SET status=$2,error_message=$3,queued_at=CASE WHEN $2='Queued' THEN $4 ELSE queued_at END,started_at=CASE WHEN $2='Running' THEN $4 ELSE started_at END,finished_at=CASE WHEN $5 THEN $4 ELSE finished_at END,updated_at=$4 WHERE id=$1`, id, status, errMsg, now, model.IsFinalExecutionStatus(status))
	return err
}

// Results.
type postgresResultRepository struct{ db *sql.DB }

func (r *postgresResultRepository) Create(res model.ExecutionResult) error {
	return r.BatchCreate([]model.ExecutionResult{res})
}
func (r *postgresResultRepository) BatchCreate(results []model.ExecutionResult) error {
	for _, res := range results {
		out, _ := pgJSON(res.OutputJSON)
		_, err := r.db.ExecContext(context.Background(), `INSERT INTO execution_results (id,tenant_id,project_id,execution_id,plugin_id,plugin_name,status,output_json,error_message,stdout_preview,stderr_preview,exit_code,duration_ms,started_at,finished_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT (id) DO NOTHING`, res.ID, res.TenantID, res.ProjectID, res.ExecutionID, res.PluginID, res.PluginName, res.Status, out, res.ErrorMessage, res.StdoutPreview, res.StderrPreview, res.ExitCode, res.DurationMS, res.StartedAt, res.FinishedAt)
		if err != nil {
			return err
		}
	}
	return nil
}
func (r *postgresResultRepository) GetByExecutionID(id string) ([]model.ExecutionResult, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT id,tenant_id,project_id,execution_id,plugin_id,plugin_name,status,COALESCE(output_json,'{}'::jsonb),COALESCE(error_message,''),COALESCE(stdout_preview,''),COALESCE(stderr_preview,''),COALESCE(exit_code,0),COALESCE(duration_ms,0),started_at,finished_at FROM execution_results WHERE execution_id=$1 ORDER BY started_at ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ExecutionResult{}
	for rows.Next() {
		var res model.ExecutionResult
		var b []byte
		if err := rows.Scan(&res.ID, &res.TenantID, &res.ProjectID, &res.ExecutionID, &res.PluginID, &res.PluginName, &res.Status, &b, &res.ErrorMessage, &res.StdoutPreview, &res.StderrPreview, &res.ExitCode, &res.DurationMS, &res.StartedAt, &res.FinishedAt); err != nil {
			return nil, err
		}
		_ = pgUnmarshal(b, &res.OutputJSON)
		out = append(out, res)
	}
	return out, rows.Err()
}
func (r *postgresResultRepository) DeleteByExecutionID(id string) error {
	_, err := r.db.ExecContext(context.Background(), `DELETE FROM execution_results WHERE execution_id=$1`, id)
	return err
}

// Audit.
type postgresAuditRepository struct{ db *sql.DB }

func (r *postgresAuditRepository) Create(a model.AuditLog) error {
	detail, _ := pgJSON(a.DetailJSON)
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO audit_logs (id,tenant_id,project_id,actor_id,action,resource_type,resource_id,decision,reason,request_id,trace_id,plugin_digest,input_hash,result_hash,message,detail_json,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, a.ID, a.TenantID, a.ProjectID, a.ActorID, a.Action, a.ResourceType, a.ResourceID, a.Decision, a.Reason, a.RequestID, a.TraceID, a.PluginDigest, a.InputHash, a.ResultHash, a.Message, detail, a.CreatedAt)
	return err
}
func (r *postgresAuditRepository) List() ([]model.AuditLog, error) { return r.list(`true`) }
func (r *postgresAuditRepository) ListByResource(t model.AuditResourceType, id string) ([]model.AuditLog, error) {
	return r.list(`resource_type=$1 AND resource_id=$2`, t, id)
}
func (r *postgresAuditRepository) ListByRequestID(requestID string) ([]model.AuditLog, error) {
	return r.list(`request_id=$1`, requestID)
}
func (r *postgresAuditRepository) list(where string, args ...any) ([]model.AuditLog, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT id,tenant_id,project_id,actor_id,action,resource_type,resource_id,decision,COALESCE(reason,''),COALESCE(request_id,''),COALESCE(trace_id,''),COALESCE(plugin_digest,''),COALESCE(input_hash,''),COALESCE(result_hash,''),message,detail_json,created_at FROM audit_logs WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AuditLog{}
	for rows.Next() {
		var a model.AuditLog
		var b []byte
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ProjectID, &a.ActorID, &a.Action, &a.ResourceType, &a.ResourceID, &a.Decision, &a.Reason, &a.RequestID, &a.TraceID, &a.PluginDigest, &a.InputHash, &a.ResultHash, &a.Message, &b, &a.CreatedAt); err != nil {
			return nil, err
		}
		_ = pgUnmarshal(b, &a.DetailJSON)
		out = append(out, a)
	}
	return out, rows.Err()
}

// Events.
type postgresEventRepository struct{ db *sql.DB }

func (r *postgresEventRepository) Create(event model.ExecutionEvent) error {
	detail, _ := pgJSON(event.Detail)
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO task_events (id,tenant_id,project_id,execution_id,plugin_id,type,status,message,detail_json,request_id,trace_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'',$11)`, event.ID, event.TenantID, event.ProjectID, event.ExecutionID, event.PluginID, event.Type, event.Status, event.Message, detail, event.RequestID, event.CreatedAt)
	return err
}
func (r *postgresEventRepository) ListByExecutionID(executionID string) ([]model.ExecutionEvent, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT id,tenant_id,project_id,execution_id,COALESCE(plugin_id,''),type,COALESCE(status,''),COALESCE(message,''),detail_json,COALESCE(request_id,''),created_at FROM task_events WHERE execution_id=$1 ORDER BY created_at ASC`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ExecutionEvent{}
	for rows.Next() {
		var e model.ExecutionEvent
		var b []byte
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ProjectID, &e.ExecutionID, &e.PluginID, &e.Type, &e.Status, &e.Message, &b, &e.RequestID, &e.CreatedAt); err != nil {
			return nil, err
		}
		_ = pgUnmarshal(b, &e.Detail)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Attempts.
type postgresAttemptRepository struct{ db *sql.DB }

func (r *postgresAttemptRepository) Create(a model.ExecutionAttempt) error { return r.upsert(a) }
func (r *postgresAttemptRepository) Update(a model.ExecutionAttempt) error { return r.upsert(a) }
func (r *postgresAttemptRepository) upsert(a model.ExecutionAttempt) error {
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO task_attempts (id,tenant_id,project_id,execution_id,attempt_no,worker_id,lease_id,status,heartbeat_at,lease_until,started_at,finished_at,error_message) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (execution_id, attempt_no) DO UPDATE SET status=EXCLUDED.status, heartbeat_at=EXCLUDED.heartbeat_at, finished_at=EXCLUDED.finished_at, error_message=EXCLUDED.error_message`, a.ID, a.TenantID, a.ProjectID, a.ExecutionID, a.AttemptNo, a.WorkerID, a.LeaseID, a.Status, a.HeartbeatAt, a.LeaseUntil, a.StartedAt, a.FinishedAt, a.ErrorMessage)
	return err
}
func (r *postgresAttemptRepository) ListByExecutionID(executionID string) ([]model.ExecutionAttempt, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT id,tenant_id,project_id,execution_id,attempt_no,worker_id,COALESCE(lease_id,''),status,heartbeat_at,started_at,finished_at,COALESCE(error_message,'') FROM task_attempts WHERE execution_id=$1 ORDER BY attempt_no ASC`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ExecutionAttempt{}
	for rows.Next() {
		var a model.ExecutionAttempt
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ProjectID, &a.ExecutionID, &a.AttemptNo, &a.WorkerID, &a.LeaseID, &a.Status, &a.HeartbeatAt, &a.StartedAt, &a.FinishedAt, &a.ErrorMessage); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *postgresAttemptRepository) NextAttemptNo(executionID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(context.Background(), `SELECT COALESCE(max(attempt_no),0)+1 FROM task_attempts WHERE execution_id=$1`, executionID).Scan(&n)
	return n, err
}

// Webhooks.
type postgresWebhookRepository struct{ db *sql.DB }

func (r *postgresWebhookRepository) CreateEndpoint(e model.WebhookEndpoint) error {
	return r.upsertEndpoint(e)
}
func (r *postgresWebhookRepository) UpdateEndpoint(e model.WebhookEndpoint) error {
	return r.upsertEndpoint(e)
}
func (r *postgresWebhookRepository) upsertEndpoint(e model.WebhookEndpoint) error {
	events, _ := pgJSONArray(e.Events)
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO webhook_endpoints (id,tenant_id,project_id,name,url,secret,events_json,status,created_by,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,url=EXCLUDED.url,events_json=EXCLUDED.events_json,status=EXCLUDED.status,updated_at=EXCLUDED.updated_at`, e.ID, e.TenantID, e.ProjectID, e.Name, e.URL, e.Secret, events, e.Status, e.CreatedBy, e.CreatedAt, e.UpdatedAt)
	return err
}
func (r *postgresWebhookRepository) GetEndpointByID(id string) (model.WebhookEndpoint, bool, error) {
	row := r.db.QueryRowContext(context.Background(), `SELECT id,tenant_id,project_id,name,url,secret,events_json,status,created_by,created_at,updated_at FROM webhook_endpoints WHERE id=$1`, id)
	e, err := scanWebhookEndpoint(row)
	if err == sql.ErrNoRows {
		return model.WebhookEndpoint{}, false, nil
	}
	return e, err == nil, err
}
func (r *postgresWebhookRepository) ListEndpoints() ([]model.WebhookEndpoint, error) {
	return r.listEndpoints(`true`)
}
func (r *postgresWebhookRepository) ListEnabledEndpoints() ([]model.WebhookEndpoint, error) {
	return r.listEndpoints(`status=$1`, model.WebhookStatusEnabled)
}
func scanWebhookEndpoint(row interface{ Scan(...any) error }) (model.WebhookEndpoint, error) {
	var e model.WebhookEndpoint
	var b []byte
	err := row.Scan(&e.ID, &e.TenantID, &e.ProjectID, &e.Name, &e.URL, &e.Secret, &b, &e.Status, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return model.WebhookEndpoint{}, err
	}
	_ = pgUnmarshal(b, &e.Events)
	return e, nil
}
func (r *postgresWebhookRepository) listEndpoints(where string, args ...any) ([]model.WebhookEndpoint, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT id,tenant_id,project_id,name,url,secret,events_json,status,created_by,created_at,updated_at FROM webhook_endpoints WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.WebhookEndpoint{}
	for rows.Next() {
		e, err := scanWebhookEndpoint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r *postgresWebhookRepository) DeleteEndpoint(id string) error {
	_, err := r.db.ExecContext(context.Background(), `DELETE FROM webhook_endpoints WHERE id=$1`, id)
	return err
}
func (r *postgresWebhookRepository) CreateDelivery(d model.WebhookDelivery) error {
	return r.upsertDelivery(d)
}
func (r *postgresWebhookRepository) UpdateDelivery(d model.WebhookDelivery) error {
	return r.upsertDelivery(d)
}
func (r *postgresWebhookRepository) upsertDelivery(d model.WebhookDelivery) error {
	payload, _ := pgJSON(d.PayloadJSON)
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO webhook_deliveries (id,tenant_id,project_id,webhook_id,event_id,event_type,target_url,status,attempt_no,max_attempts,next_retry_at,status_code,error,payload_json,created_at,delivered_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,attempt_no=EXCLUDED.attempt_no,max_attempts=EXCLUDED.max_attempts,next_retry_at=EXCLUDED.next_retry_at,status_code=EXCLUDED.status_code,error=EXCLUDED.error,payload_json=EXCLUDED.payload_json,delivered_at=EXCLUDED.delivered_at`, d.ID, d.TenantID, d.ProjectID, d.WebhookID, d.EventID, d.EventType, d.TargetURL, d.Status, d.AttemptNo, d.MaxAttempts, d.NextRetryAt, d.StatusCode, d.Error, payload, d.CreatedAt, d.DeliveredAt)
	return err
}
func (r *postgresWebhookRepository) ListDeliveries(webhookID string) ([]model.WebhookDelivery, error) {
	where := "true"
	args := []any{}
	if webhookID != "" {
		where = "webhook_id=$1"
		args = []any{webhookID}
	}
	rows, err := r.db.QueryContext(context.Background(), `SELECT id,tenant_id,project_id,webhook_id,event_id,event_type,target_url,status,attempt_no,max_attempts,next_retry_at,COALESCE(status_code,0),COALESCE(error,''),payload_json,created_at,delivered_at FROM webhook_deliveries WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.WebhookDelivery{}
	for rows.Next() {
		var d model.WebhookDelivery
		var payload []byte
		if err := rows.Scan(&d.ID, &d.TenantID, &d.ProjectID, &d.WebhookID, &d.EventID, &d.EventType, &d.TargetURL, &d.Status, &d.AttemptNo, &d.MaxAttempts, &d.NextRetryAt, &d.StatusCode, &d.Error, &payload, &d.CreatedAt, &d.DeliveredAt); err != nil {
			return nil, err
		}
		_ = pgUnmarshal(payload, &d.PayloadJSON)
		out = append(out, d)
	}
	return out, rows.Err()
}
