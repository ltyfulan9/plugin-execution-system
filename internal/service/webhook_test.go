package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/storage"
)

func newWebhookServiceForTest(t *testing.T) *WebhookService {
	t.Helper()
	store, err := storage.OpenJSONStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(store); err != nil {
		t.Fatal(err)
	}
	return NewWebhookService(repository.NewWebhookRepository(store)).WithPrivateTargetsForTests(true)
}

func TestWebhookDeliverySignsPayloadAndPersistsResult(t *testing.T) {
	var gotSignature, gotTimestamp, gotEvent string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature = r.Header.Get("X-PES-Signature")
		gotTimestamp = r.Header.Get("X-PES-Timestamp")
		gotEvent = r.Header.Get("X-PES-Event")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	svc := newWebhookServiceForTest(t)
	user := model.CurrentUser{ID: "user_admin", Username: "admin", Role: model.RoleAdmin}
	endpoint, secret, err := svc.CreateEndpoint(user, CreateWebhookInput{Name: "ci", URL: server.URL, Secret: "top-secret", Events: []string{string(model.ExecutionEventFinished)}})
	if err != nil {
		t.Fatal(err)
	}
	if secret != "top-secret" || endpoint.Secret != "top-secret" {
		t.Fatalf("unexpected secret returned")
	}
	evt := model.ExecutionEvent{ID: "evt_1", ExecutionID: "exec_1", Type: model.ExecutionEventFinished, Status: "Success", CreatedAt: time.Now().UTC()}
	deliveries, err := svc.DispatchExecutionEvent(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || deliveries[0].Status != model.WebhookDeliveryDelivered || deliveries[0].StatusCode != http.StatusAccepted {
		t.Fatalf("bad delivery: %#v", deliveries)
	}
	if gotEvent != string(model.ExecutionEventFinished) {
		t.Fatalf("wrong event header: %s", gotEvent)
	}
	wantSig := SignWebhookPayload("top-secret", gotTimestamp, gotBody)
	if gotSignature != wantSig || !strings.HasPrefix(gotSignature, "sha256=") {
		t.Fatalf("bad signature got=%s want=%s timestamp=%s body=%s", gotSignature, wantSig, gotTimestamp, string(gotBody))
	}
	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil || payload["type"] != string(model.ExecutionEventFinished) {
		t.Fatalf("bad payload: %#v err=%v", payload, err)
	}
	stored, err := svc.ListDeliveries(endpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Status != model.WebhookDeliveryDelivered {
		t.Fatalf("delivery not persisted: %#v", stored)
	}
}

func TestWebhookEventFilterAndFailedDelivery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newWebhookServiceForTest(t)
	user := model.CurrentUser{ID: "user_admin", Username: "admin", Role: model.RoleAdmin}
	_, _, err := svc.CreateEndpoint(user, CreateWebhookInput{Name: "only-created", URL: server.URL, Secret: "secret", Events: []string{string(model.ExecutionEventCreated)}})
	if err != nil {
		t.Fatal(err)
	}
	skipped, err := svc.DispatchExecutionEvent(context.Background(), model.ExecutionEvent{ID: "evt_skip", ExecutionID: "exec", Type: model.ExecutionEventFinished})
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 {
		t.Fatalf("expected filtered event to skip delivery, got %#v", skipped)
	}
	failed, err := svc.DispatchExecutionEvent(context.Background(), model.ExecutionEvent{ID: "evt_fail", ExecutionID: "exec", Type: model.ExecutionEventCreated})
	if err != nil {
		t.Fatal(err)
	}
	if len(failed) != 1 || failed[0].Status != model.WebhookDeliveryFailed || failed[0].StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected failed delivery, got %#v", failed)
	}
}

func TestWebhookRejectsInvalidURL(t *testing.T) {
	svc := newWebhookServiceForTest(t)
	_, _, err := svc.CreateEndpoint(model.CurrentUser{ID: "admin", Role: model.RoleAdmin}, CreateWebhookInput{Name: "bad", URL: "file:///tmp/pwn", Secret: "x"})
	if err == nil {
		t.Fatalf("expected invalid url error")
	}
}

func TestWebhookRejectsPrivateAndLocalTargets(t *testing.T) {
	for _, raw := range []string{"http://localhost:8080/hook", "http://127.0.0.1/hook", "http://10.0.0.1/hook", "http://192.168.1.2/hook", "http://169.254.1.1/hook"} {
		if err := validateWebhookURL(raw); err == nil {
			t.Fatalf("expected %s to be rejected", raw)
		}
	}
}

func TestWebhookFailedDeliveryPlansRetryThenDLQ(t *testing.T) {
	svc := NewWebhookService(nil)
	d := model.WebhookDelivery{AttemptNo: 1, MaxAttempts: 2}
	d = svc.markDeliveryFailed(d, "boom")
	if d.Status != model.WebhookDeliveryFailed || d.NextRetryAt == nil {
		t.Fatalf("expected retryable failure, got %+v", d)
	}
	d.AttemptNo = 2
	d = svc.markDeliveryFailed(d, "boom")
	if d.Status != model.WebhookDeliveryDLQ {
		t.Fatalf("expected dlq, got %+v", d)
	}
}

func TestWebhookScopeIsolationForListGetAndDispatch(t *testing.T) {
	svc := newWebhookServiceForTest(t)
	userA := model.CurrentUser{TenantID: "tenant_a", ProjectID: "project_a", ID: "admin_a", Role: model.RoleAdmin}
	userB := model.CurrentUser{TenantID: "tenant_b", ProjectID: "project_b", ID: "admin_b", Role: model.RoleAdmin}
	super := model.CurrentUser{TenantID: "root", ProjectID: "root", ID: "root", Role: model.RoleSuperAdmin}

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	defer serverB.Close()

	epA, _, err := svc.CreateEndpoint(userA, CreateWebhookInput{Name: "a", URL: serverA.URL, Secret: "a", Events: []string{"*"}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.CreateEndpoint(userB, CreateWebhookInput{Name: "b", URL: serverB.URL, Secret: "b", Events: []string{"*"}})
	if err != nil {
		t.Fatal(err)
	}

	itemsA, err := svc.ListEndpointsForUser(userA)
	if err != nil {
		t.Fatal(err)
	}
	if len(itemsA) != 1 || itemsA[0].TenantID != "tenant_a" {
		t.Fatalf("tenant A should see only its webhook, got %#v", itemsA)
	}
	itemsSuper, err := svc.ListEndpointsForUser(super)
	if err != nil {
		t.Fatal(err)
	}
	if len(itemsSuper) != 2 {
		t.Fatalf("super admin should see both, got %#v", itemsSuper)
	}
	if _, err := svc.GetEndpointForUser(userB, epA.ID); err == nil {
		t.Fatalf("tenant B must not read tenant A webhook")
	}

	deliveries, err := svc.DispatchExecutionEvent(context.Background(), model.ExecutionEvent{TenantID: "tenant_a", ProjectID: "project_a", ID: "evt_a", ExecutionID: "exec_a", Type: model.ExecutionEventFinished})
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || deliveries[0].TenantID != "tenant_a" {
		t.Fatalf("event should dispatch only to same-scope endpoint, got %#v", deliveries)
	}
}

func TestWebhookRetryReplaysOriginalPayload(t *testing.T) {
	attempts := 0
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		bodies = append(bodies, payload)
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	svc := newWebhookServiceForTest(t)
	user := model.CurrentUser{TenantID: "tenant-a", ProjectID: "project-a", ID: "admin", Role: model.RoleAdmin}
	ep, _, err := svc.CreateEndpoint(user, CreateWebhookInput{Name: "retry", URL: server.URL, Secret: "secret", Events: []string{string(model.ExecutionEventFinished)}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.DispatchExecutionEvent(context.Background(), model.ExecutionEvent{TenantID: "tenant-a", ProjectID: "project-a", ID: "evt-retry", ExecutionID: "exec-retry", Type: model.ExecutionEventFinished, Status: "failed"})
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.ListDeliveries(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != model.WebhookDeliveryFailed || len(items[0].PayloadJSON) == 0 {
		t.Fatalf("expected failed delivery with payload, got %#v", items)
	}
	now := time.Now().UTC().Add(time.Hour)
	retried, err := svc.RetryFailedDeliveries(context.Background(), now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(retried) != 1 || retried[0].Status != model.WebhookDeliveryDelivered {
		t.Fatalf("expected delivered retry, got %#v", retried)
	}
	if len(bodies) != 2 || bodies[0]["type"] != bodies[1]["type"] || bodies[1]["delivery_id"] == "" || bodies[1]["retry_attempt"] == nil {
		t.Fatalf("retry did not replay original payload with retry metadata: %#v", bodies)
	}
}
