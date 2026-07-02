package runtime

type RunnerType string

const (
	RunnerProcess   RunnerType = "process"
	RunnerContainer RunnerType = "container"
	RunnerWASM      RunnerType = "wasm"
	RunnerRemote    RunnerType = "remote"
)

type NetworkPolicy string

const (
	NetworkNone   NetworkPolicy = "none"
	NetworkEgress NetworkPolicy = "egress"
	NetworkHost   NetworkPolicy = "host"
)

type Contract struct {
	RunnerType       RunnerType        `json:"runner_type"`
	Protocol         string            `json:"protocol"`
	Network          NetworkPolicy     `json:"network"`
	ReadOnlyRootFS   bool              `json:"read_only_rootfs"`
	AllowedEnv       []string          `json:"allowed_env"`
	AllowedMounts    []string          `json:"allowed_mounts"`
	MemoryLimit      string            `json:"memory_limit"`
	CPULimit         string            `json:"cpu_limit"`
	PIDsLimit        int               `json:"pids_limit"`
	MaxOutputBytes   int               `json:"max_output_bytes"`
	SanitizeOutput   bool              `json:"sanitize_output"`
	SecretReferences map[string]string `json:"secret_references,omitempty"`
}
