package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jakej985-rgb/m3tal-core/core/events"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	eb := events.NewChannelEventBus()
	ch := eb.Subscribe()
	defer eb.Unsubscribe(ch)

	eb.Publish("test.event", "hello")

	select {
	case ev := <-ch:
		if ev.Type != "test.event" {
			t.Errorf("expected event type 'test.event', got %q", ev.Type)
		}
		if ev.Payload != "hello" {
			t.Errorf("expected event payload 'hello', got %v", ev.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestGetWSEvents_Upgrade(t *testing.T) {
	srv := NewServer("test-token")

	r := chi.NewRouter()
	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.chiAuthMiddleware)
		r.Get("/ws/events", srv.GetWSEvents)
	})

	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v2/ws/events?token=test-token"

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v (response status: %v)", err, resp.Status)
	}
	defer conn.Close()

	// Publish test event
	events.GlobalEventBus.Publish("ws.test", "ws-payload")

	// Read event from socket
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var ev events.ChannelEvent
	if err := json.Unmarshal(message, &ev); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if ev.Type != "ws.test" {
		t.Errorf("expected event type 'ws.test', got %q", ev.Type)
	}
	if ev.Payload != "ws-payload" {
		t.Errorf("expected event payload 'ws-payload', got %v", ev.Payload)
	}
}
