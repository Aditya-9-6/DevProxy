package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/tls/rotator"
)

// ProxyServer extends existing functionality to include TLS session ticket management.
type ProxyServer struct {
	addr             string
	certManager      *certs.CertificateManager
	ringBuffer       *ringbuffer.RingBuffer
	transport        *http.Transport
	httpServer       *http.Server
	reqCounter       atomic.Uint64
	mockEngine       *mock.Engine
	insecureUpstream bool
	upstreamProxyMu  sync.RWMutex
	upstreamProxy    *url.URL
	loopWarnOnce     sync.Once
	stekRotator      *rotator.Rotator
}

// getTLSConfig returns a TLS config configured with the current STEK.
func (s *ProxyServer) getTLSConfig() *tls.Config {
	if s.stekRotator == nil {
		return &tls.Config{}
	}
	key := s.stekRotator.Get()
	return &tls.Config{
		SessionTicketKey: *key,
	}
}
