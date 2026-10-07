package adapters

import (
	"context"
	"sync"

	"github.com/sekai-labs/kumokura/internal/transfers/ports"
)

type InMemoryEventPublisher struct {
	mu          sync.RWMutex
	subscribers map[string][]chan ports.TransferEvent
}

func NewInMemoryEventPublisher() ports.TransferEventPublisher {
	return &InMemoryEventPublisher{
		subscribers: make(map[string][]chan ports.TransferEvent),
	}
}

func (p *InMemoryEventPublisher) Publish(event ports.TransferEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subs := p.subscribers[event.JobID]
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (p *InMemoryEventPublisher) Subscribe(ctx context.Context, jobID string) (<-chan ports.TransferEvent, func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan ports.TransferEvent, 64)
	p.subscribers[jobID] = append(p.subscribers[jobID], ch)

	cleanup := func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		subs := p.subscribers[jobID]
		for i, sub := range subs {
			if sub == ch {
				p.subscribers[jobID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, cleanup
}
