package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/observability"
)

type ContainerRunner struct {
	enabled bool
	docker  string
}

func NewContainerRunner(enabled bool) *ContainerRunner {
	return &ContainerRunner{enabled: enabled, docker: "docker"}
}

func (r *ContainerRunner) RunContainerPlugin(parent context.Context, p model.Plugin, stdin []byte, maxOutputBytes int) (ProcessRunResult, error) {
	if r == nil || !r.enabled {
		observability.IncSandboxDenial()
		return ProcessRunResult{ExitCode: -1}, errors.New("container runtime is disabled")
	}
	if maxOutputBytes <= 0 {
		maxOutputBytes = 65536
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 5
	}
	args, err := r.buildDockerArgs(p)
	if err != nil {
		observability.IncSandboxDenial()
		return ProcessRunResult{ExitCode: -1}, err
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(p.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.docker, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	stdout := newLimitedBuffer(maxOutputBytes)
	stderr := newLimitedBuffer(maxOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	res := ProcessRunResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}
	if ctx.Err() == context.DeadlineExceeded {
		res.Timeout = true
		return res, nil
	}
	if stdout.Exceeded() || stderr.Exceeded() {
		return res, errors.New("plugin output exceeds max size")
	}
	if err != nil {
		return res, err
	}
	return res, nil
}

func (r *ContainerRunner) buildDockerArgs(p model.Plugin) ([]string, error) {
	image := strings.TrimSpace(p.Image)
	if image == "" {
		return nil, errors.New("container plugin image is empty")
	}
	if err := validateContainerImageRef(image); err != nil {
		return nil, err
	}
	if p.WorkDir == "" {
		return nil, errors.New("plugin workdir is empty")
	}
	if err := ensureDir(p.WorkDir); err != nil {
		return nil, err
	}
	absWorkDir, err := filepath.Abs(p.WorkDir)
	if err != nil {
		return nil, err
	}
	args := []string{"run", "--rm", "-i", "--read-only", "--network", dockerNetworkMode(p.NetworkPolicy), "--workdir", "/work", "-v", absWorkDir + ":/work:ro"}
	if p.MemoryLimit != "" {
		args = append(args, "--memory", p.MemoryLimit)
	}
	if p.CPULimit != "" {
		args = append(args, "--cpus", p.CPULimit)
	}
	if p.PIDsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", p.PIDsLimit))
	}
	for k, v := range p.Env {
		if !safeContainerEnv(k, v) {
			return nil, fmt.Errorf("container env %q is not safe", k)
		}
		args = append(args, "--env", k+"="+v)
	}
	pluginArgs, err := sanitizeArgs(p.Args)
	if err != nil {
		return nil, err
	}
	args = append(args, image)
	args = append(args, pluginArgs...)
	return args, nil
}

func dockerNetworkMode(policy string) string {
	switch policy {
	case "egress", "host":
		return "bridge"
	default:
		return "none"
	}
}

func validateContainerImageRef(image string) error {
	if image == "" || strings.HasPrefix(image, "-") || strings.ContainsAny(image, " \t\r\n\x00") {
		return errors.New("container image reference is unsafe")
	}
	if strings.Contains(image, "..") {
		return errors.New("container image reference cannot contain traversal segments")
	}
	return nil
}

func safeContainerEnv(k, v string) bool {
	if k == "" || strings.ContainsAny(k, "=\x00\r\n") || strings.HasPrefix(k, "-") {
		return false
	}
	return !strings.ContainsAny(v, "\x00\r\n")
}
