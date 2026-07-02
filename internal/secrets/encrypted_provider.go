package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"plugin-execution-system/internal/model"
)

// EncryptedProvider is a production-grade baseline for metadata-backed secrets.
// It stores only AES-GCM ciphertext in memory here; production adapters can map
// the same record format to Postgres while the master key comes from KMS/Vault.
// Raw secret values never appear in manifests, API responses, or audit details.
type EncryptedProvider struct {
	mu      sync.RWMutex
	aead    cipher.AEAD
	records map[string]EncryptedRecord
}

type EncryptedRecord struct {
	TenantID   string `json:"tenant_id"`
	ProjectID  string `json:"project_id"`
	Ref        string `json:"ref"`
	NonceB64   string `json:"nonce_b64"`
	CipherB64  string `json:"cipher_b64"`
	KeyVersion string `json:"key_version"`
}

func NewEncryptedProvider(masterKey []byte) (*EncryptedProvider, error) {
	if len(masterKey) == 0 {
		return nil, errors.New("master key is required")
	}
	key := normalizeMasterKey(masterKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &EncryptedProvider{aead: aead, records: map[string]EncryptedRecord{}}, nil
}

func (p *EncryptedProvider) PutEncrypted(scope model.ResourceScope, ref string, plaintext []byte) (EncryptedRecord, error) {
	if strings.TrimSpace(ref) == "" {
		return EncryptedRecord{}, errors.New("secret ref is required")
	}
	if len(plaintext) == 0 {
		return EncryptedRecord{}, errors.New("secret value is empty")
	}
	scope = scope.Normalize()
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedRecord{}, err
	}
	aad := []byte(scope.TenantID + "/" + scope.ProjectID + "/" + ref)
	ciphertext := p.aead.Seal(nil, nonce, plaintext, aad)
	rec := EncryptedRecord{TenantID: scope.TenantID, ProjectID: scope.ProjectID, Ref: ref, NonceB64: base64.StdEncoding.EncodeToString(nonce), CipherB64: base64.StdEncoding.EncodeToString(ciphertext), KeyVersion: "local-aesgcm-v1"}
	p.mu.Lock()
	p.records[key(scope, ref)] = rec
	p.mu.Unlock()
	return rec, nil
}

func (p *EncryptedProvider) ImportRecord(rec EncryptedRecord) error {
	scope := model.ResourceScope{TenantID: rec.TenantID, ProjectID: rec.ProjectID}.Normalize()
	if strings.TrimSpace(rec.Ref) == "" || rec.NonceB64 == "" || rec.CipherB64 == "" {
		return errors.New("encrypted secret record is incomplete")
	}
	p.mu.Lock()
	p.records[key(scope, rec.Ref)] = rec
	p.mu.Unlock()
	return nil
}

func (p *EncryptedProvider) Resolve(ctx context.Context, ref Reference) ([]byte, error) {
	_ = ctx
	scope := model.ResourceScope{TenantID: ref.TenantID, ProjectID: ref.ProjectID}.Normalize()
	p.mu.RLock()
	rec, ok := p.records[key(scope, ref.Ref)]
	p.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("secret reference not found: %s", ref.Ref)
	}
	nonce, err := base64.StdEncoding.DecodeString(rec.NonceB64)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(rec.CipherB64)
	if err != nil {
		return nil, err
	}
	aad := []byte(scope.TenantID + "/" + scope.ProjectID + "/" + ref.Ref)
	plaintext, err := p.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, errors.New("secret decrypt failed")
	}
	return plaintext, nil
}

func normalizeMasterKey(masterKey []byte) []byte {
	if len(masterKey) == 32 {
		return append([]byte(nil), masterKey...)
	}
	h := sha256.Sum256(masterKey)
	return h[:]
}
