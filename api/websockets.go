package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/events"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GetWSEvents upgrades to WebSocket and streams global system events.
func (s *Server) GetWSEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("⚠️ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ch := events.GlobalEventBus.Subscribe()
	defer events.GlobalEventBus.Unsubscribe(ch)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	disconnectCh := make(chan bool, 1)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				disconnectCh <- true
				return
			}
		}
	}()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-disconnectCh:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// GetWSLogs upgrades to WebSocket and streams container logs in real-time.
func (s *Server) GetWSLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Container name is required", nil)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("⚠️ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	mgr, err := docker.GetProvider()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Container provider unavailable: "+err.Error()))
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	reader, err := mgr.StreamLogs(ctx, name, "100")
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to stream logs: "+err.Error()))
		return
	}
	defer reader.Close()

	disconnectCh := make(chan bool, 1)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				disconnectCh <- true
				return
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		select {
		case <-disconnectCh:
			return
		case <-ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			if n > 0 {
				cleanMsg := cleanDockerHeader(buf[:n])
				if err := conn.WriteMessage(websocket.TextMessage, cleanMsg); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}
}

func cleanDockerHeader(data []byte) []byte {
	if len(data) < 8 {
		return data
	}
	if (data[0] == 0 || data[0] == 1 || data[0] == 2) && data[1] == 0 && data[2] == 0 && data[3] == 0 {
		return data[8:]
	}
	return data
}
