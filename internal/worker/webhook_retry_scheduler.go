package worker

import (
	"context"
	"sync"
	"time"

	"plugin-execution-system/internal/logging"
	"plugin-execution-system/internal/observability"
	"plugin-execution-system/internal/service"
)

type WebhookRetryOptions struct {
	Interval time.Duration
	Batch    int
}

type WebhookRetryScheduler struct {
	service *service.WebhookService
	opts    WebhookRetryOptions
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
}

func NewWebhookRetryScheduler(svc *service.WebhookService, opts WebhookRetryOptions) *WebhookRetryScheduler {
	if opts.Interval <= 0 {
		opts.Interval = 30 * time.Second
	}
	if opts.Batch <= 0 {
		opts.Batch = 50
	}
	return &WebhookRetryScheduler{service: svc, opts: opts}
}

func (s *WebhookRetryScheduler) Start(ctx context.Context) {
	if s == nil || s.service == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.loop(runCtx)
}

func (s *WebhookRetryScheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}

func (s *WebhookRetryScheduler) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.retryOnce(ctx, now)
		}
	}
}

func (s *WebhookRetryScheduler) retryOnce(ctx context.Context, now time.Time) {
	items, err := s.service.RetryFailedDeliveries(ctx, now.UTC(), s.opts.Batch)
	if err != nil {
		observability.IncWebhookRetryError()
		logging.Warn("webhook_retry_failed", logging.Fields{"error": err.Error()})
		return
	}
	if len(items) > 0 {
		observability.IncWebhookRetry(len(items))
	}
}
