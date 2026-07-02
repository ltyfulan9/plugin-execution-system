package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"plugin-execution-system/internal/model"
)

// CommandEngine is an OPA/Rego-compatible adapter boundary without importing a
// heavyweight policy runtime. Production deployments can point it at an OPA
// sidecar wrapper or an internal policy binary. The contract is intentionally
// simple: JSON input on stdin, JSON policy evaluation on stdout.
type CommandEngine struct {
	Command string
	Args    []string
	Timeout time.Duration
}

func NewCommandEngine(command string, args ...string) *CommandEngine {
	return &CommandEngine{Command: command, Args: args, Timeout: 2 * time.Second}
}

func (e *CommandEngine) Evaluate(ctx context.Context, input Input) (model.PolicyEvaluation, error) {
	if e.Command == "" {
		return model.PolicyEvaluation{}, errors.New("policy command is required")
	}
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	payload, err := json.Marshal(input)
	if err != nil {
		return model.PolicyEvaluation{}, err
	}
	cmd := exec.CommandContext(ctx, e.Command, e.Args...)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return model.PolicyEvaluation{}, ctx.Err()
		}
		return model.PolicyEvaluation{}, fmt.Errorf("policy command failed: %w: %s", err, stderr.String())
	}
	var decision model.PolicyEvaluation
	if err := json.Unmarshal(stdout.Bytes(), &decision); err != nil {
		return model.PolicyEvaluation{}, fmt.Errorf("policy command returned invalid JSON: %w", err)
	}
	if decision.Decision == "" {
		return model.PolicyEvaluation{}, errors.New("policy command omitted decision")
	}
	if decision.PolicyID == "" {
		decision.PolicyID = "command-policy"
	}
	return decision, nil
}
