package service

import (
	"context"
	"strings"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestContainerRunnerBuildsHardenedDockerArgs(t *testing.T) {
	dir := t.TempDir()
	runner := NewContainerRunner(true)
	args, err := runner.buildDockerArgs(model.Plugin{
		EntryType:      "container",
		Image:          "ghcr.io/example/plugin:1.0.0",
		Args:           []string{"run"},
		WorkDir:        dir,
		NetworkPolicy:  "none",
		MemoryLimit:    "64Mi",
		CPULimit:       "0.5",
		PIDsLimit:      64,
		Env:            map[string]string{"PLUGIN_MODE": "test"},
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"run", "--rm", "-i", "--read-only", "--network none", "--workdir /work", "--memory 64Mi", "--cpus 0.5", "--pids-limit 64", "--env PLUGIN_MODE=test", "ghcr.io/example/plugin:1.0.0"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("docker args missing %q: %s", want, joined)
		}
	}
}

func TestContainerRunnerRejectsUnsafeImage(t *testing.T) {
	runner := NewContainerRunner(true)
	_, err := runner.buildDockerArgs(model.Plugin{Image: "-bad", WorkDir: t.TempDir(), TimeoutSeconds: 1})
	if err == nil {
		t.Fatalf("expected unsafe image to be rejected")
	}
}

func TestRuntimeReturnsErrorWhenContainerRuntimeDisabled(t *testing.T) {
	rt := NewRuntimeService(NewProcessRunner(), 1024).WithContainerRunner(NewContainerRunner(false))
	res := rt.RunOnePlugin(context.Background(), model.Execution{ID: "exec1", InputJSON: map[string]any{}}, model.Plugin{ID: "p1", Name: "c", Version: "1", EntryType: "container", Image: "ghcr.io/example/plugin:1.0.0", WorkDir: t.TempDir(), TimeoutSeconds: 1}, "test")
	if res.Status != model.PluginResultRuntimeError {
		t.Fatalf("expected runtime error when container runtime is disabled, got %+v", res)
	}
}
