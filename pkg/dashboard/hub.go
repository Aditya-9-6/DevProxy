package dashboard

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// RFC 6455 WebSocket opcodes.
const (
	OpContinuation = 0x0
	OpText         = 0x1
	OpBinary       = 0x2
	OpClose        = 0x8
	OpPing         = 0x9
	OpPong         = 0xA
)

// BroadcastPayload represents structured data streamed to connected dashboard WebSocket clients.
type BroadcastPayload struct {
	Type      string                   `json:"type"` // "EVENT" or "WEBSOCKET"
	Event     *ringbuffer.TrafficEvent `json:"event,omitempty"`
	Findings  []*analysis.Finding      `json:"findings,omitempty"`
	WSMessage *WSMessage               `json:"ws_message,omitempty"`
}

// WSMessage represents a captured WebSocket frame payload.
type WSMessage struct {
	Opcode int    `json:"opcode"`
	Data   []byte `json:"data"`
	Length int    `json:"length"`
}

// Client represents an active WebSocket client connection.
type Client struct {
	Hub  *Hub
	Send chan *BroadcastPayload
	conn net.Conn
}

// Hub maintains active client connections, latency percentiles, and broadcasts events.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *BroadcastPayload
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	// latencyBuckets tracks request latency in 1ms increments (0-999ms)
	latencyBuckets [1000]atomic.Uint64
}

// NewHub initializes and returns a new Hub.
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *BroadcastPayload, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the main event loop for client registration, unregistration, and broadcasting.
// It accepts an optional context for graceful termination and resource cleanup.
func (h *Hub) Run(ctxOpt ...context.Context) {
	var ctx context.Context
	if len(ctxOpt) > 0 && ctxOpt[0] != nil {
		ctx = ctxOpt[0]
	} else {
		ctx = context.Background()
	}

	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.clients {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case payload := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- payload:
				default:
					// Drop if client buffer is full to prevent head-of-line blocking
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ServeWS handles WebSocket upgrade requests and registers client connections.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijack unsupported", http.StatusInternalServerError)
		return
	}

	conn, brw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	/* #nosec G401 -- SHA1 is mandated by RFC 6455 WebSocket Handshake specification */
	hasher := sha1.New()
	hasher.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	if _, err := io.WriteString(conn, resp); err != nil {
		_ = conn.Close()
		return
	}

	client := &Client{
		Hub:  h,
		Send: make(chan *BroadcastPayload, 64),
		conn: conn,
	}

	h.register <- client

	go client.writePump()
	go client.readPump(brw.Reader)
}

func (c *Client) writePump() {
	defer func() {
		_ = c.conn.Close()
	}()

	for payload := range c.Send {
		data, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		if err := writeWSFrame(c.conn, OpText, data); err != nil {
			return
		}
	}
	_ = writeWSFrame(c.conn, OpClose, nil)
}

func (c *Client) readPump(r io.Reader) {
	defer func() {
		select {
		case c.Hub.unregister <- c:
		case <-time.After(250 * time.Millisecond):
		}
		_ = c.conn.Close()
	}()

	buf := make([]byte, 1024)
	for {
		_, err := r.Read(buf)
		if err != nil {
			return
		}
	}
}

func writeWSFrame(w io.Writer, opcode int, payload []byte) error {
	length := len(payload)
	var header []byte
	if length <= 125 {
		header = []byte{byte(0x80 | opcode), byte(length)}
	} else if length <= 65535 {
		header = make([]byte, 4)
		header[0] = byte(0x80 | opcode)
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:], uint16(length))
	} else {
		header = make([]byte, 10)
		header[0] = byte(0x80 | opcode)
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:], uint64(length))
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	if length > 0 {
		_, err := w.Write(payload)
		return err
	}
	return nil
}

// RecordLatency increments the bucket corresponding to the duration.
func (h *Hub) RecordLatency(d time.Duration) {
	ms := d.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	if ms > 999 {
		ms = 999
	}
	h.latencyBuckets[ms].Add(1)
}

// GetPercentiles calculates p50, p90, and p99 from the latency buckets.
func (h *Hub) GetPercentiles() (p50, p90, p99 int64) {
	var total uint64
	for i := 0; i < 1000; i++ {
		total += h.latencyBuckets[i].Load()
	}
	if total == 0 {
		return 0, 0, 0
	}

	find := func(targetPercent float64) int64 {
		target := uint64(float64(total) * targetPercent)
		var count uint64
		for i := 0; i < 1000; i++ {
			count += h.latencyBuckets[i].Load()
			if count >= target {
				return int64(i)
			}
		}
		return 999
	}

	return find(0.50), find(0.90), find(0.99)
}

// BroadcastEvent sends a traffic event and its findings to all dashboard clients.
// It uses non-blocking send to prevent stalling the proxy hot path.
func (h *Hub) BroadcastEvent(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	select {
	case h.broadcast <- &BroadcastPayload{
		Type:     "EVENT",
		Event:    event,
		Findings: findings,
	}:
	default:
		// Queue full, drop event to protect proxy throughput
	}
}

// BroadcastWSMessage sends a captured WebSocket message to all dashboard clients.
// It uses non-blocking send to prevent stalling the proxy hot path.
func (h *Hub) BroadcastWSMessage(msg *WSMessage) {
	select {
	case h.broadcast <- &BroadcastPayload{
		Type:      "WEBSOCKET",
		WSMessage: msg,
	}:
	default:
		// Queue full, drop message to protect proxy throughput
	}
}
