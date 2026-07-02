package pesclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

type Envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   json.RawMessage `json:"error"`
}

type Plugin struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type Execution struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Webhook struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Status string   `json:"status"`
}

type CreateWebhookRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events,omitempty"`
}

type CreateWebhookResponse struct {
	Webhook Webhook `json:"webhook"`
	Secret  string  `json:"secret"`
}

type CreateExecutionRequest struct {
	PluginIDs []string       `json:"plugin_ids"`
	Input     map[string]any `json:"input"`
}

func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	return out, c.do(ctx, http.MethodGet, "/api/v1/health", "", nil, &out)
}

func (c *Client) ListPlugins(ctx context.Context) ([]Plugin, error) {
	var out []Plugin
	return out, c.do(ctx, http.MethodGet, "/api/v1/plugins", "", nil, &out)
}

func (c *Client) CreateExecution(ctx context.Context, req CreateExecutionRequest, idempotencyKey string) (Execution, error) {
	var out Execution
	return out, c.do(ctx, http.MethodPost, "/api/v1/executions", idempotencyKey, req, &out)
}

func (c *Client) GetExecution(ctx context.Context, id string) (Execution, error) {
	var out Execution
	return out, c.do(ctx, http.MethodGet, "/api/v1/executions/"+id, "", nil, &out)
}

func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var out []Webhook
	return out, c.do(ctx, http.MethodGet, "/api/v1/webhooks", "", nil, &out)
}

func (c *Client) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (CreateWebhookResponse, error) {
	var out CreateWebhookResponse
	return out, c.do(ctx, http.MethodPost, "/api/v1/webhooks", "", req, &out)
}

func (c *Client) do(ctx context.Context, method, path, idem string, body any, out any) error {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pes api error: status=%d code=%s message=%s error=%s", resp.StatusCode, env.Code, env.Message, string(env.Error))
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}
