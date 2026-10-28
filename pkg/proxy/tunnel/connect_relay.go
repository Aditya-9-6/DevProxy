package tunnel

import (
	"context"
	"net"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/proxy/relay"
)

// TunnelHandler handles transparent TCP tunneling between inbound and upstream sessions.
type TunnelHandler struct {
	IdleTimeout time.Duration
}

// NewTunnelHandler creates a TunnelHandler with specified idle timeout.
func NewTunnelHandler(timeout time.Duration) *TunnelHandler {
	return &TunnelHandler{
		IdleTimeout: timeout,
	}
}

// RelaySession transparently pipes traffic between client and destination target.
func (h *TunnelHandler) RelaySession(ctx context.Context, clientConn, upstreamConn net.Conn) error {
	cfg := relay.Config{
		IdleTimeout: h.IdleTimeout,
	}
	return relay.FullDuplexRelay(ctx, clientConn, upstreamConn, cfg)
}
