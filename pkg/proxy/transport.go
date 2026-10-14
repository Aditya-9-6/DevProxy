package proxy

import (
	"crypto/tls"
	"fmt"
	proxyTLS "github.com/Aditya-9-6/DevProxy/pkg/tls"
	"github.com/Aditya-9-6/DevProxy/pkg/waf"
	"net/http"
)

// Transport wraps the standard net/http transport with ALPN-aware routing and WAF inspection.
type Transport struct {
	Base       http.RoundTripper
	Inspector  *waf.ChunkInspector
	negotiator *proxyTLS.ALPNNegotiator
}

// NewTransport creates a new Transport with the given ALPNNegotiator.
func NewTransport(negotiator *proxyTLS.ALPNNegotiator, inspector *waf.ChunkInspector) *Transport {
	return &Transport{
		Base:       http.DefaultTransport,
		negotiator: negotiator,
		Inspector:  inspector,
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

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Base.RoundTrip(req)
}
