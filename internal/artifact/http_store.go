package artifact

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"plugin-execution-system/internal/model"
)

// HTTPObjectStore is a zero-dependency remote artifact adapter suitable for
// S3/MinIO fronted by a signing proxy or internal object-store gateway. It keeps
// the platform contract production-oriented: large outputs go to object storage,
// while metadata stores only URI/hash/size. Direct AWS SigV4 can be provided by
// a separate adapter without changing service code.
type HTTPObjectStore struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
	HMACSecret string
}

func NewHTTPObjectStore(baseURL string) (*HTTPObjectStore, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("artifact base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid artifact base URL")
	}
	return &HTTPObjectStore{BaseURL: baseURL, HTTPClient: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (s *HTTPObjectStore) Put(ctx context.Context, scope model.ResourceScope, kind ObjectKind, name string, body io.Reader) (Object, error) {
	scope = scope.Normalize()
	cleanName, err := cleanObjectName(name)
	if err != nil {
		return Object{}, err
	}
	buf := bytes.Buffer{}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(&buf, h), body)
	if err != nil {
		return Object{}, err
	}
	digest := hex.EncodeToString(h.Sum(nil))
	objectPath := path.Join(scope.TenantID, scope.ProjectID, string(kind), time.Now().UTC().Format("20060102T150405.000000000Z")+"-"+cleanName)
	uri := s.BaseURL + "/" + objectPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uri, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return Object{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-PES-Artifact-SHA256", digest)
	s.sign(req, digest)
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Object{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Object{}, fmt.Errorf("artifact put failed: status %d", resp.StatusCode)
	}
	return Object{TenantID: scope.TenantID, ProjectID: scope.ProjectID, ID: path.Base(objectPath), Kind: kind, URI: uri, SizeBytes: n, SHA256: digest, CreatedAt: time.Now().UTC()}, nil
}

func (s *HTTPObjectStore) Get(ctx context.Context, object Object) (io.ReadCloser, error) {
	if !strings.HasPrefix(object.URI, s.BaseURL+"/") {
		return nil, fmt.Errorf("artifact URI is outside configured base URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, object.URI, nil)
	if err != nil {
		return nil, err
	}
	s.sign(req, object.SHA256)
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("artifact get failed: status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (s *HTTPObjectStore) sign(req *http.Request, digest string) {
	if s.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.AuthToken)
	}
	if s.HMACSecret != "" {
		ts := fmt.Sprintf("%d", time.Now().UTC().Unix())
		mac := hmac.New(sha256.New, []byte(s.HMACSecret))
		_, _ = mac.Write([]byte(req.Method + "\n" + req.URL.Path + "\n" + ts + "\n" + digest))
		req.Header.Set("X-PES-Artifact-Timestamp", ts)
		req.Header.Set("X-PES-Artifact-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
}
