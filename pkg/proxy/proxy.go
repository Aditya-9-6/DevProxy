package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"golang.org/x/net/proxy"
)

var bufferPool = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

type ProxyServer struct {
	addr             string
	certManager      *certs.CertificateManager
	ringBuffer       *ringbuffer.RingBuffer
	transport        *http.Transport
	httpServer       *http.Server
	reqCounter       atomic.Uint64
	mockEngine       *mock.Engine
	insecureUpstream bool
	upstreamProxy    *url.URL
	mu               sync.RWMutex
}

func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:   false,
		MaxIdleConns:        500,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	p := &ProxyServer{addr: addr, certManager: cm, ringBuffer: rb, transport: transport, mockEngine: mock.NewEngine()}
	transport.Proxy = p.resolveForRequest
	p.httpServer = &http.Server{Addr: addr, Handler: http.HandlerFunc(p.ServeHTTP)}
	return p
}

func (p *ProxyServer) SetUpstreamProxy(proxyURL string) error {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.upstreamProxy = u
	p.mu.Unlock()
	return nil
}

func (p *ProxyServer) resolveForRequest(req *http.Request) (*url.URL, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.upstreamProxy != nil {
		if strings.Contains(req.Host, p.addr) {
			return nil, nil
		}
		return p.upstreamProxy, nil
	}
	return http.ProxyFromEnvironment(req)
}

func (p *ProxyServer) dialTunnel(network, addr string) (net.Conn, error) {
	p.mu.RLock()
	up := p.upstreamProxy
	p.mu.RUnlock()
	if up != nil && strings.HasPrefix(up.Scheme, "socks") {
		dialer, err := proxy.FromURL(up, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return dialer.Dial(network, addr)
	}
	return net.DialTimeout(network, addr, 10*time.Second)
}

func (p *ProxyServer) SetInsecureUpstreamTLS(insecure bool) {
	p.insecureUpstream = insecure
	if insecure {
		p.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

func (p *ProxyServer) SetMockEngine(eng *mock.Engine) { p.mockEngine = eng }
func (p *ProxyServer) Start() error                   { return p.httpServer.ListenAndServe() }
func (p *ProxyServer) Close() error                   { return p.httpServer.Close() }

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
		http.Error(w, "Hijacking not supported", 500)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	go p.bumpTLSConnection(clientConn, r.Host)
}

func (p *ProxyServer) bumpTLSConnection(clientConn net.Conn, targetHostPort string) {
	defer clientConn.Close()
	host, _, _ := net.SplitHostPort(targetHostPort)
	tlsCert, _ := p.certManager.GetOrCreateCertificate(host)
	tlsClientConn := tls.Server(clientConn, &tls.Config{Certificates: []tls.Certificate{*tlsCert}})
	if err := tlsClientConn.Handshake(); err != nil {
		return
	}
	defer tlsClientConn.Close()
	rawConn, _ := p.dialTunnel("tcp", targetHostPort)
	upstreamConn := tls.Client(rawConn, &tls.Config{ServerName: host, InsecureSkipVerify: p.insecureUpstream})
	defer upstreamConn.Close()
	clientReader, upstreamReader := bufio.NewReader(tlsClientConn), bufio.NewReader(upstreamConn)
	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			break
		}
		req.URL.Scheme = "https"
		req.URL.Host = host
		req.Write(upstreamConn)
		resp, _ := http.ReadResponse(upstreamReader, req)
		resp.Write(tlsClientConn)
	}
}

func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if isWebSocketUpgrade(r) {
		p.handleWebSocketUpgrade(w, r)
		return
	}
	removeHopByHopHeaders(r.Header)
	r.RequestURI = ""
	if !r.URL.IsAbs() {
		r.URL.Scheme = "http"
		r.URL.Host = r.Host
	}
	resp, err := p.transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	io.Copy(w, resp.Body)
}

func (p *ProxyServer) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	hijacker, _ := w.(http.Hijacker)
	clientConn, _, _ := hijacker.Hijack()
	defer clientConn.Close()
	upstreamConn, _ := p.dialTunnel("tcp", r.Host)
	defer upstreamConn.Close()
	r.Write(upstreamConn)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(clientConn, upstreamConn) }()
	go func() { defer wg.Done(); io.Copy(upstreamConn, clientConn) }()
	wg.Wait()
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}
func removeHopByHopHeaders(h http.Header) {
	for _, k := range []string{"Connection", "Upgrade", "Proxy-Connection", "Keep-Alive"} {
		h.Del(k)
	}
}
