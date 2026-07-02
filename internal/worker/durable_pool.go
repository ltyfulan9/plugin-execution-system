package worker

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"plugin-execution-system/internal/observability"
	"plugin-execution-system/internal/queue"
)

// LeasedExecutionHandler is the production worker boundary. It receives a
// durable queue lease and must perform attempt/event/result writes through the
// metadata store, not through process memory.
type LeasedExecutionHandler interface {
	HandleLeasedExecution(ctx context.Context, ref queue.TaskRef) error
}

type DurableWorkerPool struct {
	queue             queue.DurableExecutionQueue
	handler           LeasedExecutionHandler
	workerID          string
	workerCount       int
	leaseDuration     time.Duration
	heartbeatInterval time.Duration
	pollInterval      time.Duration
	reclaimInterval   time.Duration
	maxAttempts       int
	maxBackoff        time.Duration
	active            int64
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

type DurableWorkerOptions struct {
	WorkerID          string
	WorkerCount       int
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	ReclaimInterval   time.Duration
	MaxAttempts       int
	MaxBackoff        time.Duration
}

func NewDurableWorkerPool(q queue.DurableExecutionQueue, h LeasedExecutionHandler, opts DurableWorkerOptions) *DurableWorkerPool {
	if opts.WorkerID == "" {
		opts.WorkerID = "worker-unknown"
	}
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = 1
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 30 * time.Second
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = opts.LeaseDuration / 3
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 500 * time.Millisecond
	}
	if opts.ReclaimInterval <= 0 {
		opts.ReclaimInterval = opts.LeaseDuration
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = 60 * time.Second
	}
	return &DurableWorkerPool{queue: q, handler: h, workerID: opts.WorkerID, workerCount: opts.WorkerCount, leaseDuration: opts.LeaseDuration, heartbeatInterval: opts.HeartbeatInterval, pollInterval: opts.PollInterval, reclaimInterval: opts.ReclaimInterval, maxAttempts: opts.MaxAttempts, maxBackoff: opts.MaxBackoff}
}

func (p *DurableWorkerPool) Start(parent context.Context) error {
	if p.queue == nil || p.handler == nil {
		return errors.New("durable worker requires queue and handler")
	}
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel
	p.wg.Add(1)
	go p.reclaimLoop(ctx)
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.leaseLoop(ctx)
	}
	return nil
}

func (p *DurableWorkerPool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *DurableWorkerPool) ActiveCount() int64 { return atomic.LoadInt64(&p.active) }

func (p *DurableWorkerPool) leaseLoop(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refs, err := p.queue.LeaseNext(ctx, queue.LeaseOptions{WorkerID: p.workerID, MaxItems: 1, LeaseDuration: p.leaseDuration, VisibilityTimeout: p.leaseDuration})
			if err != nil || len(refs) == 0 {
				continue
			}
			observability.IncQueueLease(len(refs))
			for _, ref := range refs {
				p.handleOne(ctx, ref)
			}
		}
	}
}

func (p *DurableWorkerPool) handleOne(parent context.Context, ref queue.TaskRef) {
	atomic.AddInt64(&p.active, 1)
	defer atomic.AddInt64(&p.active, -1)
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan struct{})
	go p.heartbeatLoop(ctx, ref, done)
	err := p.handler.HandleLeasedExecution(ctx, ref)
	close(done)
	if err == nil {
		if ackErr := p.queue.Ack(parent, ref.TaskID, ref.LeaseID, p.workerID); ackErr == nil {
			observability.IncQueueAck()
		}
		return
	}
	observability.IncWorkerHandlerError()
	backoff := p.backoff(ref.AttemptNo)
	_ = p.queue.Nack(parent, ref.TaskID, ref.LeaseID, p.workerID, queue.NackOptions{Retryable: true, Backoff: backoff, Reason: err.Error(), MaxAttempts: p.maxAttempts})
	observability.IncQueueNack()
}

func (p *DurableWorkerPool) heartbeatLoop(ctx context.Context, ref queue.TaskRef, done <-chan struct{}) {
	ticker := time.NewTicker(p.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := p.queue.Heartbeat(ctx, ref.TaskID, ref.LeaseID, p.workerID, p.leaseDuration); err == nil {
				observability.IncWorkerHeartbeat()
			}
		}
	}
}

func (p *DurableWorkerPool) reclaimLoop(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.reclaimInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := p.queue.ReclaimExpiredLeases(ctx, time.Now().UTC(), 100)
			if err == nil && n > 0 {
				observability.IncQueueReclaimed(n)
			}
		}
	}
}

func (p *DurableWorkerPool) backoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	seconds := math.Pow(2, float64(attempt-1))
	d := time.Duration(seconds) * time.Second
	if d > p.maxBackoff {
		return p.maxBackoff
	}
	return d
}
