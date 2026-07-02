package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"plugin-execution-system/internal/model"
)

func TestRuntimeExecutesProcessPlugin(t *testing.T) {
	dir := t.TempDir()
	script := `#!/usr/bin/env python3
import json, sys
payload=json.load(sys.stdin)
print(json.dumps({"success": True, "data": {"seen": payload["input"]["x"]}, "error":"", "metrics":{}}))
`
	path := filepath.Join(dir, "run.py")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := NewRuntimeService(NewProcessRunner(), 65536)
	e := model.Execution{ID: "exec1", InputJSON: map[string]any{"x": "ok"}}
	p := model.Plugin{ID: "p1", Name: "test", Version: "1", Command: "python3", Args: []string{"run.py"}, WorkDir: dir, TimeoutSeconds: 3, Status: model.PluginStatusEnabled}
	res := rt.RunOnePlugin(context.Background(), e, p, "test")
	if res.Status != model.PluginResultSuccess {
		t.Fatalf("expected success, got %+v", res)
	}
	if res.OutputJSON["seen"] != "ok" {
		t.Fatalf("unexpected output: %#v", res.OutputJSON)
	}
}

func TestRuntimeTimeout(t *testing.T) {
	dir := t.TempDir()
	script := `#!/usr/bin/env python3
import time
time.sleep(2)
`
	if err := os.WriteFile(filepath.Join(dir, "run.py"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := NewRuntimeService(NewProcessRunner(), 65536)
	e := model.Execution{ID: "exec1", InputJSON: map[string]any{"x": "ok"}}
	p := model.Plugin{ID: "p1", Name: "slow", Version: "1", Command: "python3", Args: []string{"run.py"}, WorkDir: dir, TimeoutSeconds: 1, Status: model.PluginStatusEnabled}
	start := time.Now()
	res := rt.RunOnePlugin(context.Background(), e, p, "test")
	if time.Since(start) > 1500*time.Millisecond {
		t.Fatalf("timeout took too long")
	}
	if res.Status != model.PluginResultTimeout {
		t.Fatalf("expected timeout, got %+v", res)
	}
}

func TestRuntimeRejectsUnapprovedCommand(t *testing.T) {
	dir := t.TempDir()
	rt := NewRuntimeService(NewProcessRunner("python3"), 65536)
	e := model.Execution{ID: "exec1", InputJSON: map[string]any{"x": "ok"}}
	p := model.Plugin{ID: "p1", Name: "bad", Version: "1", Command: "sh", Args: []string{"-c", "echo unsafe"}, WorkDir: dir, TimeoutSeconds: 3, Status: model.PluginStatusEnabled}
	res := rt.RunOnePlugin(context.Background(), e, p, "test")
	if res.Status != model.PluginResultRuntimeError {
		t.Fatalf("expected runtime error for unapproved command, got %+v", res)
	}
}

func TestRuntimeRejectsOversizedOutput(t *testing.T) {
	dir := t.TempDir()
	script := `#!/usr/bin/env python3
print("x" * 2048)
`
	if err := os.WriteFile(filepath.Join(dir, "run.py"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := NewRuntimeService(NewProcessRunner("python3"), 64)
	e := model.Execution{ID: "exec1", InputJSON: map[string]any{"x": "ok"}}
	p := model.Plugin{ID: "p1", Name: "large", Version: "1", Command: "python3", Args: []string{"run.py"}, WorkDir: dir, TimeoutSeconds: 3, Status: model.PluginStatusEnabled}
	res := rt.RunOnePlugin(context.Background(), e, p, "test")
	if res.Status != model.PluginResultRuntimeError {
		t.Fatalf("expected runtime error for oversized output, got %+v", res)
	}
	if len(res.StdoutPreview) > 64 {
		t.Fatalf("stdout preview should be capped, got %d bytes", len(res.StdoutPreview))
	}
}

func TestProcessRunnerRejectsNetworkPolicy(t *testing.T) {
	dir := t.TempDir()
	p := model.Plugin{Command: "python3", Args: []string{"run.py"}, WorkDir: dir, TimeoutSeconds: 1, NetworkPolicy: "egress"}
	r := NewProcessRunner("python3")
	_, err := r.RunProcessPlugin(context.Background(), p, []byte(`{}`), 1024)
	if err == nil || !strings.Contains(err.Error(), "cannot grant network") {
		t.Fatalf("expected network policy denial, got %v", err)
	}
}

func TestProcessRunnerRejectsInlineEnv(t *testing.T) {
	dir := t.TempDir()
	p := model.Plugin{Command: "python3", Args: []string{"run.py"}, WorkDir: dir, TimeoutSeconds: 1, Env: map[string]string{"SECRET": "plain"}}
	r := NewProcessRunner("python3")
	_, err := r.RunProcessPlugin(context.Background(), p, []byte(`{}`), 1024)
	if err == nil || !strings.Contains(err.Error(), "inline environment") {
		t.Fatalf("expected env policy denial, got %v", err)
	}
}
