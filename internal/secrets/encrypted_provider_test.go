package secrets

import (
	"context"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestEncryptedProviderResolveScopeBoundSecret(t *testing.T) {
	p, err := NewEncryptedProvider([]byte("test-master-key"))
	if err != nil {
		t.Fatal(err)
	}
	scope := model.ResourceScope{TenantID: "tenant-a", ProjectID: "project-a"}
	rec, err := p.PutEncrypted(scope, "api-key", []byte("secret-value"))
	if err != nil {
		t.Fatal(err)
	}
	if rec.CipherB64 == "secret-value" || rec.CipherB64 == "" {
		t.Fatalf("secret was not encrypted: %#v", rec)
	}
	got, err := p.Resolve(context.Background(), Reference{TenantID: "tenant-a", ProjectID: "project-a", Ref: "api-key"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret-value" {
		t.Fatalf("unexpected secret: %q", got)
	}
	if _, err := p.Resolve(context.Background(), Reference{TenantID: "tenant-b", ProjectID: "project-a", Ref: "api-key"}); err == nil {
		t.Fatal("expected cross-tenant resolve to fail")
	}
}

func TestEncryptedProviderRejectsTamperedRecord(t *testing.T) {
	p, err := NewEncryptedProvider([]byte("test-master-key"))
	if err != nil {
		t.Fatal(err)
	}
	scope := model.ResourceScope{TenantID: "tenant-a", ProjectID: "project-a"}
	rec, err := p.PutEncrypted(scope, "token", []byte("value"))
	if err != nil {
		t.Fatal(err)
	}
	rec.CipherB64 = rec.CipherB64[:len(rec.CipherB64)-2] + "AA"
	if err := p.ImportRecord(rec); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Resolve(context.Background(), Reference{TenantID: "tenant-a", ProjectID: "project-a", Ref: "token"}); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
}
