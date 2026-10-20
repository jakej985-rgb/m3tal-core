package events

import (
	"sync"
	"time"
)

// ChannelEvent represents a system-wide event payload for the channel-based bus.
type ChannelEvent struct {
	Type      string `json:"type"`
	Payload   any    `json:"payload,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// ChannelEventBus implements a thread-safe publish-subscribe pattern using channels.
// This is used by the WebSocket layer to stream events to connected clients.
type ChannelEventBus struct {
	mu          sync.RWMutex
	subscribers map[chan ChannelEvent]bool
}

// NewChannelEventBus creates a new ChannelEventBus.
func NewChannelEventBus() *ChannelEventBus {
	return &ChannelEventBus{
		subscribers: make(map[chan ChannelEvent]bool),
	}
}

// Subscribe registers a new channel to receive published events.
func (eb *ChannelEventBus) Subscribe() chan ChannelEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan ChannelEvent, 100)
	eb.subscribers[ch] = true
	return ch
}

// Unsubscribe removes a channel from the subscribers list.
func (eb *ChannelEventBus) Unsubscribe(ch chan ChannelEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if _, ok := eb.subscribers[ch]; ok {
		delete(eb.subscribers, ch)
		close(ch)
	}
}

// Publish broadcasts an event to all active subscribers.
func (eb *ChannelEventBus) Publish(eventType string, payload any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	ev := ChannelEvent{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}

	for ch := range eb.subscribers {
		select {
		case ch <- ev:
		default:
			// Buffer full, drop event to prevent blocking publisher
		}
	}
}

// GlobalEventBus is the shared channel-based bus for the entire application.
// Used by api for WebSocket event streaming.
var GlobalEventBus = NewChannelEventBus()
