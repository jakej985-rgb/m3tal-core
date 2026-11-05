package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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
