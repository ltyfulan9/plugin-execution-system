package model

import "time"

type WebhookStatus string

const (
	WebhookStatusEnabled  WebhookStatus = "enabled"
	WebhookStatusDisabled WebhookStatus = "disabled"
)

type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending   WebhookDeliveryStatus = "pending"
	WebhookDeliveryDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryFailed    WebhookDeliveryStatus = "failed"
	WebhookDeliveryDLQ       WebhookDeliveryStatus = "dlq"
)

type WebhookEndpoint struct {
	TenantID  string        `json:"tenant_id"`
	ProjectID string        `json:"project_id"`
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	URL       string        `json:"url"`
	Secret    string        `json:"secret,omitempty"`
	Events    []string      `json:"events"`
	Status    WebhookStatus `json:"status"`
	CreatedBy string        `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type WebhookDelivery struct {
	TenantID    string                `json:"tenant_id"`
	ProjectID   string                `json:"project_id"`
	ID          string                `json:"id"`
	WebhookID   string                `json:"webhook_id"`
	EventID     string                `json:"event_id"`
	EventType   string                `json:"event_type"`
	TargetURL   string                `json:"target_url"`
	Status      WebhookDeliveryStatus `json:"status"`
	AttemptNo   int                   `json:"attempt_no"`
	MaxAttempts int                   `json:"max_attempts,omitempty"`
	NextRetryAt *time.Time            `json:"next_retry_at,omitempty"`
	StatusCode  int                   `json:"status_code,omitempty"`
	Error       string                `json:"error,omitempty"`
	PayloadJSON map[string]any        `json:"payload,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	DeliveredAt *time.Time            `json:"delivered_at,omitempty"`
}

func (w WebhookEndpoint) IsEnabled() bool { return w.Status == WebhookStatusEnabled }

func (w WebhookEndpoint) Matches(eventType string) bool {
	if len(w.Events) == 0 {
		return true
	}
	for _, e := range w.Events {
		if e == "*" || e == eventType {
			return true
		}
	}
	return false
}
