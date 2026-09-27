package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/google/uuid"
)

// ProxyServer is the high-throughput asynchronous proxy engine.
type ProxyServer struct {
	addr        string
	certManager *certs.CertificateManager
	ringBuffer  *ringbuffer.RingBuffer
	transport   *http.Transport
	httpServer  *http.Server
	reqCounter  atomic.Uint64
}

// NewProxyServer creates a new ProxyServer.
func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	// Optimized HTTP transport with connection pooling and fast keep-alives
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
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
	}

	p.httpServer = &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(p.ServeHTTP),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return p
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

// handleConnect performs on-the-fly TLS bumping (MITM decryption, cloning, re-encryption).
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

	// Tell client the tunnel is established
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		clientConn.Close()
		return
	}

	// Run TLS Bumping in a separate goroutine
	go p.bumpTLSConnection(clientConn, destHost)
}

func (p *ProxyServer) bumpTLSConnection(clientConn net.Conn, targetHostPort string) {
	defer clientConn.Close()

	// Extract clean host
	host := targetHostPort
	if h, _, err := net.SplitHostPort(targetHostPort); err == nil {
		host = h
	}

	// Get dynamically minted TLS certificate
	tlsCert, err := p.certManager.GetOrCreateCertificate(host)
	if err != nil {
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	// Wrap client connection with TLS server
	tlsClientConn := tls.Server(clientConn, tlsConfig)
	if err := tlsClientConn.Handshake(); err != nil {
		return
	}
	defer tlsClientConn.Close()

	// Dial upstream target via TLS
	upstreamTLSConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
	targetAddr := targetHostPort
	if !strings.Contains(targetAddr, ":") {
		targetAddr = net.JoinHostPort(targetAddr, "443")
	}

	upstreamConn, err := tls.Dial("tcp", targetAddr, upstreamTLSConfig)
	if err != nil {
		return
	}
	defer upstreamConn.Close()

	// Process HTTP requests inside the decrypted TLS tunnel
	clientReader := bufio.NewReader(tlsClientConn)
	upstreamReader := bufio.NewReader(upstreamConn)

	for {
		reqStart := time.Now()
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			break
		}

		reqID := fmt.Sprintf("req-%d-%s", p.reqCounter.Add(1), uuid.NewString()[:8])
		req.URL.Scheme = "https"
		req.URL.Host = host

		// Clone request body for analysis without blocking forwarding
		var reqBodyBytes []byte
		if req.Body != nil {
			var bodyBuf bytes.Buffer
			tee := io.TeeReader(req.Body, &bodyBuf)
			req.Body = io.NopCloser(tee)
			// Read body for forwarding
			reqBodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		}

		// Forward request to upstream immediately
		if err := req.Write(upstreamConn); err != nil {
			break
		}

		// Read response from upstream
		resp, err := http.ReadResponse(upstreamReader, req)
		if err != nil {
			break
		}

		// Tee response body: stream directly to client while capturing for analysis
		var respBodyBytes []byte
		if resp.Body != nil {
			respBodyBytes, _ = io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))
		}

		// Write response to client immediately (Primary Data Path)
		if err := resp.Write(tlsClientConn); err != nil {
			break
		}

		duration := time.Since(reqStart)

		// Asynchronously hand off to zero-allocation analysis pipeline
		event := &ringbuffer.TrafficEvent{
			ID:          reqID,
			Timestamp:   reqStart,
			Duration:    duration,
			ClientIP:    clientConn.RemoteAddr().String(),
			Scheme:      "https",
			Host:        host,
			Method:      req.Method,
			Path:        req.URL.Path,
			URL:         req.URL.String(),
			Proto:       req.Proto,
			ReqHeaders:  cloneHeaders(req.Header),
			ReqBody:     reqBodyBytes,
			StatusCode:  resp.StatusCode,
			RespHeaders: cloneHeaders(resp.Header),
			RespBody:    respBodyBytes,
			TLS:         true,
			TLSServer:   host,
		}

		// Completely non-blocking push: if ring buffer is full, drops instantly rather than stalling
		p.ringBuffer.Push(event)

		if req.Close || resp.Close {
			break
		}
	}
}

// handleHTTP forwards plain HTTP traffic asynchronously.
func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	reqStart := time.Now()
	reqID := fmt.Sprintf("req-%d-%s", p.reqCounter.Add(1), uuid.NewString()[:8])

	// Clone request body
	var reqBodyBytes []byte
	if r.Body != nil {
		reqBodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	}

	// Prepare outbound request
	outReq := new(http.Request)
	*outReq = *r
	outReq.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))

	// Resolve destination URL
	if !outReq.URL.IsAbs() {
		outReq.URL.Scheme = "http"
		outReq.URL.Host = r.Host
	}

	// Clean hop-by-hop headers
	removeHopByHopHeaders(outReq.Header)

	// Execute outbound request
	resp, err := p.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy Forwarding Error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Clone response headers
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Stream response body to client while capturing
	var respBodyBuf bytes.Buffer
	tee := io.TeeReader(resp.Body, &respBodyBuf)
	_, _ = io.Copy(w, tee)

	duration := time.Since(reqStart)

	// Dispatch to analysis ring buffer
	event := &ringbuffer.TrafficEvent{
		ID:          reqID,
		Timestamp:   reqStart,
		Duration:    duration,
		ClientIP:    r.RemoteAddr,
		Scheme:      "http",
		Host:        r.Host,
		Method:      r.Method,
		Path:        r.URL.Path,
		URL:         outReq.URL.String(),
		Proto:       r.Proto,
		ReqHeaders:  cloneHeaders(r.Header),
		ReqBody:     reqBodyBytes,
		StatusCode:  resp.StatusCode,
		RespHeaders: cloneHeaders(resp.Header),
		RespBody:    respBodyBuf.Bytes(),
		TLS:         false,
	}

	p.ringBuffer.Push(event)
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
