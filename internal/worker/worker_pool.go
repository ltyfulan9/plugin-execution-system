package worker

import (
	"context"
	"sync"
	"sync/atomic"
)

type WorkerPool struct {
	queue       *ExecutionQueue
	worker      *ExecutionWorker
	workerCount int
	active      int64
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewWorkerPool(queue *ExecutionQueue, worker *ExecutionWorker, workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &WorkerPool{queue: queue, worker: worker, workerCount: workerCount}
}
func (p *WorkerPool) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case id, ok := <-p.queue.Consume():
					if !ok {
						return
					}
					atomic.AddInt64(&p.active, 1)
					p.worker.HandleExecution(ctx, id)
					atomic.AddInt64(&p.active, -1)
				}
			}
		}()
	}
}
func (p *WorkerPool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}
func (p *WorkerPool) Submit(ctx context.Context, executionID string) error {
	return p.queue.Submit(ctx, executionID)
}
func (p *WorkerPool) ActiveCount() int64 { return atomic.LoadInt64(&p.active) }
