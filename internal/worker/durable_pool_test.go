package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"plugin-execution-system/internal/queue"
)

type fakeDurableQueue struct {
	mu         sync.Mutex
	ready      []queue.TaskRef
	acked      int
	nacked     int
	heartbeats int
	reclaimed  int
}

func (q *fakeDurableQueue) Enqueue(context.Context, queue.EnqueueOptions) error { return nil }
func (q *fakeDurableQueue) LeaseNext(ctx context.Context, opts queue.LeaseOptions) ([]queue.TaskRef, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.ready) == 0 {
		return nil, nil
	}
	ref := q.ready[0]
	q.ready = q.ready[1:]
	ref.LeaseID = "lease-1"
	ref.AttemptNo++
	return []queue.TaskRef{ref}, nil
}
func (q *fakeDurableQueue) Heartbeat(ctx context.Context, taskID, leaseID, workerID string, extendBy time.Duration) error {
	q.mu.Lock()
	q.heartbeats++
	q.mu.Unlock()
	return nil
}
func (q *fakeDurableQueue) Ack(ctx context.Context, taskID, leaseID, workerID string) error {
	q.mu.Lock()
	q.acked++
	q.mu.Unlock()
	return nil
}
func (q *fakeDurableQueue) Nack(ctx context.Context, taskID, leaseID, workerID string, opts queue.NackOptions) error {
	q.mu.Lock()
	q.nacked++
	q.mu.Unlock()
	return nil
}
func (q *fakeDurableQueue) ReclaimExpiredLeases(ctx context.Context, now time.Time, limit int) (int, error) {
	q.mu.Lock()
	q.reclaimed++
	q.mu.Unlock()
	return 0, nil
}
func (q *fakeDurableQueue) MoveToDLQ(ctx context.Context, taskID, reason string) error { return nil }
func (q *fakeDurableQueue) Depth(ctx context.Context, tenantID, projectID string) (int64, error) {
	return 0, nil
}

type leasedHandler struct {
	err   error
	calls int
}

func (h *leasedHandler) HandleLeasedExecution(ctx context.Context, ref queue.TaskRef) error {
	h.calls++
	return h.err
}

func TestDurableWorkerPoolAckOnSuccess(t *testing.T) {
	q := &fakeDurableQueue{ready: []queue.TaskRef{{TaskID: "exec-1"}}}
	h := &leasedHandler{}
	p := NewDurableWorkerPool(q, h, DurableWorkerOptions{WorkerID: "w1", WorkerCount: 1, PollInterval: 5 * time.Millisecond, ReclaimInterval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		q.mu.Lock()
		acked := q.acked
		q.mu.Unlock()
		if acked == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected ack, got calls=%d", h.calls)
}

func TestDurableWorkerPoolNackOnHandlerError(t *testing.T) {
	q := &fakeDurableQueue{ready: []queue.TaskRef{{TaskID: "exec-1"}}}
	h := &leasedHandler{err: errors.New("boom")}
	p := NewDurableWorkerPool(q, h, DurableWorkerOptions{WorkerID: "w1", WorkerCount: 1, PollInterval: 5 * time.Millisecond, ReclaimInterval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		q.mu.Lock()
		nacked := q.nacked
		q.mu.Unlock()
		if nacked == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected nack")
}
