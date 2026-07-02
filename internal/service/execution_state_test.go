package service

import (
	"testing"

	"plugin-execution-system/internal/model"
)

func TestExecutionStateMachine(t *testing.T) {
	sm := NewExecutionStateMachine()
	cases := [][2]model.ExecutionStatus{{model.ExecutionStatusPending, model.ExecutionStatusQueued}, {model.ExecutionStatusQueued, model.ExecutionStatusRunning}, {model.ExecutionStatusRunning, model.ExecutionStatusSuccess}}
	for _, c := range cases {
		if !sm.CanTransit(c[0], c[1]) {
			t.Fatalf("expected %s -> %s", c[0], c[1])
		}
	}
	if sm.CanTransit(model.ExecutionStatusSuccess, model.ExecutionStatusRunning) {
		t.Fatal("final state must not rerun")
	}
}
