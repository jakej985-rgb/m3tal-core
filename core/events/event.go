package events

import (
	"strings"
	"sync"
	"time"
)

// Event represents a system telemetry or state change notification.
type Event struct {
	Topic     string    `json:"topic"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// Subscriber is a callback function invoked when a matching event is published.
type Subscriber func(event Event)

// subscription wraps a subscriber callback and its ID for removal.
type subscription struct {
	id      int64
	pattern string
	handler Subscriber
}

// EventBus is a thread-safe publish-subscribe event broker.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]subscription
	nextID      int64
}

var (
	globalBus *EventBus
	once      sync.Once
)

// GetBus returns the global EventBus singleton.
func GetBus() *EventBus {
	once.Do(func() {
		globalBus = &EventBus{
			subscribers: make(map[string][]subscription),
		}
	})
	return globalBus
}

// Subscribe registers a handler for events matching the specified topic pattern.
// Supports simple wildcards, e.g. "docker/*" or "*".
// Returns a unique subscription ID that can be used to Unsubscribe.
func (b *EventBus) Subscribe(pattern string, handler Subscriber) int64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID

	sub := subscription{
		id:      id,
		pattern: pattern,
		handler: handler,
	}

	b.subscribers[pattern] = append(b.subscribers[pattern], sub)
	return id
}

// Unsubscribe removes a subscription by its ID.
func (b *EventBus) Unsubscribe(id int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	for pattern, subs := range b.subscribers {
		for i, sub := range subs {
			if sub.id == id {
				// Remove the item from slice
				b.subscribers[pattern] = append(subs[:i], subs[i+1:]...)
				if len(b.subscribers[pattern]) == 0 {
					delete(b.subscribers, pattern)
				}
				return true
			}
		}
	}
	return false
}

// Publish distributes an event to all subscribers matching the event's topic.
// Distribution happens asynchronously to avoid blocking the publisher.
func (b *EventBus) Publish(topic string, payload any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	// Find all matching subscribers across all registered patterns
	for pattern, subs := range b.subscribers {
		if matchTopic(pattern, topic) {
			for _, sub := range subs {
				go sub.handler(event)
			}
		}
	}
}

// matchTopic returns true if the event topic matches the subscription pattern.
func matchTopic(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(topic, prefix)
	}
	return pattern == topic
}
