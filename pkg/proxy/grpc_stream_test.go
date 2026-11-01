package proxy

import (
	"bytes"
	"context"
	"testing"
)

func TestGRPCStreamProxy_BidirectionalAndTrailer(t *testing.T) {
	proxy := NewGRPCStreamProxy()
	defer proxy.Close()

	// Construct a mock HTTP/2 frame: HEADERS frame with END_STREAM (trailer)
	payload := []byte(":status\x00\x0220\x00\x0ccontent-type\x00\x0capplication/grpc")
	length := len(payload)

	var clientBuf bytes.Buffer
	// Write frame header + payload to clientBuf
	clientBuf.Write([]byte{
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
		0x1,  // Type HEADERS
		0x1,  // Flags END_STREAM
		0x0, 0x0, 0x0, 0x1, // Stream ID 1
	})
	clientBuf.Write(payload)

	var backendBuf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := proxy.ProxyBidirectional(ctx, &clientBuf, &backendBuf)
	if err != nil && err != context.Canceled {
		t.Fatalf("unexpected proxy error: %v", err)
	}

	// Verify backend received the frame
	if backendBuf.Len() == 0 {
		t.Fatalf("expected backend to receive frames, got 0 bytes")
	}

	// Verify trailers were intercepted and stored
	trailers := proxy.GetTrailers(1)
	if len(trailers) == 0 {
		t.Fatalf("expected trailers to be captured for stream 1")
	}
}

func TestGRPCStreamProxy_RstStreamCancellation(t *testing.T) {
	proxy := NewGRPCStreamProxy()
	defer proxy.Close()

	// Construct RST_STREAM frame (Type 0x3)
	var clientBuf bytes.Buffer
	clientBuf.Write([]byte{
		0x0, 0x0, 0x4, // Length 4
		0x3,         // Type RST_STREAM
		0x0,         // Flags
		0x0, 0x0, 0x0, 0x5, // Stream ID 5
		0x0, 0x0, 0x0, 0x8, // Error code: CANCEL (0x8)
	})

	var backendBuf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = proxy.ProxyBidirectional(ctx, &clientBuf, &backendBuf)

	if backendBuf.Len() == 0 {
		t.Fatalf("expected RST_STREAM to be forwarded")
	}
}
