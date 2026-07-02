package service

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/response"
)

type ManifestService struct {
	trustedEd25519Keys []ed25519.PublicKey
}

func NewManifestService(trustedPublicKeys ...string) *ManifestService {
	return &ManifestService{trustedEd25519Keys: parseTrustedEd25519Keys(trustedPublicKeys)}
}
func (s *ManifestService) LoadManifest(path string) (model.PluginManifest, []byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return model.PluginManifest{}, nil, err
	}
	var m model.PluginManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return model.PluginManifest{}, nil, response.NewDetailedError(response.CodeManifestInvalid, "invalid manifest json", err.Error())
	}
	if err := s.ValidateManifest(m); err != nil {
		return model.PluginManifest{}, nil, err
	}
	if err := s.VerifyIntegrity(m, filepath.Dir(path)); err != nil {
		return model.PluginManifest{}, nil, err
	}
	return m, b, nil
}
func (s *ManifestService) ValidateManifest(m model.PluginManifest) error {
	if m.APIVersion != "" && m.APIVersion != "plugin.exec/v1" {
		return response.NewAppError(response.CodeManifestInvalid, "unsupported manifest apiVersion")
	}
	if m.Kind != "" && m.Kind != "Plugin" {
		return response.NewAppError(response.CodeManifestInvalid, "unsupported manifest kind")
	}
	if m.EffectiveName() == "" || m.EffectiveVersion() == "" {
		return response.NewAppError(response.CodeManifestInvalid, "manifest requires name and version")
	}
	switch m.EffectiveEntryType() {
	case "process":
		if m.EffectiveCommand() == "" {
			return response.NewAppError(response.CodeManifestInvalid, "process manifest requires command")
		}
		if m.Runtime.Protocol != "" && m.Runtime.Protocol != "stdio-json" {
			return response.NewAppError(response.CodeManifestInvalid, "only stdio-json protocol is supported in this runtime")
		}
	case "container":
		if strings.TrimSpace(m.Runtime.Image) == "" {
			return response.NewAppError(response.CodeManifestInvalid, "container manifest requires runtime.image")
		}
		if m.Runtime.Protocol != "" && m.Runtime.Protocol != "stdio-json" {
			return response.NewAppError(response.CodeManifestInvalid, "only stdio-json protocol is supported in this runtime")
		}
	default:
		return response.NewAppError(response.CodeManifestInvalid, "only process and container runtime types are supported")
	}
	if m.EffectiveTimeoutSeconds() <= 0 {
		return response.NewAppError(response.CodeManifestInvalid, "timeout_seconds must be positive")
	}
	if m.Permissions.Network == "" {
		return nil
	}
	switch m.Permissions.Network {
	case "none", "egress", "host":
		return nil
	default:
		return response.NewAppError(response.CodeManifestInvalid, "unsupported network permission")
	}
}
func (s *ManifestService) BuildPluginFromManifest(m model.PluginManifest, dir string) model.Plugin {
	return model.Plugin{
		TenantID:         model.DefaultTenantID,
		ProjectID:        model.DefaultProjectID,
		Name:             m.EffectiveName(),
		Version:          m.EffectiveVersion(),
		Description:      m.EffectiveDescription(),
		EntryType:        m.EffectiveEntryType(),
		Command:          m.EffectiveCommand(),
		Args:             m.EffectiveArgs(),
		Image:            strings.TrimSpace(m.Runtime.Image),
		Env:              cloneStringMap(m.Runtime.Env),
		SecretRefs:       cloneStringMap(m.Runtime.SecretRefs),
		WorkDir:          dir,
		TimeoutSeconds:   m.EffectiveTimeoutSeconds(),
		MemoryLimit:      m.Resources.Memory,
		CPULimit:         m.Resources.CPU,
		PIDsLimit:        m.Resources.PIDs,
		APIVersion:       m.APIVersion,
		Protocol:         m.Runtime.Protocol,
		Capabilities:     append([]string(nil), m.Capabilities...),
		NetworkPolicy:    m.Permissions.Network,
		Checksum:         m.Security.Checksum,
		ChecksumVerified: isReleaseChecksum(m.Security.Checksum),
	}
}

func (s *ManifestService) VerifyIntegrity(m model.PluginManifest, dir string) error {
	if err := s.VerifyChecksum(m, dir); err != nil {
		return err
	}
	return s.VerifySignature(m, dir)
}

func (s *ManifestService) VerifyChecksum(m model.PluginManifest, dir string) error {
	want := strings.TrimSpace(m.Security.Checksum)
	if want == "" {
		return nil
	}
	want = strings.TrimPrefix(want, "sha256:")
	if len(want) != 64 {
		// Placeholder checksums such as dev-local are allowed for local development manifests.
		// Release-grade manifests should use security.checksum = sha256:<64 hex chars>.
		return nil
	}
	target, err := s.checksumTarget(m, dir)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(target)
	if err != nil {
		return response.NewDetailedError(response.CodeManifestInvalid, "cannot read checksum target", err.Error())
	}
	sum := sha256.Sum256(b)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return response.NewDetailedError(response.CodeManifestInvalid, "plugin checksum mismatch", fmt.Sprintf("target=%s", filepath.Base(target)))
	}
	return nil
}

func (s *ManifestService) VerifySignature(m model.PluginManifest, dir string) error {
	sigText := strings.TrimSpace(m.Security.Signature)
	if sigText == "" || sigText == "dev-local" {
		return nil
	}
	if !strings.HasPrefix(sigText, "ed25519:") {
		return response.NewAppError(response.CodeManifestInvalid, "unsupported plugin signature format")
	}
	if len(s.trustedEd25519Keys) == 0 {
		return response.NewAppError(response.CodeManifestInvalid, "plugin signature provided but no trusted public keys configured")
	}
	target, err := s.checksumTarget(m, dir)
	if err != nil {
		return err
	}
	payload, err := os.ReadFile(target)
	if err != nil {
		return response.NewDetailedError(response.CodeManifestInvalid, "cannot read signature target", err.Error())
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sigText, "ed25519:"))
	if err != nil {
		return response.NewDetailedError(response.CodeManifestInvalid, "invalid plugin signature encoding", err.Error())
	}
	for _, key := range s.trustedEd25519Keys {
		if ed25519.Verify(key, payload, sig) {
			return nil
		}
	}
	return response.NewAppError(response.CodeManifestInvalid, "plugin signature verification failed")
}

func (s *ManifestService) checksumTarget(m model.PluginManifest, dir string) (string, error) {
	if m.EffectiveEntryType() == "container" {
		return "", response.NewAppError(response.CodeManifestInvalid, "container signatures require image attestations; file checksum is only supported for process plugins")
	}
	cmd := m.EffectiveCommand()
	if strings.ContainsAny(cmd, `/\`) && !filepath.IsAbs(cmd) {
		return safeJoinPluginPath(dir, cmd)
	}
	args := m.EffectiveArgs()
	if len(args) > 0 && args[0] != "" && !strings.HasPrefix(args[0], "-") && !filepath.IsAbs(args[0]) {
		return safeJoinPluginPath(dir, args[0])
	}
	return "", response.NewAppError(response.CodeManifestInvalid, "security.checksum requires a relative entrypoint file")
}
func (s *ManifestService) HashManifest(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func ManifestPath(pluginDir string) string { return filepath.Join(pluginDir, "manifest.json") }

func parseTrustedEd25519Keys(items []string) []ed25519.PublicKey {
	out := make([]ed25519.PublicKey, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(strings.TrimPrefix(item, "ed25519:"))
		if item == "" {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(item)
		if err != nil || len(b) != ed25519.PublicKeySize {
			continue
		}
		out = append(out, ed25519.PublicKey(b))
	}
	return out
}

func isReleaseChecksum(checksum string) bool {
	checksum = strings.TrimSpace(strings.TrimPrefix(checksum, "sha256:"))
	if len(checksum) != 64 {
		return false
	}
	_, err := hex.DecodeString(checksum)
	return err == nil
}

func safeJoinPluginPath(root, relPath string) (string, error) {
	clean := filepath.Clean(relPath)
	if strings.Contains(clean, "\x00") || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", response.NewAppError(response.CodeManifestInvalid, "entrypoint path cannot escape plugin directory")
	}
	full := filepath.Join(root, clean)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absFull)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", response.NewAppError(response.CodeManifestInvalid, "entrypoint path cannot escape plugin directory")
	}
	return full, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
