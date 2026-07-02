//go:build integration && pgx

package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/queue"
)

func TestPostgresMetadataAndQueueIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := OpenDB(ctx, OpenOptions{DriverName: "pgx", DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if os.Getenv("POSTGRES_AUTO_MIGRATE") == "true" {
		if err := Migrate(ctx, db, "../../../migrations/postgres/001_enterprise_metadata.sql"); err != nil {
			t.Fatal(err)
		}
	}
	seedScope(t, ctx, db)
	store := NewMetadataStore(db)
	now := time.Now().UTC()
	exec := model.Execution{TenantID: "tenant_it", ProjectID: "project_it", ID: "exec_it_1", UserID: "user_it", PluginIDs: []string{"plugin_it"}, InputJSON: map[string]any{"hello": "world"}, InputHash: "hash-input", PluginIDsHash: "hash-plugins", IdempotencyKey: "idem-it", Status: model.ExecutionStatusQueued, CreatedAt: now, QueuedAt: &now}
	event := model.ExecutionEvent{TenantID: exec.TenantID, ProjectID: exec.ProjectID, ID: "evt_it_1", ExecutionID: exec.ID, Type: model.ExecutionEventQueued, Status: string(model.ExecutionStatusQueued), CreatedAt: now}
	audit := model.AuditLog{TenantID: exec.TenantID, ProjectID: exec.ProjectID, ID: "audit_it_1", ActorID: exec.UserID, Action: model.AuditExecutionCreated, ResourceType: model.AuditResourceExecution, ResourceID: exec.ID, Decision: model.AuditDecisionAllow, Message: "created", CreatedAt: now}
	created, didCreate, err := store.CreateExecutionAndEnqueue(ctx, exec, []model.ExecutionEvent{event}, audit, now)
	if err != nil {
		t.Fatal(err)
	}
	if !didCreate || created.ID != exec.ID {
		t.Fatalf("expected create, got %#v created=%v", created, didCreate)
	}
	created2, didCreate2, err := store.CreateExecutionAndEnqueue(ctx, exec, []model.ExecutionEvent{event}, audit, now)
	if err != nil {
		t.Fatal(err)
	}
	if didCreate2 || created2.ID != exec.ID {
		t.Fatalf("expected idempotent existing execution, got %#v created=%v", created2, didCreate2)
	}
	q := queue.NewPostgresQueue(db)
	leased, err := q.LeaseNext(ctx, queue.LeaseOptions{WorkerID: "worker-it", MaxItems: 1, LeaseDuration: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if len(leased) != 1 || leased[0].TaskID != exec.ID || leased[0].LeaseID == "" {
		t.Fatalf("expected leased task, got %#v", leased)
	}
	if err := q.Heartbeat(ctx, exec.ID, leased[0].LeaseID, "worker-it", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := q.Ack(ctx, exec.ID, leased[0].LeaseID, "worker-it"); err != nil {
		t.Fatal(err)
	}
}

func seedScope(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `INSERT INTO tenants (id,name) VALUES ('tenant_it','Integration Tenant') ON CONFLICT DO NOTHING`)
	_, _ = db.ExecContext(ctx, `INSERT INTO projects (tenant_id,id,name) VALUES ('tenant_it','project_it','Integration Project') ON CONFLICT DO NOTHING`)
	_, _ = db.ExecContext(ctx, `INSERT INTO users (id,tenant_id,project_id,username,role,token_hash,created_at,updated_at) VALUES ('user_it','tenant_it','project_it','it','admin','hash',now(),now()) ON CONFLICT DO NOTHING`)
}
