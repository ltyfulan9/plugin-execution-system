package service

import (
	"testing"

	"plugin-execution-system/internal/model"
)

func TestAggregateExecutionStatus(t *testing.T) {
	s := NewResultService(nil)
	if got := s.AggregateExecutionStatus([]model.ExecutionResult{{Status: model.PluginResultSuccess}, {Status: model.PluginResultSuccess}}); got != model.ExecutionStatusSuccess {
		t.Fatalf("got %s", got)
	}
	if got := s.AggregateExecutionStatus([]model.ExecutionResult{{Status: model.PluginResultSuccess}, {Status: model.PluginResultFailed}}); got != model.ExecutionStatusPartialSuccess {
		t.Fatalf("got %s", got)
	}
	if got := s.AggregateExecutionStatus([]model.ExecutionResult{{Status: model.PluginResultTimeout}, {Status: model.PluginResultTimeout}}); got != model.ExecutionStatusTimeout {
		t.Fatalf("got %s", got)
	}
}
