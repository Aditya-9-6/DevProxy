package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

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
}

func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	p := &ProxyServer{
		addr:        addr,
		certManager: cm,
		ringBuffer:  rb,
		mockEngine:  mock.NewEngine(),
	}
	p.transport = &http.Transport{Proxy: p.resolveForRequest}
	p.httpServer = &http.Server{Addr: addr, Handler: http.HandlerFunc(p.ServeHTTP)}
	return p
}

func (p *ProxyServer) resolveForRequest(r *http.Request) (*url.URL, error) {
	p.upstreamProxyMu.RLock()
	defer p.upstreamProxyMu.RUnlock()
	return p.upstreamProxy, nil
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *ProxyServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	go p.bumpTLSConnection(conn, r.Host)
}

func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := p.transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ProxyServer) bumpTLSConnection(c net.Conn, host string) { defer c.Close() }
func (p *ProxyServer) Start() error                              { return p.httpServer.ListenAndServe() }
func (p *ProxyServer) Close() error                              { return p.httpServer.Close() }
func (p *ProxyServer) SetInsecureUpstreamTLS(b bool)             { p.insecureUpstream = b }
func (p *ProxyServer) SetMockEngine(e *mock.Engine)              { p.mockEngine = e }
func (p *ProxyServer) GetMockEngine() *mock.Engine               { return p.mockEngine }
