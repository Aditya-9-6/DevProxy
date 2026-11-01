package proxy

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http2frame"
)

// GRPCTrailer represents captured HTTP/2 trailers for a gRPC stream.
type GRPCTrailer struct {
	StreamID uint32
	Payload  []byte
}

// GRPCStreamProxy manages transparent bidirectional gRPC frame routing,
// cancellation propagation via RST_STREAM, and end-to-end trailer preservation.
type GRPCStreamProxy struct {
	mu          sync.Mutex
	activeStreams map[uint32]*streamState
	closed       atomic.Bool
}

type streamState struct {
	streamID uint32
	cancel   context.CancelFunc
	trailers []byte
}

// NewGRPCStreamProxy instantiates a high-performance gRPC stream proxy.
func NewGRPCStreamProxy() *GRPCStreamProxy {
	return &GRPCStreamProxy{
		activeStreams: make(map[uint32]*streamState),
	}
}

// ProxyBidirectional pipes frames bidirectionally between client and backend,
// intercepting RST_STREAM frames for cancellation propagation and capturing trailers.
func (p *GRPCStreamProxy) ProxyBidirectional(ctx context.Context, clientConn io.ReadWriter, backendConn io.ReadWriter) error {
	if p.closed.Load() {
		return fmt.Errorf("grpc stream proxy is closed")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)

	// Client to Backend pump
	go func() {
		err := http2frame.ParseFrames(ctx, clientConn, &frameInspector{
			proxy:   p,
			out:     backendConn,
			isClient: true,
		})
		errChan <- err
	}()

	// Backend to Client pump
	go func() {
		err := http2frame.ParseFrames(ctx, backendConn, &frameInspector{
			proxy:    p,
			out:      clientConn,
			isClient: false,
		})
		errChan <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

type frameInspector struct {
	proxy    *GRPCStreamProxy
	out      io.Writer
	isClient bool
}

func (fi *frameInspector) HandleFrame(ctx context.Context, ev http2frame.FrameEvent) error {
	// Inspect frame for RST_STREAM (Type 0x3)
	if ev.Type == 0x3 && len(ev.Payload) >= 4 {
		fi.proxy.handleRstStream(ev.StreamID)
	}

	// Inspect for trailers (HEADERS frame with END_STREAM flag 0x1)
	if ev.Type == 0x1 && (ev.Flags&0x1) != 0 {
		fi.proxy.storeTrailer(ev.StreamID, ev.Payload)
	}

	// Forward frame: Header (9 bytes) + Payload
	hdr := [9]byte{
		byte(len(ev.Payload) >> 16),
		byte(len(ev.Payload) >> 8),
		byte(len(ev.Payload)),
		ev.Type,
		ev.Flags,
		byte(ev.StreamID >> 24),
		byte(ev.StreamID >> 16),
		byte(ev.StreamID >> 8),
		byte(ev.StreamID),
	}

	fi.proxy.mu.Lock()
	_, err := fi.out.Write(hdr[:])
	if err == nil && len(ev.Payload) > 0 {
		_, err = fi.out.Write(ev.Payload)
	}
	fi.proxy.mu.Unlock()

	return err
}

func (p *GRPCStreamProxy) handleRstStream(streamID uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if state, exists := p.activeStreams[streamID]; exists && state.cancel != nil {
		state.cancel()
	}
}

func (p *GRPCStreamProxy) storeTrailer(streamID uint32, payload []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	state, exists := p.activeStreams[streamID]
	if !exists {
		state = &streamState{streamID: streamID}
		p.activeStreams[streamID] = state
	}
	state.trailers = append([]byte(nil), payload...)
}

// GetTrailers retrieves captured trailers for a given gRPC stream.
func (p *GRPCStreamProxy) GetTrailers(streamID uint32) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	if state, exists := p.activeStreams[streamID]; exists {
		return state.trailers
	}
	return nil
}

// Close shuts down the proxy.
func (p *GRPCStreamProxy) Close() error {
	p.closed.Store(true)
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, state := range p.activeStreams {
		if state.cancel != nil {
			state.cancel()
		}
	}
	return nil
}
