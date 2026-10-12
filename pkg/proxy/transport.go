package proxy

import (
	"crypto/tls"
	proxyTLS "github.com/Aditya-9-6/DevProxy/pkg/tls"
)

// Transport wraps the standard net/http transport with ALPN-aware routing.
type Transport struct {
	negotiator *proxyTLS.ALPNNegotiator
}

func (t *Transport) HandleConnection(conn *tls.Conn) error {
	state := conn.ConnectionState()
	proto := state.NegotiatedProtocol

	config := t.negotiator.GetConfig()
	if len(config.NextProtos) > 0 && config.NextProtos[0] == proto {
		// Logic to invoke handler from router
	}
	return nil
}
