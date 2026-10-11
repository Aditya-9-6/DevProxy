package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestHub_LatencyPercentiles(t *testing.T) {
	hub := NewHub()
	p50, p90, p99 := hub.GetPercentiles()
	if p50 != 0 || p90 != 0 || p99 != 0 {
		t.Fatalf("expected 0 percentiles for empty hub, got %d, %d, %d", p50, p90, p99)
	}

	// Record boundary latencies
	hub.RecordLatency(-5 * time.Millisecond)
	hub.RecordLatency(10 * time.Millisecond)
	hub.RecordLatency(50 * time.Millisecond)
	hub.RecordLatency(100 * time.Millisecond)
	hub.RecordLatency(1500 * time.Millisecond) // clamped to 999

	p50, p90, p99 = hub.GetPercentiles()
	if p50 == 0 && p90 == 0 && p99 == 0 {
		t.Fatalf("expected non-zero percentiles after recording latencies")
	}
}

func TestHub_ConcurrentLatencyRecording(t *testing.T) {
	hub := NewHub()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(ms int) {
			defer wg.Done()
			hub.RecordLatency(time.Duration(ms) * time.Millisecond)
		}(i)
	}
	wg.Wait()

	p50, _, _ := hub.GetPercentiles()
	if p50 < 0 || p50 > 999 {
		t.Fatalf("unexpected p50: %d", p50)
	}
}

func TestHub_ServeWS_HandshakeAndBroadcast(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer srv.Close()

	// 1. Non-upgrade request returns 400 Bad Request
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("failed to send HTTP request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	// 2. Perform WebSocket handshake over direct TCP connection
	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}
	defer conn.Close()

	req := fmt.Sprintf("GET /ws HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", strings.TrimPrefix(srv.URL, "http://"))
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("failed to write upgrade request: %v", err)
	}

	br := bufio.NewReader(conn)
	handshakeResp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("failed to read handshake response: %v", err)
	}
	if handshakeResp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected status 101, got %d", handshakeResp.StatusCode)
	}
	if handshakeResp.Header.Get("Sec-WebSocket-Accept") != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("invalid Sec-WebSocket-Accept header: %s", handshakeResp.Header.Get("Sec-WebSocket-Accept"))
	}

	// Wait briefly for registration
	time.Sleep(30 * time.Millisecond)

	// 3. Broadcast a WebSocket message
	testMsg := &WSMessage{Opcode: OpText, Data: []byte("{\"test\":\"ws\"}"), Length: 15}
	hub.BroadcastWSMessage(testMsg)

	// Read framed message on client
	header := make([]byte, 2)
	if _, err := io.ReadFull(br, header); err != nil {
		t.Fatalf("failed to read frame header: %v", err)
	}
	if header[0] != 0x81 { // FIN + OpText
		t.Fatalf("expected 0x81 opcode, got 0x%x", header[0])
	}
	length := int(header[1] & 0x7F)
	payloadBuf := make([]byte, length)
	if _, err := io.ReadFull(br, payloadBuf); err != nil {
		t.Fatalf("failed to read payload: %v", err)
	}

	var received BroadcastPayload
	if err := json.Unmarshal(payloadBuf, &received); err != nil {
		t.Fatalf("failed to unmarshal broadcast payload: %v", err)
	}
	if received.Type != "WEBSOCKET" || received.WSMessage == nil {
		t.Fatalf("payload mismatch: %+v", received)
	}
}

func TestHub_ContextShutdown(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()

	client := &Client{
		Hub:  hub,
		Send: make(chan *BroadcastPayload, 10),
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Cancel context to initiate shutdown
	cancel()

	select {
	case <-done:
		// Succeeded gracefully
	case <-time.After(1 * time.Second):
		t.Fatalf("Hub.Run failed to exit on context cancellation")
	}

	// Verify client send channel was closed
	_, ok := <-client.Send
	if ok {
		t.Errorf("expected client Send channel to be closed upon shutdown")
	}
}

func TestHub_NonBlockingBroadcast(t *testing.T) {
	hub := NewHub()
	// Fill broadcast channel to capacity without draining
	for i := 0; i < 300; i++ {
		hub.BroadcastEvent(&ringbuffer.TrafficEvent{ID: "evt"}, []*analysis.Finding{})
		hub.BroadcastWSMessage(&WSMessage{Opcode: OpText, Data: []byte("data")})
	}
	// Verify that calls complete without hanging
}
