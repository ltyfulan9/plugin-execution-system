package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"plugin-execution-system/internal/event"
	"plugin-execution-system/internal/handler"
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/service"
	"plugin-execution-system/internal/storage"
	"plugin-execution-system/internal/worker"
)

type testApp struct {
	handler http.Handler
	pool    *worker.WorkerPool
}

type testEnvelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	Error     json.RawMessage `json:"error"`
	RequestID string          `json:"request_id"`
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	root := t.TempDir()
	storeDir := filepath.Join(root, "data")
	pluginDir := filepath.Join(root, "plugins")
	copyDir(t, filepath.Join("..", "..", "plugins"), pluginDir)

	store, err := storage.OpenJSONStore(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(store); err != nil {
		t.Fatal(err)
	}
	if err := storage.SeedDefaultUsers(store, "demo-token", "admin-token"); err != nil {
		t.Fatal(err)
	}

	repos := repository.NewRepositories(store)
	authSvc := service.NewAuthService(repos.Auth)
	auditSvc := service.NewAuditService(repos.Audit)
	eventSvc := service.NewExecutionEventService(repos.Event, event.NewBus())
	attemptSvc := service.NewExecutionAttemptService(repos.Attempt)
	webhookSvc := service.NewWebhookService(repos.Webhook).WithPrivateTargetsForTests(true)
	eventSvc.Subscribe("*", webhookSvc.HandleExecutionEvent)
	pluginSvc := service.NewPluginService(repos.Plugin, service.NewPluginStateMachine())
	manifestSvc := service.NewManifestService()
	registrySvc := service.NewRegistryService(manifestSvc, pluginSvc, repos.Registry, repos.Plugin)
	if _, err := registrySvc.ReloadPlugins(pluginDir); err != nil {
		t.Fatal(err)
	}
	queue := worker.NewExecutionQueue(16)
	execSvc := service.NewExecutionService(repos.Execution, pluginSvc, service.NewIdempotencyService(repos.Execution), service.NewExecutionStateMachine(), auditSvc, queue).WithEventService(eventSvc)
	resultSvc := service.NewResultService(repos.Result)
	runtimeSvc := service.NewRuntimeService(service.NewProcessRunner("python3", "python"), 65536)
	execWorker := worker.NewExecutionWorkerWithEvents(execSvc, pluginSvc, runtimeSvc, resultSvc, auditSvc, eventSvc, attemptSvc)
	pool := worker.NewWorkerPool(queue, execWorker, 2)
	pool.Start(context.Background())
	t.Cleanup(pool.Stop)

	r := NewRouter(Handlers{
		Auth:      handler.NewAuthHandler(authSvc),
		Plugin:    handler.NewPluginHandler(pluginSvc, registrySvc, pluginDir, auditSvc),
		Execution: handler.NewExecutionHandler(execSvc),
		Result:    handler.NewResultHandler(resultSvc, execSvc),
		Observe:   handler.NewExecutionObserveHandler(execSvc, eventSvc, attemptSvc),
		Audit:     handler.NewAuditHandler(auditSvc),
		Health:    handler.NewHealthHandler(pluginDir),
		Webhook:   handler.NewWebhookHandler(webhookSvc),
		AuthSvc:   authSvc,
	})
	return &testApp{handler: r, pool: pool}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, s, d)
			continue
		}
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(d, b, info.Mode()); err != nil {
			t.Fatal(err)
		}
	}
}

func doJSON(t *testing.T, app *testApp, method, path, token string, body any, extraHeaders map[string]string) (int, testEnvelope) {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	app.handler.ServeHTTP(w, req)
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("status=%d body=%s err=%v", w.Code, w.Body.String(), err)
	}
	return w.Code, env
}

func mustData[T any](t *testing.T, env testEnvelope) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		t.Fatalf("decode data: %v raw=%s", err, string(env.Data))
	}
	return out
}

func findPlugin(t *testing.T, plugins []model.Plugin, name string) model.Plugin {
	t.Helper()
	for _, p := range plugins {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("plugin %s not found in %#v", name, plugins)
	return model.Plugin{}
}

func enablePlugin(t *testing.T, app *testApp, p model.Plugin) model.Plugin {
	t.Helper()
	status, env := doJSON(t, app, http.MethodPost, "/api/plugins/"+p.ID+"/enable", "admin-token", nil, nil)
	if status != http.StatusOK || env.Code != response.CodeOK {
		t.Fatalf("enable plugin status=%d env=%+v", status, env)
	}
	return mustData[model.Plugin](t, env)
}

func waitExecutionFinal(t *testing.T, app *testApp, id string) model.Execution {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, env := doJSON(t, app, http.MethodGet, "/api/executions/"+id, "demo-token", nil, nil)
		if status != http.StatusOK {
			t.Fatalf("get execution status=%d env=%+v", status, env)
		}
		exec := mustData[model.Execution](t, env)
		if model.IsFinalExecutionStatus(exec.Status) {
			return exec
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("execution %s did not finish", id)
	return model.Execution{}
}

func TestAPIExecutionSuccessFlow(t *testing.T) {
	app := newTestApp(t)

	status, env := doJSON(t, app, http.MethodPost, "/api/plugins/reload", "demo-token", nil, nil)
	if status != http.StatusForbidden || env.Code != response.CodeForbidden {
		t.Fatalf("demo reload should be forbidden, status=%d env=%+v", status, env)
	}

	status, env = doJSON(t, app, http.MethodGet, "/api/v1/plugins", "demo-token", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("v1 list plugins status=%d env=%+v", status, env)
	}

	status, env = doJSON(t, app, http.MethodGet, "/api/plugins", "demo-token", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("list plugins status=%d env=%+v", status, env)
	}
	plugins := mustData[[]model.Plugin](t, env)
	echo := enablePlugin(t, app, findPlugin(t, plugins, "echo"))
	textStats := enablePlugin(t, app, findPlugin(t, plugins, "text_stats"))

	createBody := map[string]any{
		"plugin_ids": []string{echo.ID, textStats.ID},
		"input":      map[string]any{"text": "hello world"},
	}
	status, env = doJSON(t, app, http.MethodPost, "/api/executions", "demo-token", createBody, map[string]string{"Idempotency-Key": "api-success-1"})
	if status != http.StatusCreated {
		t.Fatalf("create execution status=%d env=%+v", status, env)
	}
	created := mustData[model.Execution](t, env)
	final := waitExecutionFinal(t, app, created.ID)
	if final.Status != model.ExecutionStatusSuccess {
		t.Fatalf("expected success, got %+v", final)
	}

	status, env = doJSON(t, app, http.MethodGet, "/api/executions/"+created.ID+"/results", "demo-token", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("results status=%d env=%+v", status, env)
	}
	results := mustData[[]model.ExecutionResult](t, env)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %#v", len(results), results)
	}

	status, env = doJSON(t, app, http.MethodGet, "/api/executions/"+created.ID+"/summary", "demo-token", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("summary status=%d env=%+v", status, env)
	}
	summary := mustData[model.ExecutionSummary](t, env)
	if summary.Status != model.ExecutionStatusSuccess || summary.Success != 2 || summary.Total != 2 {
		t.Fatalf("bad summary: %+v", summary)
	}

	status, env = doJSON(t, app, http.MethodGet, "/api/executions/"+created.ID+"/events", "demo-token", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("events status=%d env=%+v", status, env)
	}
	events := mustData[[]model.ExecutionEvent](t, env)
	if len(events) == 0 {
		t.Fatalf("expected execution events")
	}

	status, env = doJSON(t, app, http.MethodGet, "/api/executions/"+created.ID+"/attempts", "demo-token", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("attempts status=%d env=%+v", status, env)
	}
	attempts := mustData[[]model.ExecutionAttempt](t, env)
	if len(attempts) == 0 || attempts[0].Status != model.ExecutionAttemptSuccess {
		t.Fatalf("expected successful attempt, got %#v", attempts)
	}
}

func TestAPIIdempotencyAndPartialSuccess(t *testing.T) {
	app := newTestApp(t)
	_, env := doJSON(t, app, http.MethodGet, "/api/plugins", "admin-token", nil, nil)
	plugins := mustData[[]model.Plugin](t, env)
	echo := enablePlugin(t, app, findPlugin(t, plugins, "echo"))
	errorDemo := enablePlugin(t, app, findPlugin(t, plugins, "error_demo"))

	body := map[string]any{"plugin_ids": []string{echo.ID, errorDemo.ID}, "input": map[string]any{"text": "hello"}}
	status, env := doJSON(t, app, http.MethodPost, "/api/executions", "demo-token", body, map[string]string{"Idempotency-Key": "idem-partial"})
	if status != http.StatusCreated {
		t.Fatalf("create execution status=%d env=%+v", status, env)
	}
	first := mustData[model.Execution](t, env)
	final := waitExecutionFinal(t, app, first.ID)
	if final.Status != model.ExecutionStatusPartialSuccess {
		t.Fatalf("expected partial success, got %+v", final)
	}

	status, env = doJSON(t, app, http.MethodPost, "/api/executions", "demo-token", body, map[string]string{"Idempotency-Key": "idem-partial"})
	if status != http.StatusCreated {
		t.Fatalf("repeat execution status=%d env=%+v", status, env)
	}
	repeated := mustData[model.Execution](t, env)
	if repeated.ID != first.ID {
		t.Fatalf("expected same execution id, got first=%s repeated=%s", first.ID, repeated.ID)
	}

	conflictBody := map[string]any{"plugin_ids": []string{echo.ID, errorDemo.ID}, "input": map[string]any{"text": "different"}}
	status, env = doJSON(t, app, http.MethodPost, "/api/executions", "demo-token", conflictBody, map[string]string{"Idempotency-Key": "idem-partial"})
	if status != http.StatusConflict || env.Code != response.CodeIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, status=%d env=%+v", status, env)
	}
}

func TestAPIWebhookAdminFlow(t *testing.T) {
	app := newTestApp(t)
	received := make(chan string, 8)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("X-PES-Event")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer webhookServer.Close()

	status, env := doJSON(t, app, http.MethodPost, "/api/v1/webhooks", "demo-token", map[string]any{"name": "forbidden", "url": webhookServer.URL}, nil)
	if status != http.StatusForbidden || env.Code != response.CodeForbidden {
		t.Fatalf("demo user should not create webhook, status=%d env=%+v", status, env)
	}
	body := map[string]any{"name": "finished", "url": webhookServer.URL, "secret": "hook-secret", "events": []string{string(model.ExecutionEventFinished)}}
	status, env = doJSON(t, app, http.MethodPost, "/api/v1/webhooks", "admin-token", body, nil)
	if status != http.StatusOK || env.Code != response.CodeOK {
		t.Fatalf("create webhook status=%d env=%+v", status, env)
	}
	var created struct {
		Webhook model.WebhookEndpoint `json:"webhook"`
		Secret  string                `json:"secret"`
	}
	if err := json.Unmarshal(env.Data, &created); err != nil {
		t.Fatal(err)
	}
	if created.Webhook.ID == "" || created.Webhook.Secret != "" || created.Secret != "hook-secret" {
		t.Fatalf("bad create webhook response: %+v", created)
	}

	_, env = doJSON(t, app, http.MethodGet, "/api/plugins", "admin-token", nil, nil)
	plugins := mustData[[]model.Plugin](t, env)
	echo := enablePlugin(t, app, findPlugin(t, plugins, "echo"))
	createBody := map[string]any{"plugin_ids": []string{echo.ID}, "input": map[string]any{"text": "webhook"}}
	status, env = doJSON(t, app, http.MethodPost, "/api/executions", "demo-token", createBody, map[string]string{"Idempotency-Key": "webhook-flow"})
	if status != http.StatusCreated {
		t.Fatalf("create execution status=%d env=%+v", status, env)
	}
	exec := mustData[model.Execution](t, env)
	final := waitExecutionFinal(t, app, exec.ID)
	if final.Status != model.ExecutionStatusSuccess {
		t.Fatalf("expected success: %+v", final)
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case eventType := <-received:
			if eventType == string(model.ExecutionEventFinished) {
				status, env = doJSON(t, app, http.MethodGet, "/api/v1/webhooks/"+created.Webhook.ID+"/deliveries", "admin-token", nil, nil)
				if status != http.StatusOK {
					t.Fatalf("deliveries status=%d env=%+v", status, env)
				}
				deliveries := mustData[[]model.WebhookDelivery](t, env)
				if len(deliveries) == 0 || deliveries[0].Status != model.WebhookDeliveryDelivered {
					t.Fatalf("expected delivered webhook, got %#v", deliveries)
				}
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for webhook")
		}
	}
}
