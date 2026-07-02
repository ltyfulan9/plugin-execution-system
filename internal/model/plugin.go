package model

import (
	"fmt"
	"time"
)

type Plugin struct {
	TenantID         string            `json:"tenant_id"`
	ProjectID        string            `json:"project_id"`
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description"`
	EntryType        string            `json:"entry_type"`
	Command          string            `json:"command"`
	Args             []string          `json:"args"`
	Image            string            `json:"image,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	SecretRefs       map[string]string `json:"secret_refs,omitempty"`
	WorkDir          string            `json:"work_dir"`
	TimeoutSeconds   int               `json:"timeout_seconds"`
	MemoryLimit      string            `json:"memory_limit,omitempty"`
	CPULimit         string            `json:"cpu_limit,omitempty"`
	PIDsLimit        int               `json:"pids_limit,omitempty"`
	Status           PluginStatus      `json:"status"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	APIVersion       string            `json:"api_version,omitempty"`
	Protocol         string            `json:"protocol,omitempty"`
	Capabilities     []string          `json:"capabilities,omitempty"`
	NetworkPolicy    string            `json:"network_policy,omitempty"`
	Checksum         string            `json:"checksum,omitempty"`
	ChecksumVerified bool              `json:"checksum_verified"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type PluginCapability struct {
	InputSchema  map[string]any `json:"input_schema,omitempty"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
}

func (p Plugin) PluginIdentity() string { return fmt.Sprintf("%s@%s", p.Name, p.Version) }
func (p Plugin) DisplayName() string    { return p.PluginIdentity() }

func (p Plugin) Scope() ResourceScope {
	return ResourceScope{TenantID: p.TenantID, ProjectID: p.ProjectID}.Normalize()
}
