package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all local connections
	},
}

// Hub maintains the set of active WebSocket clients and broadcasts events.
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

// WSMessage format for dashboard communication.
type WSMessage struct {
	Type     string              `json:"type"` // "REQUEST", "FINDING", "STATS"
	Event    interface{}         `json:"event,omitempty"`
	Findings []*analysis.Finding `json:"findings,omitempty"`
}

// NewHub creates a new WebSocket Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the event loop for the Hub.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					go func(c *websocket.Conn) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastEvent publishes a newly processed traffic event and any findings to all active dashboard connections.
func (h *Hub) BroadcastEvent(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	msg := WSMessage{
		Type: "REQUEST",
		Event: map[string]interface{}{
			"id":            event.ID,
			"timestamp":     event.Timestamp,
			"duration_ms":   float64(event.Duration.Nanoseconds()) / 1e6,
			"client_ip":     event.ClientIP,
			"scheme":        event.Scheme,
			"host":          event.Host,
			"method":        event.Method,
			"path":          event.Path,
			"url":           event.URL,
			"status_code":   event.StatusCode,
			"tls":           event.TLS,
			"finding_count": len(findings),
		},
		Findings: findings,
	}

	bytes, err := json.Marshal(msg)
	if err == nil {
		select {
		case h.broadcast <- bytes:
		default:
			// Buffer full, drop non-critical WS broadcast
		}
	}
}

// ServeWS handles incoming WebSocket upgrade requests from the dashboard.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.register <- conn

	// Reader pump to detect disconnects
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}
