package proxy

import (
	"crypto/tls"
	"fmt"
	proxyTLS "github.com/Aditya-9-6/DevProxy/pkg/tls"
)

// Transport wraps the standard net/http transport with ALPN-aware routing.
type Transport struct {
	negotiator *proxyTLS.ALPNNegotiator
}

// NewTransport creates a new Transport with the given ALPNNegotiator.
func NewTransport(negotiator *proxyTLS.ALPNNegotiator) *Transport {
	return &Transport{
		negotiator: negotiator,
	}
}

func (t *Transport) HandleConnection(conn *tls.Conn) error {
	state := conn.ConnectionState()
	proto := state.NegotiatedProtocol

	if proto == "" {
		return fmt.Errorf("no ALPN protocol negotiated")
	}

	return t.negotiator.HandleConnection(conn)
}
