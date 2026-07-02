package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Mode      Mode
	Server    ServerConfig
	Storage   StorageConfig
	Plugin    PluginConfig
	Execution ExecutionConfig
	Auth      AuthConfig
	Security  SecurityConfig
}

type ServerConfig struct{ Addr string }

type Mode string

const (
	ModeProduction Mode = "production"
	ModeDev        Mode = "dev"
	ModeTest       Mode = "test"
)

type StorageConfig struct {
	Driver          string // postgres in production; local-json is accepted only for local/dev tests
	Dir             string
	PostgresDSN     string
	PostgresDriver  string
	RequireHA       bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
	AutoMigrate     bool
	MigrationPath   string
}

type PluginConfig struct{ Dir string }
type ExecutionConfig struct {
	WorkerCount             int
	QueueSize               int // local/dev adapter only; production queues use durable leasing
	MaxOutputBytes          int
	AllowedCommands         []string
	ContainerRuntimeEnabled bool
	LeaseDurationSeconds    int
	HeartbeatSeconds        int
	MaxAttempts             int
}
type AuthConfig struct {
	DemoToken       string
	AdminToken      string
	OIDCIssuerURL   string
	OIDCClientID    string
	SAMLMetadataURL string
	AuthMode        string // local-token, oidc, saml
}
type SecurityConfig struct{ TrustedPluginPublicKeys []string }

func Default() Config {
	return Config{
		Mode:   Mode(getEnv("APP_MODE", "production")),
		Server: ServerConfig{Addr: getEnv("SERVER_ADDR", ":8080")},
		Storage: StorageConfig{
			Driver:          getEnv("METADATA_STORE", "postgres"),
			Dir:             getEnv("STORAGE_DIR", "data"),
			PostgresDSN:     getEnv("POSTGRES_DSN", ""),
			PostgresDriver:  getEnv("POSTGRES_DRIVER", "postgres"),
			RequireHA:       getEnvBool("REQUIRE_HA_METADATA", true),
			MaxOpenConns:    getEnvInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("POSTGRES_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvInt("POSTGRES_CONN_MAX_LIFETIME_SECONDS", 300),
			AutoMigrate:     getEnvBool("POSTGRES_AUTO_MIGRATE", false),
			MigrationPath:   getEnv("POSTGRES_MIGRATION_PATH", "migrations/postgres/001_enterprise_metadata.sql"),
		},
		Plugin: PluginConfig{Dir: getEnv("PLUGIN_DIR", "plugins")},
		Execution: ExecutionConfig{
			WorkerCount: getEnvInt("WORKER_COUNT", 2), QueueSize: getEnvInt("QUEUE_SIZE", 64),
			MaxOutputBytes:          getEnvInt("MAX_OUTPUT_BYTES", 65536),
			AllowedCommands:         getEnvCSV("PLUGIN_ALLOWED_COMMANDS", []string{"python3", "python", "node"}),
			ContainerRuntimeEnabled: getEnvBool("PLUGIN_CONTAINER_RUNTIME_ENABLED", true),
			LeaseDurationSeconds:    getEnvInt("WORKER_LEASE_SECONDS", 30),
			HeartbeatSeconds:        getEnvInt("WORKER_HEARTBEAT_SECONDS", 10),
			MaxAttempts:             getEnvInt("TASK_MAX_ATTEMPTS", 3),
		},
		Auth:     AuthConfig{DemoToken: getEnv("DEMO_TOKEN", "demo-token"), AdminToken: getEnv("ADMIN_TOKEN", "admin-token"), AuthMode: getEnv("AUTH_MODE", "local-token"), OIDCIssuerURL: getEnv("OIDC_ISSUER_URL", ""), OIDCClientID: getEnv("OIDC_CLIENT_ID", ""), SAMLMetadataURL: getEnv("SAML_METADATA_URL", "")},
		Security: SecurityConfig{TrustedPluginPublicKeys: getEnvCSV("PLUGIN_TRUSTED_PUBLIC_KEYS", nil)},
	}
}

func Load() (Config, error) { cfg := Default(); return cfg, cfg.Validate() }
func (c Config) Validate() error {
	if c.Mode != ModeProduction && c.Mode != ModeDev && c.Mode != ModeTest {
		return errors.New("APP_MODE must be production, dev, or test")
	}
	switch c.Storage.Driver {
	case "postgres":
		if c.Storage.PostgresDSN == "" && !getEnvBool("ALLOW_UNCONFIGURED_POSTGRES", false) {
			return errors.New("POSTGRES_DSN is required when METADATA_STORE=postgres; set METADATA_STORE=local-json only for local/dev")
		}
	case "local-json":
		if c.Mode == ModeProduction {
			return errors.New("local-json store is forbidden in APP_MODE=production")
		}
		if !getEnvBool("ALLOW_LOCAL_JSON_STORE", false) {
			return errors.New("local-json store is only allowed when ALLOW_LOCAL_JSON_STORE=true")
		}
	default:
		return errors.New("unsupported METADATA_STORE: " + c.Storage.Driver)
	}
	if c.Mode == ModeProduction && c.Auth.AuthMode == "local-token" && !getEnvBool("ALLOW_LOCAL_TOKEN_AUTH", false) {
		return errors.New("local-token auth is forbidden in APP_MODE=production unless ALLOW_LOCAL_TOKEN_AUTH=true for bootstrap only")
	}
	if c.Auth.AuthMode == "oidc" && (c.Auth.OIDCIssuerURL == "" || c.Auth.OIDCClientID == "") {
		return errors.New("OIDC_ISSUER_URL and OIDC_CLIENT_ID are required when AUTH_MODE=oidc")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func getEnvBool(key string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	return def
}
func getEnvCSV(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}
