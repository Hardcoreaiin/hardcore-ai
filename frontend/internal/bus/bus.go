// Package bus is a typed publish/subscribe channel for agent events.
//
// Publishers don't know who subscribes. Each subscriber gets its own
// buffered channel and sees every event published after it subscribed.
// Slow subscribers drop events rather than blocking the publisher.
package bus

import "sync"

type Event = any

type Bus struct {
	mu   sync.RWMutex
	subs []chan Event
	buf  int
}

func New(bufferPerSub int) *Bus {
	if bufferPerSub <= 0 {
		bufferPerSub = 64
	}
	return &Bus{buf: bufferPerSub}
}

func (b *Bus) Subscribe() <-chan Event {
	ch := make(chan Event, b.buf)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()
	return ch
}

func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
}
