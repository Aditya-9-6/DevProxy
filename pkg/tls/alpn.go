package tls

import (
	"crypto/tls"
	"fmt"
	"sync"
)

// ProtocolHandler defines the interface for protocol-specific multiplexers.
type ProtocolHandler interface {
	Handle(conn *tls.Conn) error
}

// ALPNNegotiator manages protocol routing based on negotiated ALPN strings.
type ALPNNegotiator struct {
	router     *ProtocolRouter
	configPool sync.Pool
}

func NewALPNNegotiator(router *ProtocolRouter) *ALPNNegotiator {
	n := &ALPNNegotiator{
		router: router,
	}

	n.configPool = sync.Pool{
		New: func() interface{} {
			return &tls.Config{
				NextProtos: router.SupportedProtocols(),
				GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
					return nil, nil
				},
			}
		},
	}

	return n
}

func (n *ALPNNegotiator) GetConfig() *tls.Config {
	cfg := n.configPool.Get().(*tls.Config)
	cfg.NextProtos = n.router.SupportedProtocols()
	return cfg
}

// PutConfig returns a tls.Config to the pool for zero-allocation reuse.
func (n *ALPNNegotiator) PutConfig(cfg *tls.Config) {
	if cfg != nil {
		n.configPool.Put(cfg)
	}
}

// HandleConnection inspects the connection state and invokes the appropriate registered ProtocolHandler.
func (n *ALPNNegotiator) HandleConnection(conn *tls.Conn) error {
	state := conn.ConnectionState()
	proto := state.NegotiatedProtocol
	if proto == "" {
		return fmt.Errorf("no ALPN protocol negotiated")
	}

	handler, ok := n.router.Route(proto)
	if !ok {
		return fmt.Errorf("no handler registered for ALPN protocol: %s", proto)
	}

	return handler.Handle(conn)
}
