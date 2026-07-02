package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/observability"
	"plugin-execution-system/internal/secrets"
	"plugin-execution-system/internal/security"
)

type RuntimeService struct {
	runner          *ProcessRunner
	containerRunner *ContainerRunner
	maxOutputBytes  int
	secretProvider  secrets.Provider
}

func NewRuntimeService(runner *ProcessRunner, maxOutputBytes int) *RuntimeService {
	return &RuntimeService{runner: runner, maxOutputBytes: maxOutputBytes}
}

func (s *RuntimeService) WithContainerRunner(runner *ContainerRunner) *RuntimeService {
	s.containerRunner = runner
	return s
}

func (s *RuntimeService) WithSecretProvider(provider secrets.Provider) *RuntimeService {
	s.secretProvider = provider
	return s
}

type RunRequest struct {
	Execution model.Execution
	Plugins   []model.Plugin
	RequestID string
	Events    *ExecutionEventService
}
type PluginOutput struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data"`
	Error   string         `json:"error"`
	Metrics map[string]any `json:"metrics"`
}

type pluginInput struct {
	ExecutionID string         `json:"execution_id"`
	PluginID    string         `json:"plugin_id"`
	RequestID   string         `json:"request_id"`
	Input       map[string]any `json:"input"`
	Metadata    map[string]any `json:"metadata"`
}

func (s *RuntimeService) RunPlugins(ctx context.Context, req RunRequest) []model.ExecutionResult {
	out := make([]model.ExecutionResult, 0, len(req.Plugins))
	for _, p := range req.Plugins {
		out = append(out, s.RunOnePluginWithEvents(ctx, req.Execution, p, req.RequestID, req.Events))
	}
	return out
}
func (s *RuntimeService) RunOnePlugin(ctx context.Context, e model.Execution, p model.Plugin, requestID string) (res model.ExecutionResult) {
	return s.RunOnePluginWithEvents(ctx, e, p, requestID, nil)
}

func (s *RuntimeService) RunOnePluginWithEvents(ctx context.Context, e model.Execution, p model.Plugin, requestID string, events *ExecutionEventService) (res model.ExecutionResult) {
	start := time.Now().UTC()
	res = model.ExecutionResult{ID: newID("result"), ExecutionID: e.ID, PluginID: p.ID, PluginName: p.Name, StartedAt: start, ExitCode: -1}
	observability.IncPluginStarted(p.Name)
	if events != nil {
		events.Record(e.ID, p.ID, model.ExecutionEventPluginStarted, "", "plugin started", requestID, map[string]any{"plugin_name": p.Name, "plugin_version": p.Version})
	}
	defer func() {
		if r := recover(); r != nil {
			res.Status = model.PluginResultRuntimeError
			res.ErrorMessage = fmt.Sprintf("runtime panic: %v", r)
		}
		if res.FinishedAt.IsZero() {
			res.FinishedAt = time.Now().UTC()
		}
		res.DurationMS = res.FinishedAt.Sub(res.StartedAt).Milliseconds()
		if res.Status != "" {
			observability.IncPluginCompleted(res.Status)
			if events != nil {
				events.Record(e.ID, p.ID, model.ExecutionEventPluginFinished, string(res.Status), "plugin finished", requestID, map[string]any{"duration_ms": res.DurationMS, "error": res.ErrorMessage})
			}
		}
	}()
	payload := pluginInput{ExecutionID: e.ID, PluginID: p.ID, RequestID: requestID, Input: e.InputJSON, Metadata: map[string]any{"plugin_name": p.Name, "plugin_version": p.Version}}
	stdin, _ := json.Marshal(payload)
	runPlugin, prepErr := s.preparePluginForRuntime(ctx, p)
	if prepErr != nil {
		res.Status = model.PluginResultRuntimeError
		res.ErrorMessage = prepErr.Error()
		return res
	}
	var proc ProcessRunResult
	var err error
	if runPlugin.EntryType == "container" {
		proc, err = s.containerRunner.RunContainerPlugin(ctx, runPlugin, stdin, s.maxOutputBytes)
	} else {
		proc, err = s.runner.RunProcessPlugin(ctx, runPlugin, stdin, s.maxOutputBytes)
	}
	res.ExitCode = proc.ExitCode
	res.StdoutPreview = security.SanitizePreview(proc.Stdout, 2048)
	res.StderrPreview = security.SanitizePreview(proc.Stderr, 2048)
	if proc.Timeout {
		res.Status = model.PluginResultTimeout
		res.ErrorMessage = "plugin timeout"
		return res
	}
	if err != nil {
		res.Status = model.PluginResultRuntimeError
		res.ErrorMessage = err.Error()
		return res
	}
	var po PluginOutput
	if err := json.Unmarshal([]byte(proc.Stdout), &po); err != nil {
		res.Status = model.PluginResultInvalidOutput
		res.ErrorMessage = "plugin output is not valid json"
		return res
	}
	if !po.Success {
		res.Status = model.PluginResultFailed
		if po.Error != "" {
			res.ErrorMessage = po.Error
		} else {
			res.ErrorMessage = "plugin returned failure"
		}
		return res
	}
	res.Status = model.PluginResultSuccess
	res.OutputJSON = po.Data
	return res
}

func (s *RuntimeService) preparePluginForRuntime(ctx context.Context, p model.Plugin) (model.Plugin, error) {
	if len(p.Env) > 0 && len(p.SecretRefs) == 0 {
		return model.Plugin{}, fmt.Errorf("inline environment variables are not allowed; use runtime.secretRefs")
	}
	if len(p.SecretRefs) == 0 {
		return p, nil
	}
	if p.EntryType != "container" {
		return model.Plugin{}, fmt.Errorf("secret injection is only enabled for container runtime in the enterprise contract")
	}
	env, err := secrets.ResolveEnv(ctx, s.secretProvider, p.Scope(), p.SecretRefs)
	if err != nil {
		return model.Plugin{}, err
	}
	clone := p
	clone.Env = env
	return clone, nil
}
