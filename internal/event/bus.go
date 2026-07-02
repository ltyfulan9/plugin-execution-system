package event

import (
	"sync"

	"plugin-execution-system/internal/model"
)

type Subscriber func(model.ExecutionEvent)

type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]Subscriber
}

func NewBus() *Bus { return &Bus{subscribers: map[string][]Subscriber{}} }

func (b *Bus) Subscribe(executionID string, fn Subscriber) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[executionID] = append(b.subscribers[executionID], fn)
	idx := len(b.subscribers[executionID]) - 1
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		items := b.subscribers[executionID]
		if idx >= 0 && idx < len(items) {
			items[idx] = nil
		}
	}
}

func (b *Bus) Publish(event model.ExecutionEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, fn := range b.subscribers[event.ExecutionID] {
		if fn != nil {
			fn(event)
		}
	}
	for _, fn := range b.subscribers["*"] {
		if fn != nil {
			fn(event)
		}
	}
}
