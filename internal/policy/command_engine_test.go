package policy

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"plugin-execution-system/internal/model"
)

func TestCommandEngineEvaluatesExternalPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ncat >/dev/null\necho '{\"decision\":\"allow\",\"reason\":\"script allowed\",\"policy_id\":\"test\"}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	engine := NewCommandEngine(path)
	engine.Timeout = time.Second
	got, err := engine.Evaluate(context.Background(), Input{Subject: model.PolicySubject{ActorID: "u1", Role: model.RoleUser}, Action: ActionExecutionRead})
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision != model.PolicyDecisionAllow || got.PolicyID != "test" {
		t.Fatalf("unexpected decision: %#v", got)
	}
}

func TestCommandEngineRejectsInvalidOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-policy.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho not-json\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := NewCommandEngine(path).Evaluate(context.Background(), Input{Subject: model.PolicySubject{ActorID: "u1"}, Action: ActionExecutionRead})
	if err == nil {
		t.Fatal("expected invalid output error")
	}
}
