package sse

import (
	"sync"
)

type client struct {
	id     string
	topics map[string]struct{}
	send   chan Event
}

type broker struct {
	mu      sync.RWMutex
	clients map[string]*client
}

func newBroker() *broker {
	return &broker{
		clients: make(map[string]*client),
	}
}

func (b *broker) add(c *client) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[c.id] = c
}

func (b *broker) remove(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, id)
}

func (b *broker) count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

func (b *broker) publish(topic string, event Event) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	delivered := 0
	for _, c := range b.clients {
		if topic != "" {
			if _, ok := c.topics[topic]; !ok {
				continue
			}
		}

		select {
		case c.send <- event:
			delivered++
		default:
		}
	}

	return delivered
}
