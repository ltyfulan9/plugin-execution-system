package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/response"
	"plugin-execution-system/internal/security"
)

type WebhookService struct {
	repo                repository.WebhookStore
	httpClient          *http.Client
	maxAttempts         int
	allowPrivateTargets bool
}

type CreateWebhookInput struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events,omitempty"`
}

func NewWebhookService(repo repository.WebhookStore) *WebhookService {
	return &WebhookService{repo: repo, httpClient: &http.Client{Timeout: 5 * time.Second}, maxAttempts: 5}
}

func (s *WebhookService) WithHTTPClient(client *http.Client) *WebhookService {
	if client != nil {
		s.httpClient = client
	}
	return s
}

func (s *WebhookService) WithPrivateTargetsForTests(allow bool) *WebhookService {
	s.allowPrivateTargets = allow
	return s
}

func sameWebhookScope(user model.CurrentUser, tenantID, projectID string) bool {
	if model.IsSuperAdminRole(user.Role) {
		return true
	}
	return model.SameScope(user.Scope(), model.ResourceScope{TenantID: tenantID, ProjectID: projectID})
}

func (s *WebhookService) ListEndpointsForUser(user model.CurrentUser) ([]model.WebhookEndpoint, error) {
	items, err := s.repo.ListEndpoints()
	if err != nil {
		return nil, err
	}
	out := make([]model.WebhookEndpoint, 0, len(items))
	for _, item := range items {
		if sameWebhookScope(user, item.TenantID, item.ProjectID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *WebhookService) GetEndpointForUser(user model.CurrentUser, id string) (model.WebhookEndpoint, error) {
	endpoint, err := s.GetEndpoint(id)
	if err != nil {
		return model.WebhookEndpoint{}, err
	}
	if !sameWebhookScope(user, endpoint.TenantID, endpoint.ProjectID) {
		return model.WebhookEndpoint{}, response.NewAppError(response.CodeForbidden, "webhook scope denied")
	}
	return endpoint, nil
}

func (s *WebhookService) SetEndpointStatusForUser(user model.CurrentUser, id string, status model.WebhookStatus) (model.WebhookEndpoint, error) {
	endpoint, err := s.GetEndpointForUser(user, id)
	if err != nil {
		return model.WebhookEndpoint{}, err
	}
	if status != model.WebhookStatusEnabled && status != model.WebhookStatusDisabled {
		return model.WebhookEndpoint{}, response.NewAppError(response.CodeInvalidArgument, "invalid webhook status")
	}
	endpoint.Status = status
	endpoint.UpdatedAt = time.Now().UTC()
	return endpoint, s.repo.UpdateEndpoint(endpoint)
}

func (s *WebhookService) DeleteEndpointForUser(user model.CurrentUser, id string) error {
	if _, err := s.GetEndpointForUser(user, id); err != nil {
		return err
	}
	return s.repo.DeleteEndpoint(id)
}

func (s *WebhookService) ListDeliveriesForUser(user model.CurrentUser, webhookID string) ([]model.WebhookDelivery, error) {
	items, err := s.repo.ListDeliveries(webhookID)
	if err != nil {
		return nil, err
	}
	out := make([]model.WebhookDelivery, 0, len(items))
	for _, item := range items {
		if sameWebhookScope(user, item.TenantID, item.ProjectID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *WebhookService) CreateEndpoint(user model.CurrentUser, input CreateWebhookInput) (model.WebhookEndpoint, string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return model.WebhookEndpoint{}, "", response.NewAppError(response.CodeInvalidArgument, "webhook name is required")
	}
	if err := validateWebhookURLWithOptions(input.URL, s.allowPrivateTargets); err != nil {
		return model.WebhookEndpoint{}, "", err
	}
	secret := strings.TrimSpace(input.Secret)
	if secret == "" {
		secret = generateWebhookSecret()
	}
	now := time.Now().UTC()
	scope := user.Scope()
	endpoint := model.WebhookEndpoint{TenantID: scope.TenantID, ProjectID: scope.ProjectID, ID: newID("wh"), Name: name, URL: input.URL, Secret: secret, Events: normalizeWebhookEvents(input.Events), Status: model.WebhookStatusEnabled, CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateEndpoint(endpoint); err != nil {
		return model.WebhookEndpoint{}, "", err
	}
	return endpoint, secret, nil
}

func (s *WebhookService) ListEndpoints() ([]model.WebhookEndpoint, error) {
	return s.repo.ListEndpoints()
}

func (s *WebhookService) GetEndpoint(id string) (model.WebhookEndpoint, error) {
	endpoint, ok, err := s.repo.GetEndpointByID(id)
	if err != nil {
		return model.WebhookEndpoint{}, err
	}
	if !ok {
		return model.WebhookEndpoint{}, response.NewAppError(response.CodeNotFound, "webhook not found")
	}
	return endpoint, nil
}

func (s *WebhookService) SetEndpointStatus(id string, status model.WebhookStatus) (model.WebhookEndpoint, error) {
	if status != model.WebhookStatusEnabled && status != model.WebhookStatusDisabled {
		return model.WebhookEndpoint{}, response.NewAppError(response.CodeInvalidArgument, "invalid webhook status")
	}
	endpoint, err := s.GetEndpoint(id)
	if err != nil {
		return model.WebhookEndpoint{}, err
	}
	endpoint.Status = status
	endpoint.UpdatedAt = time.Now().UTC()
	return endpoint, s.repo.UpdateEndpoint(endpoint)
}

func (s *WebhookService) DeleteEndpoint(id string) error {
	if _, err := s.GetEndpoint(id); err != nil {
		return err
	}
	return s.repo.DeleteEndpoint(id)
}

func (s *WebhookService) ListDeliveries(webhookID string) ([]model.WebhookDelivery, error) {
	return s.repo.ListDeliveries(webhookID)
}

func (s *WebhookService) HandleExecutionEvent(evt model.ExecutionEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = s.DispatchExecutionEvent(ctx, evt)
	}()
}

func (s *WebhookService) DispatchExecutionEvent(ctx context.Context, evt model.ExecutionEvent) ([]model.WebhookDelivery, error) {
	endpoints, err := s.repo.ListEnabledEndpoints()
	if err != nil {
		return nil, err
	}
	deliveries := []model.WebhookDelivery{}
	for _, endpoint := range endpoints {
		if !model.SameScope(model.ResourceScope{TenantID: endpoint.TenantID, ProjectID: endpoint.ProjectID}, evt.Scope()) {
			continue
		}
		if !endpoint.Matches(string(evt.Type)) {
			continue
		}
		delivery := s.deliver(ctx, endpoint, evt)
		deliveries = append(deliveries, delivery)
	}
	return deliveries, nil
}

func (s *WebhookService) deliver(ctx context.Context, endpoint model.WebhookEndpoint, evt model.ExecutionEvent) model.WebhookDelivery {
	now := time.Now().UTC()
	delivery := model.WebhookDelivery{TenantID: endpoint.TenantID, ProjectID: endpoint.ProjectID, ID: newID("whd"), WebhookID: endpoint.ID, EventID: evt.ID, EventType: string(evt.Type), TargetURL: endpoint.URL, Status: model.WebhookDeliveryPending, AttemptNo: 1, MaxAttempts: s.maxAttempts, CreatedAt: now}
	payload := map[string]any{"type": evt.Type, "event": evt, "sent_at": now}
	delivery.PayloadJSON = payload
	body, err := json.Marshal(payload)
	if err != nil {
		delivery = s.markDeliveryFailed(delivery, err.Error())
		_ = s.repo.CreateDelivery(delivery)
		return delivery
	}
	timestamp := fmt.Sprintf("%d", now.Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		delivery = s.markDeliveryFailed(delivery, err.Error())
		_ = s.repo.CreateDelivery(delivery)
		return delivery
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pes-webhook/1")
	req.Header.Set("X-PES-Event", string(evt.Type))
	req.Header.Set("X-PES-Delivery", delivery.ID)
	req.Header.Set("X-PES-Timestamp", timestamp)
	req.Header.Set("X-PES-Signature", SignWebhookPayload(endpoint.Secret, timestamp, body))
	resp, err := s.httpClient.Do(req)
	if err != nil {
		delivery = s.markDeliveryFailed(delivery, err.Error())
		_ = s.repo.CreateDelivery(delivery)
		return delivery
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	delivery.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		deliveredAt := time.Now().UTC()
		delivery.Status = model.WebhookDeliveryDelivered
		delivery.DeliveredAt = &deliveredAt
	} else {
		delivery = s.markDeliveryFailed(delivery, fmt.Sprintf("unexpected status code %d", resp.StatusCode))
	}
	_ = s.repo.CreateDelivery(delivery)
	return delivery
}

func (s *WebhookService) markDeliveryFailed(delivery model.WebhookDelivery, reason string) model.WebhookDelivery {
	delivery.Error = sanitizeDeliveryError(reason)
	if delivery.MaxAttempts <= 0 {
		delivery.MaxAttempts = s.maxAttempts
	}
	if delivery.AttemptNo >= delivery.MaxAttempts {
		delivery.Status = model.WebhookDeliveryDLQ
		return delivery
	}
	next := time.Now().UTC().Add(backoffForAttempt(delivery.AttemptNo))
	delivery.Status = model.WebhookDeliveryFailed
	delivery.NextRetryAt = &next
	return delivery
}

func (s *WebhookService) RetryFailedDeliveries(ctx context.Context, now time.Time, max int) ([]model.WebhookDelivery, error) {
	items, err := s.repo.ListDeliveries("")
	if err != nil {
		return nil, err
	}
	out := []model.WebhookDelivery{}
	for _, item := range items {
		if max > 0 && len(out) >= max {
			break
		}
		if item.Status != model.WebhookDeliveryFailed || item.NextRetryAt == nil || item.NextRetryAt.After(now) {
			continue
		}
		endpoint, ok, err := s.repo.GetEndpointByID(item.WebhookID)
		if err != nil {
			return out, err
		}
		if !ok || endpoint.Status != model.WebhookStatusEnabled {
			item.Status = model.WebhookDeliveryDLQ
			item.Error = "webhook endpoint missing or disabled"
			_ = s.repo.UpdateDelivery(item)
			out = append(out, item)
			continue
		}
		retried := s.retryDelivery(ctx, endpoint, item)
		_ = s.repo.UpdateDelivery(retried)
		out = append(out, retried)
	}
	return out, nil
}

func (s *WebhookService) retryDelivery(ctx context.Context, endpoint model.WebhookEndpoint, delivery model.WebhookDelivery) model.WebhookDelivery {
	delivery.AttemptNo++
	delivery.NextRetryAt = nil
	payload := delivery.PayloadJSON
	if len(payload) == 0 {
		payload = map[string]any{"type": delivery.EventType, "event_id": delivery.EventID}
	}
	payload["delivery_id"] = delivery.ID
	payload["retry_attempt"] = delivery.AttemptNo
	body, _ := json.Marshal(payload)
	timestamp := fmt.Sprintf("%d", time.Now().UTC().Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		return s.markDeliveryFailed(delivery, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pes-webhook/1")
	req.Header.Set("X-PES-Event", delivery.EventType)
	req.Header.Set("X-PES-Delivery", delivery.ID)
	req.Header.Set("X-PES-Timestamp", timestamp)
	req.Header.Set("X-PES-Signature", SignWebhookPayload(endpoint.Secret, timestamp, body))
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return s.markDeliveryFailed(delivery, err.Error())
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	delivery.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now().UTC()
		delivery.Status = model.WebhookDeliveryDelivered
		delivery.DeliveredAt = &now
		delivery.Error = ""
		return delivery
	}
	return s.markDeliveryFailed(delivery, fmt.Sprintf("unexpected status code %d", resp.StatusCode))
}

func backoffForAttempt(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	seconds := 1 << min(attempt-1, 6)
	return time.Duration(seconds) * time.Minute
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SignWebhookPayload(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func validateWebhookURL(raw string) error {
	return validateWebhookURLWithOptions(raw, false)
}

func validateWebhookURLWithOptions(raw string, allowPrivateTargets bool) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return response.NewAppError(response.CodeInvalidArgument, "invalid webhook url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return response.NewAppError(response.CodeInvalidArgument, "webhook url must use http or https")
	}
	if strings.Contains(raw, "\x00") {
		return response.NewAppError(response.CodeInvalidArgument, "invalid webhook url")
	}
	if !allowPrivateTargets && isBlockedWebhookHost(parsed.Hostname()) {
		return response.NewAppError(response.CodeInvalidArgument, "webhook url host is not allowed")
	}
	return nil
}

func isBlockedWebhookHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func normalizeWebhookEvents(events []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" || seen[event] {
			continue
		}
		seen[event] = true
		out = append(out, event)
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func generateWebhookSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return newID("secret")
	}
	return hex.EncodeToString(b)
}

func sanitizeDeliveryError(err string) string {
	return security.SanitizePreview(err, 512)
}
