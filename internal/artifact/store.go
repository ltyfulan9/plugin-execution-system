package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"plugin-execution-system/internal/model"
)

type ObjectKind string

const (
	ObjectResult ObjectKind = "result"
	ObjectLog    ObjectKind = "log"
	ObjectBlob   ObjectKind = "blob"
)

type Object struct {
	TenantID  string     `json:"tenant_id"`
	ProjectID string     `json:"project_id"`
	ID        string     `json:"id"`
	Kind      ObjectKind `json:"kind"`
	URI       string     `json:"uri"`
	SizeBytes int64      `json:"size_bytes"`
	SHA256    string     `json:"sha256"`
	CreatedAt time.Time  `json:"created_at"`
}

type Store interface {
	Put(ctx context.Context, scope model.ResourceScope, kind ObjectKind, name string, body io.Reader) (Object, error)
	Get(ctx context.Context, object Object) (io.ReadCloser, error)
}

// LocalStore is a dev/test artifact adapter. Production should use S3/GCS/Azure
// Blob/MinIO or an equivalent object store, while the database stores only the
// URI, hash, and metadata.
type LocalStore struct{ root string }

func NewLocalStore(root string) (*LocalStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("artifact root is required")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &LocalStore{root: root}, nil
}

func (s *LocalStore) Put(ctx context.Context, scope model.ResourceScope, kind ObjectKind, name string, body io.Reader) (Object, error) {
	_ = ctx
	scope = scope.Normalize()
	cleanName, err := cleanObjectName(name)
	if err != nil {
		return Object{}, err
	}
	dir := filepath.Join(s.root, scope.TenantID, scope.ProjectID, string(kind))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return Object{}, err
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + cleanName
	path := filepath.Join(dir, id)
	if !inside(s.root, path) {
		return Object{}, errors.New("artifact path escapes root")
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return Object{}, err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return Object{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return Object{}, closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return Object{}, err
	}
	return Object{TenantID: scope.TenantID, ProjectID: scope.ProjectID, ID: id, Kind: kind, URI: localFileURI(path), SizeBytes: n, SHA256: hex.EncodeToString(h.Sum(nil)), CreatedAt: time.Now().UTC()}, nil
}

func (s *LocalStore) Get(ctx context.Context, object Object) (io.ReadCloser, error) {
	_ = ctx
	path, err := localFilePath(object.URI)
	if err != nil {
		return nil, fmt.Errorf("unsupported local artifact uri: %s", object.URI)
	}
	if !inside(s.root, path) {
		return nil, errors.New("artifact path escapes root")
	}
	return os.Open(path)
}

func localFileURI(path string) string {
	clean := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return (&url.URL{Scheme: "file", Path: clean}).String()
}

func localFilePath(rawURI string) (string, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "file" || parsed.Path == "" {
		return "", errors.New("invalid local artifact uri")
	}
	path := parsed.Path
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/")
	}
	return filepath.FromSlash(path), nil
}

func cleanObjectName(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "\x00/\\") {
		return "", errors.New("invalid artifact name")
	}
	return name, nil
}

func inside(root, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	return err == nil && (rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)))
}
