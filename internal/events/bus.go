package events

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Buffer       int
	DropWhenFull bool
}

type Bus struct {
	mu     sync.RWMutex
	config Config
	nextID uint64
	subs   map[uint64]*Subscription
	closed bool
}

type Subscription struct {
	id     uint64
	topic  string
	events chan Event
	bus    *Bus
	once   sync.Once
}

func NewBus(config Config) *Bus {
	if config.Buffer < 1 {
		config.Buffer = 32
	}
	return &Bus{config: config, subs: make(map[uint64]*Subscription)}
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, subscription := range b.subs {
		if subscription.topic != "" && subscription.topic != event.Type {
			continue
		}
		if b.config.DropWhenFull {
			select {
			case subscription.events <- event.Clone():
			default:
			}
		} else {
			subscription.events <- event
		}
	}
}

func (b *Bus) NewEvent(eventType, subject, actor string, payload map[string]any, now time.Time) Event {
	id := atomic.AddUint64(&b.nextID, 1)
	return Event{
		ID: fmt.Sprintf("evt-%d", id), Type: eventType, Subject: subject,
		Actor: actor, Payload: payload, CreatedAt: now,
	}
}

func (b *Bus) Subscribe(topic string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return &Subscription{events: closedEvents()}
	}
	id := atomic.AddUint64(&b.nextID, 1)
	subscription := &Subscription{
		id: id, topic: topic, events: make(chan Event, b.config.Buffer), bus: b,
	}
	b.subs[id] = subscription
	return subscription
}

func (s *Subscription) Events() <-chan Event { return s.events }

func (s *Subscription) Close() {
	s.once.Do(func() {
		s.bus.mu.Lock()
		delete(s.bus.subs, s.id)
		close(s.events)
		s.bus.mu.Unlock()
	})
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, subscription := range b.subs {
		close(subscription.events)
		delete(b.subs, id)
	}
}

func closedEvents() chan Event {
	channel := make(chan Event)
	close(channel)
	return channel
}
