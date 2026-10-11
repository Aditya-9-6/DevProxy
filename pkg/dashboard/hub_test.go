package dashboard

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
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
	go hub.Run()

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
	time.Sleep(20 * time.Millisecond)

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

func TestWSDefragmenter_Unfragmented(t *testing.T) {
	defrag := NewWSDefragmenter(65536)
	msg, err := defrag.ProcessFrame(true, OpText, []byte("complete message"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || string(msg.Data) != "complete message" || msg.Opcode != OpText {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func TestWSDefragmenter_Fragmented(t *testing.T) {
	defrag := NewWSDefragmenter(65536)

	// Fragment 1: OpText, FIN=false
	msg1, err := defrag.ProcessFrame(false, OpText, []byte("Hello, "))
	if err != nil || msg1 != nil {
		t.Fatalf("expected nil message and nil error for non-fin frame, got msg=%v, err=%v", msg1, err)
	}

	// Fragment 2: OpContinuation, FIN=false
	msg2, err := defrag.ProcessFrame(false, OpContinuation, []byte("fragmented "))
	if err != nil || msg2 != nil {
		t.Fatalf("expected nil message and nil error, got msg=%v, err=%v", msg2, err)
	}

	// Fragment 3: OpContinuation, FIN=true
	msg3, err := defrag.ProcessFrame(true, OpContinuation, []byte("world!"))
	if err != nil {
		t.Fatalf("unexpected error on final frame: %v", err)
	}
	if msg3 == nil || string(msg3.Data) != "Hello, fragmented world!" {
		t.Fatalf("expected full reassembled message, got %+v", msg3)
	}
	if msg3.Opcode != OpText {
		t.Fatalf("expected OpText opcode %d, got %d", OpText, msg3.Opcode)
	}
}

func TestWSDefragmenter_ControlFrames(t *testing.T) {
	defrag := NewWSDefragmenter(65536)

	// Start a fragmented message
	_, _ = defrag.ProcessFrame(false, OpText, []byte("part 1"))

	// Control frame (Ping) interleaving
	pingMsg, err := defrag.ProcessFrame(true, OpPing, []byte("heartbeat"))
	if err != nil {
		t.Fatalf("unexpected error on interleaved control frame: %v", err)
	}
	if pingMsg == nil || pingMsg.Opcode != OpPing || string(pingMsg.Data) != "heartbeat" {
		t.Fatalf("unexpected ping msg: %+v", pingMsg)
	}

	// Control frame with FIN=false must fail
	_, err = defrag.ProcessFrame(false, OpPing, []byte("invalid"))
	if !errors.Is(err, ErrControlFragmented) {
		t.Fatalf("expected ErrControlFragmented, got %v", err)
	}
}

func TestWSDefragmenter_Errors(t *testing.T) {
	defrag := NewWSDefragmenter(10)

	// Continuation before start frame
	_, err := defrag.ProcessFrame(false, OpContinuation, []byte("early"))
	if !errors.Is(err, ErrFragmentBeforeStart) {
		t.Fatalf("expected ErrFragmentBeforeStart, got %v", err)
	}

	// Nested start frame without FIN
	_, _ = defrag.ProcessFrame(false, OpText, []byte("first"))
	_, err = defrag.ProcessFrame(false, OpText, []byte("second"))
	if !errors.Is(err, ErrNestedStartFrame) {
		t.Fatalf("expected ErrNestedStartFrame, got %v", err)
	}
	defrag.Reset()

	// Message too large
	_, err = defrag.ProcessFrame(true, OpBinary, []byte("this exceeds 10 bytes!"))
	if !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("expected ErrMessageTooLarge, got %v", err)
	}

	// Invalid UTF-8 in text frame
	defragLarge := NewWSDefragmenter(1024)
	_, err = defragLarge.ProcessFrame(true, OpText, []byte{0xff, 0xfe, 0xfd})
	if !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("expected ErrInvalidUTF8, got %v", err)
	}
}
