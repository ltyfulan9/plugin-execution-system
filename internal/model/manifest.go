package model

// PluginManifest supports both the original flat demo manifest and the v1 open-source
// manifest shape described in PLUGIN_SPEC.md. Keeping both formats lets existing
// community plugins keep running while the platform moves toward a stable API.
type PluginManifest struct {
	APIVersion   string              `json:"apiVersion,omitempty"`
	Kind         string              `json:"kind,omitempty"`
	Metadata     ManifestMetadata    `json:"metadata,omitempty"`
	Runtime      ManifestRuntime     `json:"runtime,omitempty"`
	Compat       ManifestCompat      `json:"compat,omitempty"`
	Capabilities []string            `json:"capabilities,omitempty"`
	Permissions  ManifestPermissions `json:"permissions,omitempty"`
	Resources    ManifestResources   `json:"resources,omitempty"`
	Security     ManifestSecurity    `json:"security,omitempty"`

	// Legacy flat manifest fields kept for backward compatibility.
	Name           string         `json:"name,omitempty"`
	Version        string         `json:"version,omitempty"`
	Description    string         `json:"description,omitempty"`
	EntryType      string         `json:"entry_type,omitempty"`
	Command        string         `json:"command,omitempty"`
	Args           []string       `json:"args,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
	InputSchema    map[string]any `json:"input_schema,omitempty"`
	OutputSchema   map[string]any `json:"output_schema,omitempty"`
}

type ManifestMetadata struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	License     string `json:"license,omitempty"`
}

type ManifestRuntime struct {
	Type           string            `json:"type,omitempty"`
	Protocol       string            `json:"protocol,omitempty"`
	Entrypoint     string            `json:"entrypoint,omitempty"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Image          string            `json:"image,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	SecretRefs     map[string]string `json:"secretRefs,omitempty"`
	TimeoutSeconds int               `json:"timeoutSeconds,omitempty"`
}

type ManifestCompat struct {
	CoreAPI         string         `json:"coreApi,omitempty"`
	ProtocolVersion int            `json:"protocolVersion,omitempty"`
	SDK             map[string]any `json:"sdk,omitempty"`
}

type ManifestPermissions struct {
	Network string           `json:"network,omitempty"`
	FS      []ManifestFSRule `json:"fs,omitempty"`
	Process ManifestProcess  `json:"process,omitempty"`
}

type ManifestFSRule struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type ManifestProcess struct {
	AllowSubprocess bool     `json:"allowSubprocess,omitempty"`
	AllowedCommands []string `json:"allowedCommands,omitempty"`
}

type ManifestResources struct {
	CPU      string `json:"cpu,omitempty"`
	Memory   string `json:"memory,omitempty"`
	PIDs     int    `json:"pids,omitempty"`
	TempDisk string `json:"tempDisk,omitempty"`
}

type ManifestSecurity struct {
	Checksum    string `json:"checksum,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Attestation string `json:"attestation,omitempty"`
	SBOM        string `json:"sbom,omitempty"`
}

func (m PluginManifest) ManifestIdentity() string {
	return m.EffectiveName() + "@" + m.EffectiveVersion()
}
func (m PluginManifest) EffectiveName() string {
	if m.Metadata.Name != "" {
		return m.Metadata.Name
	}
	return m.Name
}
func (m PluginManifest) EffectiveVersion() string {
	if m.Metadata.Version != "" {
		return m.Metadata.Version
	}
	return m.Version
}
func (m PluginManifest) EffectiveDescription() string {
	if m.Metadata.Description != "" {
		return m.Metadata.Description
	}
	return m.Description
}
func (m PluginManifest) EffectiveEntryType() string {
	if m.Runtime.Type != "" {
		return m.Runtime.Type
	}
	if m.EntryType != "" {
		return m.EntryType
	}
	return "process"
}
func (m PluginManifest) EffectiveCommand() string {
	if m.Runtime.Entrypoint != "" {
		return m.Runtime.Entrypoint
	}
	if m.Runtime.Command != "" {
		return m.Runtime.Command
	}
	return m.Command
}
func (m PluginManifest) EffectiveArgs() []string {
	if len(m.Runtime.Args) > 0 {
		return m.Runtime.Args
	}
	return m.Args
}
func (m PluginManifest) EffectiveTimeoutSeconds() int {
	if m.Runtime.TimeoutSeconds > 0 {
		return m.Runtime.TimeoutSeconds
	}
	return m.TimeoutSeconds
}
