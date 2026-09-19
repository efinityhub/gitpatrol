package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *Hub) HandleWebSocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	h.mu.Lock()
	h.clients[ws] = true
	h.mu.Unlock()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			h.mu.Lock()
			delete(h.clients, ws)
			h.mu.Unlock()
			break
		}
	}
	return nil
}

func (h *Hub) BroadcastStatus(id int, status, errMsg string) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type":    "status_update",
		"id":      id,
		"status":  status,
		"error":   errMsg,
	})

	h.mu.Lock()
	for client := range h.clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
	h.mu.Unlock()
}

func (h *Hub) BroadcastHealthStatus(health map[string]interface{}) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type":   "health_update",
		"health": health,
	})

	h.mu.Lock()
	for client := range h.clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
	h.mu.Unlock()
}

func (h *Hub) BroadcastSystemLog(logEntry map[string]interface{}) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "syslog",
		"log":  logEntry,
	})

	h.mu.Lock()
	for client := range h.clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
	h.mu.Unlock()
}
