package worker

import (
	"context"
	"errors"
)

type ExecutionQueue struct{ ch chan string }

func NewExecutionQueue(size int) *ExecutionQueue { return &ExecutionQueue{ch: make(chan string, size)} }
func (q *ExecutionQueue) Submit(ctx context.Context, executionID string) error {
	select {
	case q.ch <- executionID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("queue full")
	}
}
func (q *ExecutionQueue) Consume() <-chan string { return q.ch }
func (q *ExecutionQueue) Len() int               { return len(q.ch) }
func (q *ExecutionQueue) Close()                 { close(q.ch) }
