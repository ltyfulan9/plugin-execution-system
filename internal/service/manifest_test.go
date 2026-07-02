package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestManifestServiceSupportsV1Manifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	raw := `{
  "apiVersion":"plugin.exec/v1",
  "kind":"Plugin",
  "metadata":{"name":"hello","version":"1.2.3","description":"hello plugin","license":"Apache-2.0"},
  "runtime":{"type":"process","protocol":"stdio-json","entrypoint":"python3","args":["run.py"],"timeoutSeconds":5},
  "capabilities":["task.execute"],
  "permissions":{"network":"none"},
  "resources":{"memory":"64Mi"},
  "security":{"checksum":"sha256:demo"}
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewManifestService()
	m, _, err := svc.LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	p := svc.BuildPluginFromManifest(m, dir)
	if p.Name != "hello" || p.Version != "1.2.3" || p.Command != "python3" || p.Args[0] != "run.py" || p.TimeoutSeconds != 5 {
		t.Fatalf("unexpected plugin: %+v", p)
	}
}

func TestManifestServiceRejectsUnsupportedProtocol(t *testing.T) {
	svc := NewManifestService()
	err := svc.ValidateManifest(model.PluginManifest{
		APIVersion: "plugin.exec/v1",
		Kind:       "Plugin",
		Metadata:   model.ManifestMetadata{Name: "bad", Version: "1.0.0"},
		Runtime:    model.ManifestRuntime{Type: "process", Protocol: "grpc", Entrypoint: "python3", TimeoutSeconds: 1},
	})
	if err == nil {
		t.Fatalf("expected unsupported protocol error")
	}
}

func TestManifestServiceVerifiesSHA256Checksum(t *testing.T) {
	dir := t.TempDir()
	runPath := filepath.Join(dir, "run.py")
	content := []byte("print('ok')\n")
	if err := os.WriteFile(runPath, content, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := NewManifestService().HashManifest(content)
	raw := `{
  "apiVersion":"plugin.exec/v1",
  "kind":"Plugin",
  "metadata":{"name":"hello","version":"1.0.0"},
  "runtime":{"type":"process","protocol":"stdio-json","entrypoint":"python3","args":["run.py"],"timeoutSeconds":5},
  "security":{"checksum":"sha256:` + sum + `"}
}`
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewManifestService().LoadManifest(path); err != nil {
		t.Fatalf("expected checksum to verify: %v", err)
	}
}

func TestManifestServiceRejectsSHA256ChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "run.py"), []byte("print('ok')\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{
  "apiVersion":"plugin.exec/v1",
  "kind":"Plugin",
  "metadata":{"name":"hello","version":"1.0.0"},
  "runtime":{"type":"process","protocol":"stdio-json","entrypoint":"python3","args":["run.py"],"timeoutSeconds":5},
  "security":{"checksum":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}
}`
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewManifestService().LoadManifest(path); err == nil {
		t.Fatalf("expected checksum mismatch")
	}
}

func TestManifestServiceVerifiesEd25519Signature(t *testing.T) {
	dir := t.TempDir()
	content := []byte("print('signed')\n")
	if err := os.WriteFile(filepath.Join(dir, "run.py"), content, 0o755); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, content)
	sum := NewManifestService().HashManifest(content)
	raw := `{
  "apiVersion":"plugin.exec/v1",
  "kind":"Plugin",
  "metadata":{"name":"hello","version":"1.0.0"},
  "runtime":{"type":"process","protocol":"stdio-json","entrypoint":"python3","args":["run.py"],"timeoutSeconds":5},
  "security":{"checksum":"sha256:` + sum + `", "signature":"ed25519:` + base64.StdEncoding.EncodeToString(sig) + `"}
}`
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewManifestService("ed25519:" + base64.StdEncoding.EncodeToString(pub))
	if _, _, err := svc.LoadManifest(path); err != nil {
		t.Fatalf("expected signature to verify: %v", err)
	}
}

func TestManifestServiceRejectsEd25519SignatureWithoutTrustedKey(t *testing.T) {
	dir := t.TempDir()
	content := []byte("print('signed')\n")
	if err := os.WriteFile(filepath.Join(dir, "run.py"), content, 0o755); err != nil {
		t.Fatal(err)
	}
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, content)
	sum := NewManifestService().HashManifest(content)
	raw := `{
  "apiVersion":"plugin.exec/v1",
  "kind":"Plugin",
  "metadata":{"name":"hello","version":"1.0.0"},
  "runtime":{"type":"process","protocol":"stdio-json","entrypoint":"python3","args":["run.py"],"timeoutSeconds":5},
  "security":{"checksum":"sha256:` + sum + `", "signature":"ed25519:` + base64.StdEncoding.EncodeToString(sig) + `"}
}`
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewManifestService().LoadManifest(path); err == nil {
		t.Fatalf("expected missing trusted key to fail")
	}
}

func TestManifestServiceSupportsContainerManifest(t *testing.T) {
	svc := NewManifestService()
	m := model.PluginManifest{
		APIVersion:  "plugin.exec/v1",
		Kind:        "Plugin",
		Metadata:    model.ManifestMetadata{Name: "container-hello", Version: "1.0.0"},
		Runtime:     model.ManifestRuntime{Type: "container", Protocol: "stdio-json", Image: "ghcr.io/example/plugin:1.0.0", Args: []string{"run"}, TimeoutSeconds: 5},
		Permissions: model.ManifestPermissions{Network: "none"},
		Resources:   model.ManifestResources{Memory: "64Mi", CPU: "0.5", PIDs: 64},
	}
	if err := svc.ValidateManifest(m); err != nil {
		t.Fatalf("expected container manifest to validate: %v", err)
	}
	p := svc.BuildPluginFromManifest(m, "/plugins/container-hello")
	if p.EntryType != "container" || p.Image != "ghcr.io/example/plugin:1.0.0" || p.MemoryLimit != "64Mi" || p.PIDsLimit != 64 {
		t.Fatalf("unexpected plugin: %+v", p)
	}
}
