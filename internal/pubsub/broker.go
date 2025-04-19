package pubsub

import (
	"context"
	"sync"
)

const bufferSize = 1024

// Broker allows clients to publish events and subscribe to events
type Broker[T any] struct {
	subs map[chan Event[T]]struct{} // subscriptions
	mu   sync.Mutex                 // sync access to map
	done chan struct{}              // close when broker is shutting down
}

// NewBroker constructs a pub/sub broker.
func NewBroker[T any]() *Broker[T] {
	b := &Broker[T]{
		subs: make(map[chan Event[T]]struct{}),
		done: make(chan struct{}),
	}
	return b
}

// Shutdown the broker, terminating any subscriptions.
func (b *Broker[T]) Shutdown() {
	close(b.done)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove each subscriber entry, so Publish() cannot send any further
	// messages, and close each subscriber's channel, so the subscriber cannot
	// consume any more messages.
	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}
}

// Subscribe subscribes the caller to a stream of events. The returned channel
// is closed when the broker is shutdown.
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if broker has shutdown and if so return closed channel
	select {
	case <-b.done:
		ch := make(chan Event[T])
		close(ch)
		return ch
	default:
	}

	// Subscribe
	sub := make(chan Event[T], bufferSize)
	b.subs[sub] = struct{}{}

	// Unsubscribe when context is done.
	go func() {
		<-ctx.Done()

		b.mu.Lock()
		defer b.mu.Unlock()

		// Check if broker has shutdown and if so do nothing
		select {
		case <-b.done:
			return
		default:
		}

		delete(b.subs, sub)
		close(sub)
	}()

	return sub
}

// Publish an event to subscribers.
func (b *Broker[T]) Publish(t EventType, payload T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for sub := range b.subs {
		select {
		case sub <- Event[T]{Type: t, Payload: payload}:
		case <-b.done:
			return
		}
	}
}
