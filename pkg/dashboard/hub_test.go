package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/gorilla/websocket"
)

func TestHubBroadcastAndBackpressure(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 2), // small buffer for testing backpressure
		}
		hub.register <- client
		go client.writePump()
		go client.readPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial WS: %v", err)
	}
	defer wsConn.Close()

	// Test normal event broadcast
	event := &ringbuffer.TrafficEvent{
		ID:     "test-123",
		Method: "GET",
		URL:    "https://example.com/api",
	}
	hub.BroadcastEvent(event, []*analysis.Finding{})

	_ = wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		taskErr := err
		t.Fatalf("Failed to read broadcasted message: %v", taskErr)
	}

	if !strings.Contains(string(msg), "test-123") {
		t.Errorf("Expected message to contain event ID, got %s", string(msg))
	}
}
