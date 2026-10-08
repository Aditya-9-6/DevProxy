package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
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
	"github.com/google/uuid"
	"golang.org/x/net/proxy"
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
	upstreamProxy    *url.URL
	mu               sync.RWMutex
}

// NewProxyServer creates a new ProxyServer.
func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
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
	}
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

// SetUpstreamProxy configures an upstream proxy for egress traffic.
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

// SetInsecureUpstreamTLS enables or disables skipping certificate verification on upstream endpoints.
func (p *ProxyServer) SetInsecureUpstreamTLS(insecure bool) {
	p.insecureUpstream = insecure
	if insecure {
		p.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

// SetMockEngine configures the mock & chaos engine.
func (p *ProxyServer) SetMockEngine(eng *mock.Engine) {
	p.mockEngine = eng
}

// Start runs the proxy server listener.
func (p *ProxyServer) Start() error {
	return p.httpServer.ListenAndServe()
}

// Close terminates the proxy server.
func (p *ProxyServer) Close() error {
	return p.httpServer.Close()
}

// ServeHTTP delegates between HTTPS CONNECT tunneling and plain HTTP proxy requests.
func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *ProxyServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	destHost := r.Host
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		clientConn.Close()
		return
	}

	go p.bumpTLSConnection(clientConn, destHost)
}

func (p *ProxyServer) bumpTLSConnection(clientConn net.Conn, targetHostPort string) {
	defer clientConn.Close()

	host := targetHostPort
	if h, _, err := net.SplitHostPort(targetHostPort); err == nil {
		host = h
	}

	tlsCert, err := p.certManager.GetOrCreateCertificate(host)
	if err != nil {
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	tlsClientConn := tls.Server(clientConn, tlsConfig)
	if err := tlsClientConn.Handshake(); err != nil {
		return
	}
	defer tlsClientConn.Close()

	upstreamTLSConfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: p.insecureUpstream,
	}
	targetAddr := targetHostPort
	if !strings.Contains(targetAddr, ":") {
		targetAddr = net.JoinHostPort(targetAddr, "443")
	}

	rawConn, err := p.dialTunnel("tcp", targetAddr)
	if err != nil {
		return
	}
	upstreamConn := tls.Client(rawConn, upstreamTLSConfig)
	if err := upstreamConn.Handshake(); err != nil {
		_ = rawConn.Close()
		return
	}
	defer upstreamConn.Close()

	clientReader := bufio.NewReader(tlsClientConn)
	upstreamReader := bufio.NewReader(upstreamConn)

	for {
		reqStart := time.Now()
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			break
		}

		reqID := fmt.Sprintf("req-%d-%s", p.reqCounter.Add(1), uuid.NewString()[:8])
		traceCtx := ExtractTraceContext(req.Header)
		InjectTraceContext(req.Header, traceCtx)
		req.URL.Scheme = "https"
		req.URL.Host = host

		if isWebSocketUpgrade(req) {
			if err := req.Write(upstreamConn); err != nil {
				break
			}
			resp, err := http.ReadResponse(upstreamReader, req)
			if err != nil {
				break
			}
			if err := resp.Write(tlsClientConn); err != nil {
				break
			}

			event := &ringbuffer.TrafficEvent{
				ID:          reqID,
				Timestamp:   reqStart,
				Duration:    time.Since(reqStart),
				ClientIP:    clientConn.RemoteAddr().String(),
				Scheme:      "wss",
				Host:        host,
				Method:      req.Method,
				Path:        req.URL.Path,
				URL:         "wss://" + host + req.URL.Path,
				Proto:       "WebSocket",
				StatusCode:  resp.StatusCode,
				ReqHeaders:  cloneHeaders(req.Header),
				RespHeaders: cloneHeaders(resp.Header),
				TLS:         true,
				TLSServer:   host,
			}
			p.ringBuffer.Push(event)

			p.splice(tlsClientConn, upstreamConn)
			return
		}

		rawURL := fmt.Sprintf("https://%s%s", host, req.URL.RequestURI())

		reqBodyReader, reqCap := ReadAndCapture(req.Body, DefaultMaxBodyCaptureBytes)
		req.Body = reqBodyReader

		if err := req.Write(upstreamConn); err != nil {
			break
		}

		resp, err := http.ReadResponse(upstreamReader, req)
		if err != nil {
			break
		}

		respBodyReader, respCap := ReadAndCapture(resp.Body, DefaultMaxBodyCaptureBytes)
		resp.Body = respBodyReader

		if err := resp.Write(tlsClientConn); err != nil {
			break
		}

		duration := time.Since(reqStart)

		event := &ringbuffer.TrafficEvent{
			ID:             reqID,
			Timestamp:      reqStart,
			Duration:       duration,
			ClientIP:       clientConn.RemoteAddr().String(),
			Scheme:         "https",
			Host:           host,
			Method:         req.Method,
			Path:           req.URL.Path,
			URL:            req.URL.String(),
			Proto:          req.Proto,
			ReqHeaders:     cloneHeaders(req.Header),
			ReqBody:        reqCap.Bytes(),
			StatusCode:     resp.StatusCode,
			RespHeaders:    cloneHeaders(resp.Header),
			RespBody:       respCap.Bytes(),
			TLS:            true,
			TLSServer:      host,
			TraceID:        traceCtx.TraceID,
			SpanID:         traceCtx.SpanID,
			ClientBotClass: ClassifyClient("", req.UserAgent()),
		}

		p.ringBuffer.Push(event)

		if req.Close || resp.Close {
			break
		}
	}
}

func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	reqStart := time.Now()
	reqID := fmt.Sprintf("req-%d-%s", p.reqCounter.Add(1), uuid.NewString()[:8])

	if isWebSocketUpgrade(r) {
		p.handleWebSocketUpgrade(w, r, reqID, reqStart)
		return
	}

	var reqBodyBytes []byte
	if r.Body != nil {
		capWriter := NewBoundedCaptureWriter(DefaultMaxBodyCaptureBytes)
		tee := io.TeeReader(r.Body, capWriter)
		allBody, _ := io.ReadAll(tee)
		reqBodyBytes = capWriter.Bytes()
		r.Body = io.NopCloser(bytes.NewReader(allBody))
	}

	outReq := new(http.Request)
	*outReq = *r
	outReq.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	traceCtx := ExtractTraceContext(r.Header)
	InjectTraceContext(outReq.Header, traceCtx)

	if !outReq.URL.IsAbs() {
		outReq.URL.Scheme = "http"
		outReq.URL.Host = r.Host
	}

	removeHopByHopHeaders(outReq.Header)

	resp, err := p.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy Forwarding Error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	capWriter := NewBoundedCaptureWriter(DefaultMaxBodyCaptureBytes)
	tee := io.TeeReader(resp.Body, capWriter)

	flusher, canFlush := w.(http.Flusher)
	bufPtr := GetBuffer()
	defer PutBuffer(bufPtr)
	buf := *bufPtr
	for {
		n, err := tee.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if canFlush {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}

	duration := time.Since(reqStart)

	event := &ringbuffer.TrafficEvent{
		ID:             reqID,
		Timestamp:      reqStart,
		Duration:       duration,
		ClientIP:       r.RemoteAddr,
		Scheme:         "http",
		Host:           r.Host,
		Method:         r.Method,
		Path:           r.URL.Path,
		URL:            outReq.URL.String(),
		Proto:          r.Proto,
		ReqHeaders:     cloneHeaders(r.Header),
		ReqBody:        reqBodyBytes,
		StatusCode:     resp.StatusCode,
		RespHeaders:    cloneHeaders(resp.Header),
		RespBody:       capWriter.Bytes(),
		TLS:            false,
		TraceID:        traceCtx.TraceID,
		SpanID:         traceCtx.SpanID,
		ClientBotClass: ClassifyClient("", r.UserAgent()),
	}

	p.ringBuffer.Push(event)
}

func (p *ProxyServer) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request, reqID string, reqStart time.Time) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	destAddr := r.Host
	if !strings.Contains(destAddr, ":") {
		destAddr = net.JoinHostPort(destAddr, "80")
	}

	upstreamConn, err := p.dialTunnel("tcp", destAddr)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer upstreamConn.Close()

	if err := r.Write(upstreamConn); err != nil {
		return
	}

	p.splice(clientConn, upstreamConn)
}

func (p *ProxyServer) splice(c1, c2 net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(c1, c2)
		if tc, ok := c1.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(c2, c1)
		if tc, ok := c2.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	wg.Wait()
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, vv := range src {
		cp := make([]string, len(vv))
		copy(cp, vv)
		dst[k] = cp
	}
	return dst
}

func removeHopByHopHeaders(header http.Header) {
	hopByHop := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, h := range hopByHop {
		header.Del(h)
	}
}
