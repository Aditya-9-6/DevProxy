package tls

import (
	"crypto/tls"
	"sync"
)

// ProtocolHandler defines the interface for protocol-specific multiplexers.
type ProtocolHandler interface {
	Handle(conn *tls.Conn) error
}

// ALPNNegotiator manages protocol routing based on negotiated ALPN strings.
type ALPNNegotiator struct {
	router *ProtocolRouter
	pool   sync.Pool
}

func NewALPNNegotiator(router *ProtocolRouter) *ALPNNegotiator {
	return &ALPNNegotiator{
		router: router,
	}
}

func (n *ALPNNegotiator) GetConfig() *tls.Config {
	return &tls.Config{
		NextProtos: n.router.SupportedProtocols(),
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			return nil, nil
		},
	}
}
