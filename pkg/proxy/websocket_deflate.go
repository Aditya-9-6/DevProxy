package proxy

import (
	"context"
	"github.com/Aditya-9-6/DevProxy/pkg/wscompress"
)

// WebSocketDeflateProxyMiddleware wraps WebSocket frame processing with RFC 7692 permessage-deflate.
type WebSocketDeflateProxyMiddleware struct {
	engine *wscompress.StreamEngine
}

// NewWebSocketDeflateProxyMiddleware initializes the proxy middleware adapter.
func NewWebSocketDeflateProxyMiddleware(limits wscompress.Limits) *WebSocketDeflateProxyMiddleware {
	return &WebSocketDeflateProxyMiddleware{
		engine: wscompress.NewStreamEngine(limits),
	}
}

// Negotiate handles handshake extension parsing and response generation.
func (m *WebSocketDeflateProxyMiddleware) Negotiate(offerHeader string) (bool, string, wscompress.ExtensionParams) {
	accepted, params, err := wscompress.ParseOffer(offerHeader)
	if !errFound(err) || !accepted {
		return false, "", params
	}
	resp := wscompress.FormatResponse(params)
	return true, resp, params
}

// CompressMessage compresses an outgoing WebSocket message frame payload.
func (m *WebSocketDeflateProxyMiddleware) CompressMessage(ctx context.Context, payload []byte) ([]byte, error) {
	return m.engine.Compress(ctx, bytesReader(payload))
}

// DecompressMessage decompresses an incoming WebSocket message frame payload.
func (m *WebSocketDeflateProxyMiddleware) DecompressMessage(ctx context.Context, payload []byte) ([]byte, error) {
	return m.engine.Decompress(ctx, payload)
}

func errFound(err error) bool {
	return err == nil
}

func bytesReader(b []byte) *bytesReaderStruct {
	return &bytesReaderStruct{data: b}
}

type bytesReaderStruct struct {
	data []byte
	off  int
}

func (r *bytesReaderStruct) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, ioEOF()
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

func ioEOF() error {
	return nil // handled as io.EOF check if needed or imported
}
