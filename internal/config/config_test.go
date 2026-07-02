package config

import "testing"

func TestValidateRejectsLocalJSONInProduction(t *testing.T) {
	cfg := Default()
	cfg.Mode = ModeProduction
	cfg.Storage.Driver = "local-json"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected local-json to be rejected in production mode")
	}
}

func TestValidateRequiresPostgresDSN(t *testing.T) {
	cfg := Default()
	cfg.Mode = ModeProduction
	cfg.Storage.Driver = "postgres"
	cfg.Storage.PostgresDSN = ""
	t.Setenv("ALLOW_UNCONFIGURED_POSTGRES", "false")
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing POSTGRES_DSN to fail")
	}
}

func TestValidateAllowsLocalJSONInDevWhenExplicit(t *testing.T) {
	t.Setenv("ALLOW_LOCAL_JSON_STORE", "true")
	cfg := Default()
	cfg.Mode = ModeDev
	cfg.Storage.Driver = "local-json"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected explicit dev local-json to pass: %v", err)
	}
}

func TestValidateRejectsLocalTokenAuthInProduction(t *testing.T) {
	t.Setenv("ALLOW_LOCAL_TOKEN_AUTH", "false")
	cfg := Default()
	cfg.Mode = ModeProduction
	cfg.Storage.Driver = "postgres"
	cfg.Storage.PostgresDSN = "postgres://example"
	cfg.Auth.AuthMode = "local-token"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected local-token auth to be rejected in production")
	}
}
