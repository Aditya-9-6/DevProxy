package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/google/uuid"
)

// handleConnect performs on-the-fly TLS bumping (MITM decryption, streaming, re-encryption).
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

	// Acknowledge connection
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

	rawConn, err := p.dialTunnel("https", targetAddr)
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

		// Check for WebSocket or Upgrade inside TLS tunnel
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

			// Splice raw WebSocket streams
			p.splice(tlsClientConn, upstreamConn)
			return
		}

		rawURL := fmt.Sprintf("https://%s%s", host, req.URL.RequestURI())

		// 1. Chaos Injection
		if p.mockEngine != nil {
			if injected, status, body := p.mockEngine.ApplyChaos(rawURL); injected {
				resp := &http.Response{
					StatusCode:    status,
					ProtoMajor:    1,
					ProtoMinor:    1,
					Header:        make(http.Header),
					Body:          io.NopCloser(bytes.NewReader([]byte(body))),
					ContentLength: int64(len(body)),
				}
				resp.Header.Set("Content-Type", "application/json; charset=utf-8")
				resp.Header.Set("X-DevProxy-Chaos", "true")
				_ = resp.Write(tlsClientConn)
				p.ringBuffer.Push(&ringbuffer.TrafficEvent{
					ID:          reqID,
					Timestamp:   reqStart,
					Duration:    time.Since(reqStart),
					ClientIP:    clientConn.RemoteAddr().String(),
					Scheme:      "https",
					Host:        host,
					Method:      req.Method,
					Path:        req.URL.Path,
					URL:         rawURL,
					Proto:       req.Proto,
					ReqHeaders:  cloneHeaders(req.Header),
					StatusCode:  status,
					RespHeaders: cloneHeaders(resp.Header),
					RespBody:    []byte(body),
					TLS:         true,
					TLSServer:   host,
				})
				continue
			}
		}

		// 2. Map Local Mocking
		if p.mockEngine != nil {
			if rule, mockBody, err := p.mockEngine.MatchMapLocal(rawURL); rule != nil && err == nil {
				resp := &http.Response{
					StatusCode:    rule.StatusCode,
					ProtoMajor:    1,
					ProtoMinor:    1,
					Header:        make(http.Header),
					Body:          io.NopCloser(bytes.NewReader(mockBody)),
					ContentLength: int64(len(mockBody)),
				}
				for k, v := range rule.Headers {
					resp.Header.Set(k, v)
				}
				resp.Header.Set("Content-Type", rule.ContentType)
				resp.Header.Set("X-DevProxy-Mock", "MapLocal")
				_ = resp.Write(tlsClientConn)
				p.ringBuffer.Push(&ringbuffer.TrafficEvent{
					ID:          reqID,
					Timestamp:   reqStart,
					Duration:    time.Since(reqStart),
					ClientIP:    clientConn.RemoteAddr().String(),
					Scheme:      "https",
					Host:        host,
					Method:      req.Method,
					Path:        req.URL.Path,
					URL:         rawURL,
					Proto:       req.Proto,
					ReqHeaders:  cloneHeaders(req.Header),
					StatusCode:  rule.StatusCode,
					RespHeaders: cloneHeaders(resp.Header),
					RespBody:    mockBody,
					TLS:         true,
					TLSServer:   host,
				})
				continue
			}
		}

		// 3. Map Remote Rewriting
		if p.mockEngine != nil {
			if newURL, newHost, matched := p.mockEngine.MatchMapRemote(rawURL, host); matched {
				host = newHost
				req.URL.Host = newHost
				rawURL = newURL
			}
		}

		// Stream request body with bounded capture for analysis
		reqBodyReader, reqCap := ReadAndCapture(req.Body, DefaultMaxBodyCaptureBytes)
		req.Body = reqBodyReader

		// Forward to upstream
		if err := req.Write(upstreamConn); err != nil {
			break
		}

		// Read response
		resp, err := http.ReadResponse(upstreamReader, req)
		if err != nil {
			break
		}

		// Stream response body with bounded capture (avoids freezing SSE / LLM streams and OOM)
		respBodyReader, respCap := ReadAndCapture(resp.Body, DefaultMaxBodyCaptureBytes)
		resp.Body = respBodyReader

		// Streams chunks straight to tlsClientConn
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

// handleHTTP forwards plain HTTP traffic, streaming SSE/chunked bodies and upgrading WebSockets.
func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	reqStart := time.Now()
	reqID := fmt.Sprintf("req-%d-%s", p.reqCounter.Add(1), uuid.NewString()[:8])

	// Check for WebSocket Upgrade
	if isWebSocketUpgrade(r) {
		p.handleWebSocketUpgrade(w, r, reqID, reqStart)
		return
	}

	// Stream request body with bounded capture without loading whole payload into RAM
	var reqCap *BoundedCaptureWriter
	if r.Body != nil && r.Body != http.NoBody {
		var reqReader io.ReadCloser
		reqReader, reqCap = ReadAndCapture(r.Body, DefaultMaxBodyCaptureBytes)
		r.Body = reqReader
	}

	outReq := new(http.Request)
	*outReq = *r
	outReq.Body = r.Body
	traceCtx := ExtractTraceContext(r.Header)
	InjectTraceContext(outReq.Header, traceCtx)

	if !outReq.URL.IsAbs() {
		outReq.URL.Scheme = "http"
		outReq.URL.Host = r.Host
	}

	targetURL := outReq.URL.String()
	if !outReq.URL.IsAbs() {
		targetURL = "http://" + r.Host + r.URL.RequestURI()
	}

	// 1. Chaos Injection
	if p.mockEngine != nil {
		if injected, status, body := p.mockEngine.ApplyChaos(targetURL); injected {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-DevProxy-Chaos", "true")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
			var reqBody []byte
			if reqCap != nil {
				reqBody = reqCap.Bytes()
			}
			p.ringBuffer.Push(&ringbuffer.TrafficEvent{
				ID:          reqID,
				Timestamp:   reqStart,
				Duration:    time.Since(reqStart),
				ClientIP:    r.RemoteAddr,
				Scheme:      "http",
				Host:        r.Host,
				Method:      r.Method,
				Path:        r.URL.Path,
				URL:         targetURL,
				Proto:       r.Proto,
				ReqHeaders:  cloneHeaders(r.Header),
				ReqBody:     reqBody,
				StatusCode:  status,
				RespHeaders: cloneHeaders(w.Header()),
				RespBody:    []byte(body),
				TLS:         false,
			})
			return
		}
	}

	// 2. Map Local Mocking
	if p.mockEngine != nil {
		if rule, mockBody, err := p.mockEngine.MatchMapLocal(targetURL); rule != nil && err == nil {
			for k, v := range rule.Headers {
				w.Header().Set(k, v)
			}
			w.Header().Set("Content-Type", rule.ContentType)
			w.Header().Set("X-DevProxy-Mock", "MapLocal")
			w.WriteHeader(rule.StatusCode)
			_, _ = w.Write(mockBody)
			var reqBody []byte
			if reqCap != nil {
				reqBody = reqCap.Bytes()
			}
			p.ringBuffer.Push(&ringbuffer.TrafficEvent{
				ID:          reqID,
				Timestamp:   reqStart,
				Duration:    time.Since(reqStart),
				ClientIP:    r.RemoteAddr,
				Scheme:      "http",
				Host:        r.Host,
				Method:      r.Method,
				Path:        r.URL.Path,
				URL:         targetURL,
				Proto:       r.Proto,
				ReqHeaders:  cloneHeaders(r.Header),
				ReqBody:     reqBody,
				StatusCode:  rule.StatusCode,
				RespHeaders: cloneHeaders(w.Header()),
				RespBody:    mockBody,
				TLS:         false,
			})
			return
		}
	}

	// 3. Map Remote Rewriting
	if p.mockEngine != nil {
		if newURL, newHost, matched := p.mockEngine.MatchMapRemote(targetURL, r.Host); matched {
			outReq.Host = newHost
			if parsed, parseErr := url.Parse(newURL); parseErr == nil {
				outReq.URL = parsed
			}
		}
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

	// Stream chunks immediately with flusher to support SSE / LLM tokens in real-time
	respBodyBytes := streamResponseWithCapture(w, resp, DefaultMaxBodyCaptureBytes)
	duration := time.Since(reqStart)

	var reqBodyBytes []byte
	if reqCap != nil {
		reqBodyBytes = reqCap.Bytes()
	}

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
		RespBody:       respBodyBytes,
		TLS:            false,
		TraceID:        traceCtx.TraceID,
		SpanID:         traceCtx.SpanID,
		ClientBotClass: ClassifyClient("", r.UserAgent()),
	}

	p.ringBuffer.Push(event)
}

func streamResponseWithCapture(w http.ResponseWriter, resp *http.Response, maxCapture int) []byte {
	capWriter := NewBoundedCaptureWriter(maxCapture)
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
	return capWriter.Bytes()
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

	upstreamConn, err := p.dialTunnel("http", destAddr)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer upstreamConn.Close()

	if err := r.Write(upstreamConn); err != nil {
		return
	}

	event := &ringbuffer.TrafficEvent{
		ID:         reqID,
		Timestamp:  reqStart,
		Duration:   time.Since(reqStart),
		ClientIP:   r.RemoteAddr,
		Scheme:     "ws",
		Host:       r.Host,
		Method:     r.Method,
		Path:       r.URL.Path,
		URL:        "ws://" + r.Host + r.URL.Path,
		Proto:      "WebSocket",
		StatusCode: http.StatusSwitchingProtocols,
		ReqHeaders: cloneHeaders(r.Header),
		TLS:        false,
	}
	p.ringBuffer.Push(event)

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
