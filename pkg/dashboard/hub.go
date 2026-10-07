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
		return true
	},
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

type WSMessage struct {
	Type     string              `json:"type"`
	Event    interface{}         `json:"event,omitempty"`
	Findings []*analysis.Finding `json:"findings,omitempty"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

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
				_ = client.WriteMessage(websocket.TextMessage, message)
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastEvent(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	msg := WSMessage{
		Type: "REQUEST",
		Event: map[string]interface{}{
			"id":            event.ID,
			"timestamp":     event.Timestamp,
			"method":        event.Method,
			"url":           event.URL,
			"content_type":  event.ReqHeaders.Get("Content-Type"),
			"finding_count": len(findings),
		},
		Findings: findings,
	}

	bytes, err := json.Marshal(msg)
	if err == nil {
		select {
		case h.broadcast <- bytes:
		default:
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.register <- conn
	go func() {
		defer func() { h.unregister <- conn }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}
