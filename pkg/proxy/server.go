package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/tls/rotator"
)

// ProxyServer is the high-throughput asynchronous proxy engine.
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

// NewProxyServer creates a new ProxyServer.
func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false, // Ensure clean HTTP/1.1 wire protocol for proxying
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	p := &ProxyServer{
		addr:        addr,
		certManager: cm,
		ringBuffer:  rb,
		transport:   transport,
		mockEngine:  mock.NewEngine(),
		stekRotator: rotator.NewRotator(),
	}
	// Single resolver for every egress path: the -upstream-proxy flag wins,
	// otherwise HTTPS_PROXY/HTTP_PROXY/NO_PROXY/ALL_PROXY apply.
	transport.Proxy = p.resolveForRequest

	p.httpServer = &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(p.ServeHTTP),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return p
}

// SetInsecureUpstreamTLS enables or disables skipping certificate verification on upstream endpoints.
func (p *ProxyServer) SetInsecureUpstreamTLS(insecure bool) {
	p.insecureUpstream = insecure
	if p.transport.TLSClientConfig == nil {
		p.transport.TLSClientConfig = &tls.Config{}
	}
	p.transport.TLSClientConfig.InsecureSkipVerify = insecure
}

// SetMockEngine configures the mock & chaos engine.
func (p *ProxyServer) SetMockEngine(eng *mock.Engine) {
	p.mockEngine = eng
}

// GetMockEngine returns the active mock & chaos engine.
func (p *ProxyServer) GetMockEngine() *mock.Engine {
	return p.mockEngine
}

// SetSTEKRotator configures the TLS session ticket rotator.
func (p *ProxyServer) SetSTEKRotator(r *rotator.Rotator) {
	p.stekRotator = r
}

// GetSTEKRotator returns the active TLS session ticket rotator.
func (p *ProxyServer) GetSTEKRotator() *rotator.Rotator {
	return p.stekRotator
}

// getTLSConfig returns a TLS config configured with the current STEK.
func (p *ProxyServer) getTLSConfig() *tls.Config {
	if p.stekRotator == nil {
		return &tls.Config{}
	}
	key := p.stekRotator.Get()
	if key == nil {
		return &tls.Config{}
	}
	return &tls.Config{
		SessionTicketKey: *key,
	}
}

// Start runs the proxy server listener.
func (p *ProxyServer) Start() error {
	return p.httpServer.ListenAndServe()
}

// Close terminates the proxy server.
func (p *ProxyServer) Close() error {
	return p.httpServer.Close()
}

// ServeHTTP delegates between HTTPS CONNECT tunneling (TLS bumping) and plain HTTP proxy requests.
func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}
