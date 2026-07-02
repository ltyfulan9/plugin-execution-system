package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/observability"
)

const defaultMaxPluginTimeoutSeconds = 30

type ProcessRunner struct {
	allowedCommands       map[string]struct{}
	maxTimeoutSeconds     int
	allowRelativeBinaries bool
}

func NewProcessRunner(allowedCommands ...string) *ProcessRunner {
	if len(allowedCommands) == 0 {
		allowedCommands = []string{"python3", "python", "node"}
	}
	allowed := make(map[string]struct{}, len(allowedCommands))
	for _, cmd := range allowedCommands {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			allowed[cmd] = struct{}{}
		}
	}
	return &ProcessRunner{allowedCommands: allowed, maxTimeoutSeconds: defaultMaxPluginTimeoutSeconds, allowRelativeBinaries: true}
}

type ProcessRunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Timeout  bool
}

func (r *ProcessRunner) RunProcessPlugin(parent context.Context, p model.Plugin, stdin []byte, maxOutputBytes int) (ProcessRunResult, error) {
	if err := validateProcessPolicy(p); err != nil {
		observability.IncSandboxDenial()
		return ProcessRunResult{ExitCode: -1}, err
	}
	if maxOutputBytes <= 0 {
		maxOutputBytes = 65536
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 5
	}
	if r.maxTimeoutSeconds <= 0 {
		r.maxTimeoutSeconds = defaultMaxPluginTimeoutSeconds
	}
	if p.TimeoutSeconds > r.maxTimeoutSeconds {
		p.TimeoutSeconds = r.maxTimeoutSeconds
	}
	command, args, err := r.buildSafeCommand(p)
	if err != nil {
		observability.IncSandboxDenial()
		return ProcessRunResult{ExitCode: -1}, err
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(p.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = p.WorkDir
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

func validateProcessPolicy(p model.Plugin) error {
	if strings.TrimSpace(p.NetworkPolicy) != "" && strings.TrimSpace(p.NetworkPolicy) != "none" {
		return errors.New("process runner cannot grant network access; use container or remote runner with explicit policy")
	}
	if len(p.Env) > 0 {
		return errors.New("process runner refuses inline environment variables; use secret references and runtime injection")
	}
	return nil
}

func (r *ProcessRunner) buildSafeCommand(p model.Plugin) (string, []string, error) {
	command := strings.TrimSpace(p.Command)
	if command == "" {
		return "", nil, errors.New("plugin command is empty")
	}
	if p.WorkDir == "" {
		return "", nil, errors.New("plugin workdir is empty")
	}
	if err := ensureDir(p.WorkDir); err != nil {
		return "", nil, err
	}
	if strings.Contains(command, "\x00") {
		return "", nil, errors.New("plugin command contains invalid byte")
	}
	if strings.ContainsAny(command, `/\\`) {
		if !r.allowRelativeBinaries {
			return "", nil, errors.New("plugin command path is not allowed")
		}
		if filepath.IsAbs(command) {
			return "", nil, errors.New("absolute plugin command path is not allowed")
		}
		clean := filepath.Clean(command)
		if strings.HasPrefix(clean, "..") {
			return "", nil, errors.New("plugin command cannot escape plugin directory")
		}
		full := filepath.Join(p.WorkDir, clean)
		if !pathInside(p.WorkDir, full) {
			return "", nil, errors.New("plugin command cannot escape plugin directory")
		}
		args, err := sanitizeArgs(p.Args)
		if err != nil {
			return "", nil, err
		}
		return full, args, nil
	}
	if _, ok := r.allowedCommands[command]; !ok {
		return "", nil, fmt.Errorf("plugin command %q is not allowed", command)
	}
	args, err := sanitizeArgs(p.Args)
	if err != nil {
		return "", nil, err
	}
	resolved, err := resolveBareCommand(command)
	if err != nil {
		return "", nil, err
	}
	return resolved, args, nil
}

func resolveBareCommand(command string) (string, error) {
	if runtime.GOOS != "windows" {
		return command, nil
	}
	candidates := windowsCommandCandidates(command)
	var fallback string
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(candidate))
		lower := strings.ToLower(candidate)
		if info.Size() == 0 && strings.Contains(lower, `\windowsapps\`) {
			continue
		}
		if (command == "python" || command == "python3") && (ext == ".bat" || ext == ".cmd") {
			if fallback == "" {
				fallback = candidate
			}
			continue
		}
		return candidate, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return command, nil
}

func windowsCommandCandidates(command string) []string {
	dirs := filepath.SplitList(os.Getenv("PATH"))
	exts := []string{""}
	if filepath.Ext(command) == "" {
		exts = strings.Split(os.Getenv("PATHEXT"), ";")
		if len(exts) == 0 || exts[0] == "" {
			exts = []string{".COM", ".EXE", ".BAT", ".CMD"}
		}
	}
	out := make([]string, 0, len(dirs)*len(exts))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		for _, ext := range exts {
			out = append(out, filepath.Join(dir, command+ext))
		}
	}
	return out
}

func sanitizeArgs(args []string) ([]string, error) {
	clean := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.Contains(arg, "\x00") {
			return nil, errors.New("plugin argument contains invalid byte")
		}
		clean = append(clean, arg)
	}
	return clean, nil
}

func ensureDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("plugin workdir %s is not a directory", dir)
	}
	return nil
}

func pathInside(root, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

type limitedBuffer struct {
	buf      bytes.Buffer
	limit    int
	exceeded bool
}

func newLimitedBuffer(limit int) *limitedBuffer {
	if limit <= 0 {
		limit = 65536
	}
	return &limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.exceeded = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.exceeded = true
		return len(p), nil
	}
	_, err := b.buf.Write(p)
	if errors.Is(err, io.ErrShortWrite) {
		b.exceeded = true
		return len(p), nil
	}
	return len(p), err
}

func (b *limitedBuffer) String() string { return b.buf.String() }
func (b *limitedBuffer) Exceeded() bool { return b.exceeded }
