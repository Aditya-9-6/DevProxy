package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newUpstreamTestProxy(t *testing.T, addr string) *ProxyServer {
	t.Helper()
	tempDir := t.TempDir()
	ca, err := certs.NewCertificateAuthority(filepath.Join(tempDir, "ca.crt"), filepath.Join(tempDir, "ca.key"))
	if err != nil {
		t.Fatalf("CA init failed: %v", err)
	}
	return NewProxyServer(addr, certs.NewCertificateManager(ca), ringbuffer.NewRingBuffer(64))
}

// directTransport never consults environment proxy variables, so the tests
// are independent of whatever HTTP_PROXY the host happens to set.
func directTransport() *http.Transport {
	return &http.Transport{MaxIdleConns: 4}
}

// relay copies both directions until either side closes.
func relay(a, aReader io.Reader, aWrite net.Conn, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(b, aReader)
		if tc, ok := b.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(aWrite, b)
		if tc, ok := aWrite.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()
	wg.Wait()
}

// mockHTTPProxy is a minimal forward proxy: plain requests are relayed,
// CONNECT requests are tunneled. It records every target it was asked to reach.
type mockHTTPProxy struct {
	server   *httptest.Server
	mu       sync.Mutex
	connects []string
	requests []string
}

func newMockHTTPProxy(t *testing.T) *mockHTTPProxy {
	t.Helper()
	m := &mockHTTPProxy{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			target := r.Host
			m.mu.Lock()
			m.connects = append(m.connects, target)
			m.mu.Unlock()

			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack unsupported", http.StatusInternalServerError)
				return
			}
			clientConn, clientRW, err := hj.Hijack()
			if err != nil {
				return
			}
			targetConn, err := net.DialTimeout("tcp", target, 3*time.Second)
			if err != nil {
				_ = clientConn.Close()
				return
			}
			if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
				_ = clientConn.Close()
				_ = targetConn.Close()
				return
			}
			go relay(clientConn, clientRW.Reader, clientConn, targetConn)
			return
		}

		m.mu.Lock()
		m.requests = append(m.requests, r.Host)
		m.mu.Unlock()

		out := r.Clone(r.Context())
		out.RequestURI = ""
		out.Close = false
		resp, err := directTransport().RoundTrip(out)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockHTTPProxy) URL() string { return m.server.URL }

func (m *mockHTTPProxy) connected() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.connects...)
}

func (m *mockHTTPProxy) forwarded() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.requests...)
}

// mockSOCKS5 implements the no-auth subset of RFC 1928 and records targets.
type mockSOCKS5 struct {
	ln      net.Listener
	mu      sync.Mutex
	targets []string
}

func newMockSOCKS5(t *testing.T) *mockSOCKS5 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("socks5 listen failed: %v", err)
	}
	m := &mockSOCKS5{ln: ln}
	go m.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return m
}

func (m *mockSOCKS5) addr() string { return m.ln.Addr().String() }

func (m *mockSOCKS5) served() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.targets...)
}

func (m *mockSOCKS5) serve() {
	for {
		conn, err := m.ln.Accept()
		if err != nil {
			return
		}
		go m.handle(conn)
	}
}

func (m *mockSOCKS5) handle(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)

	greeting := make([]byte, 2)
	if _, err := io.ReadFull(br, greeting); err != nil || greeting[0] != 0x05 {
		return
	}
	methods := make([]byte, int(greeting[1]))
	if _, err := io.ReadFull(br, methods); err != nil {
		return
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil { // no-auth accepted
		return
	}

	req := make([]byte, 4)
	if _, err := io.ReadFull(br, req); err != nil || req[0] != 0x05 || req[1] != 0x01 {
		return
	}
	var host string
	switch req[3] {
	case 0x01:
		b := make([]byte, 4)
		if _, err := io.ReadFull(br, b); err != nil {
			return
		}
		host = net.IP(b).String()
	case 0x03:
		lenb := make([]byte, 1)
		if _, err := io.ReadFull(br, lenb); err != nil {
			return
		}
		nb := make([]byte, int(lenb[0]))
		if _, err := io.ReadFull(br, nb); err != nil {
			return
		}
		host = string(nb)
	case 0x04:
		b := make([]byte, 16)
		if _, err := io.ReadFull(br, b); err != nil {
			return
		}
		host = net.IP(b).String()
	default:
		return
	}
	pb := make([]byte, 2)
	if _, err := io.ReadFull(br, pb); err != nil {
		return
	}
	target := net.JoinHostPort(host, fmt.Sprintf("%d", binary.BigEndian.Uint16(pb)))

	m.mu.Lock()
	m.targets = append(m.targets, target)
	m.mu.Unlock()

	up, err := net.DialTimeout("tcp", target, 3*time.Second)
	if err != nil {
		_, _ = conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer up.Close()
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	relay(conn, br, conn, up)
}

// ---------------------------------------------------------------------------
// (a) plain HTTP forwarding is chained through the upstream proxy
// ---------------------------------------------------------------------------

func TestUpstreamProxy_ForwardsPlainHTTPThroughProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("via-target"))
	}))
	defer target.Close()

	px := newMockHTTPProxy(t)

	proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
	if err := proxySrv.SetUpstreamProxy(px.URL()); err != nil {
		t.Fatalf("SetUpstreamProxy failed: %v", err)
	}

	devProxy := httptest.NewServer(http.HandlerFunc(proxySrv.ServeHTTP))
	defer devProxy.Close()

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(mustParse(t, devProxy.URL))},
		Timeout:   5 * time.Second,
	}
	resp, err := client.Get(target.URL + "/chained")
	if err != nil {
		t.Fatalf("request through DevProxy failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if string(body) != "via-target" {
		t.Fatalf("unexpected body %q, want %q", body, "via-target")
	}

	want := strings.TrimPrefix(target.URL, "http://")
	if got := px.forwarded(); len(got) != 1 || got[0] != want {
		t.Fatalf("upstream proxy saw %v, want [%s]", got, want)
	}
	if got := px.connected(); len(got) != 0 {
		t.Fatalf("unexpected CONNECT for plain HTTP: %v", got)
	}
}

// ---------------------------------------------------------------------------
// (b) CONNECT tunnel: dialTunnel + TLS handshake keeps SNI and verification knobs
// ---------------------------------------------------------------------------

func TestUpstreamProxy_CONNECTTunnelPreservesTLSAndSNI(t *testing.T) {
	var mu sync.Mutex
	var seenServerName string

	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tls-ok"))
	}))
	var leaf tls.Certificate
	target.TLS = &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			mu.Lock()
			seenServerName = chi.ServerName
			mu.Unlock()
			return &leaf, nil
		},
	}
	target.StartTLS()
	defer target.Close()
	leaf = target.TLS.Certificates[0]

	targetAddr := target.Listener.Addr().String()
	host, _, err := net.SplitHostPort(targetAddr)
	if err != nil {
		t.Fatalf("split target addr: %v", err)
	}

	px := newMockHTTPProxy(t)
	proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
	if err := proxySrv.SetUpstreamProxy(px.URL()); err != nil {
		t.Fatalf("SetUpstreamProxy failed: %v", err)
	}

	raw, err := proxySrv.dialTunnel("https", targetAddr)
	if err != nil {
		t.Fatalf("dialTunnel failed: %v", err)
	}
	defer raw.Close()

	if got := px.connected(); len(got) != 1 || got[0] != targetAddr {
		t.Fatalf("upstream proxy saw CONNECT %v, want [%s]", got, targetAddr)
	}

	// Same config knobs bumpTLSConnection relies on: SNI from host, opt-out of
	// upstream certificate verification, TLS 1.2 minimum.
	//
	// ServerName is a hostname rather than the loopback IP on purpose: Go
	// (correctly) omits SNI for IP literals, in tls.Dial too.
	const sniName = "upstream.test"
	tlsConn := tls.Client(raw, &tls.Config{
		ServerName:         sniName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	})
	if err := tlsConn.HandshakeContext(context.Background()); err != nil {
		t.Fatalf("TLS handshake through tunnel failed: %v", err)
	}
	if got := tlsConn.ConnectionState().ServerName; got != sniName {
		t.Fatalf("ConnectionState.ServerName = %q, want %q", got, sniName)
	}
	mu.Lock()
	sni := seenServerName
	mu.Unlock()
	if sni != sniName {
		t.Fatalf("server received SNI %q, want %q", sni, sniName)
	}
	if tlsConn.ConnectionState().Version < tls.VersionTLS12 {
		t.Fatalf("negotiated TLS version %#x below TLS 1.2", tlsConn.ConnectionState().Version)
	}

	if _, err := fmt.Fprintf(tlsConn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host); err != nil {
		t.Fatalf("write over tunnel failed: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		t.Fatalf("read over tunnel failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "tls-ok" {
		t.Fatalf("unexpected body %q", body)
	}
	_ = tlsConn.Close()

	// InsecureSkipVerify must actually gate verification: with it off the same
	// target certificate is rejected.
	strict := tls.Client(mustDialTunnel(t, proxySrv, "https", targetAddr), &tls.Config{
		ServerName: sniName,
		MinVersion: tls.VersionTLS12,
	})
	if err := strict.HandshakeContext(context.Background()); err == nil {
		_ = strict.Close()
		t.Fatal("handshake succeeded with InsecureSkipVerify disabled, want a certificate error")
	}
}

// ---------------------------------------------------------------------------
// (c) URL validation: scheme + userinfo rejection, no URL ever in the message
// ---------------------------------------------------------------------------

func TestUpstreamProxy_RejectsUnsupportedURLs(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantSub string
	}{
		{"unsupported scheme", "ftp://proxy.example:21", "scheme"},
		{"userinfo rejected", "http://user:secret@proxy.example:8080", "authentication"},
		{"userinfo without password", "http://user@proxy.example:8080", "authentication"},
		{"missing host", "http://", "host"},
		{"not a URL", "http://\x7f", "URL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
			err := proxySrv.SetUpstreamProxy(tc.raw)
			if err == nil {
				t.Fatalf("SetUpstreamProxy(%q) succeeded, want error", tc.raw)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
			// Never leak credentials or the raw URL (requirement: do not log it).
			for _, leak := range []string{tc.raw, "user", "secret", "password"} {
				if strings.Contains(err.Error(), leak) {
					t.Fatalf("error %q leaks %q", err.Error(), leak)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// (d) environment variables, incl. ALL_PROXY, and the loop guard
// ---------------------------------------------------------------------------

func TestUpstreamProxy_EnvFallbackAndLoopGuard(t *testing.T) {
	t.Run("https proxy from env", func(t *testing.T) {
		t.Setenv("REQUEST_METHOD", "")
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("NO_PROXY", "")
		t.Setenv("ALL_PROXY", "")
		t.Setenv("HTTPS_PROXY", "http://corp-proxy.example:8443")

		proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
		u, err := proxySrv.resolveUpstream("https", "api.example:443")
		if err != nil {
			t.Fatalf("resolveUpstream failed: %v", err)
		}
		if u == nil || u.String() != "http://corp-proxy.example:8443" {
			t.Fatalf("resolveUpstream = %v, want http://corp-proxy.example:8443", u)
		}
	})

	t.Run("NO_PROXY bypasses", func(t *testing.T) {
		t.Setenv("REQUEST_METHOD", "")
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("ALL_PROXY", "")
		t.Setenv("HTTPS_PROXY", "http://corp-proxy.example:8443")
		t.Setenv("NO_PROXY", "api.example")

		proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
		u, err := proxySrv.resolveUpstream("https", "api.example:443")
		if err != nil {
			t.Fatalf("resolveUpstream failed: %v", err)
		}
		if u != nil {
			t.Fatalf("resolveUpstream = %v, want nil (NO_PROXY)", u)
		}
	})

	t.Run("ALL_PROXY is honoured when no scheme proxy is set", func(t *testing.T) {
		t.Setenv("REQUEST_METHOD", "")
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")
		t.Setenv("NO_PROXY", "")
		t.Setenv("ALL_PROXY", "socks5://socks.example:1080")

		proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
		u, err := proxySrv.resolveUpstream("https", "api.example:443")
		if err != nil {
			t.Fatalf("resolveUpstream failed: %v", err)
		}
		if u == nil || u.String() != "socks5://socks.example:1080" {
			t.Fatalf("resolveUpstream = %v, want socks5://socks.example:1080", u)
		}
	})

	t.Run("env pointing at own listen address is ignored", func(t *testing.T) {
		t.Setenv("REQUEST_METHOD", "")
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("ALL_PROXY", "")
		t.Setenv("NO_PROXY", "")
		t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8080")

		proxySrv := newUpstreamTestProxy(t, ":8080")
		u, err := proxySrv.resolveUpstream("https", "api.example:443")
		if err != nil {
			t.Fatalf("resolveUpstream failed: %v", err)
		}
		if u != nil {
			t.Fatalf("resolveUpstream = %v, want nil (would loop back into DevProxy)", u)
		}
	})

	t.Run("flag pointing at own listen address is refused", func(t *testing.T) {
		proxySrv := newUpstreamTestProxy(t, ":8080")
		err := proxySrv.SetUpstreamProxy("http://127.0.0.1:8080")
		if err == nil {
			t.Fatal("SetUpstreamProxy accepted its own listen address, want an error")
		}
		if !strings.Contains(err.Error(), "loop") {
			t.Fatalf("error %q does not explain the loop", err)
		}
	})
}

// ---------------------------------------------------------------------------
// (e) SOCKS5 tunnel
// ---------------------------------------------------------------------------

func TestUpstreamProxy_SOCKS5Tunnel(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("socks-ok"))
	}))
	defer target.Close()

	socks := newMockSOCKS5(t)

	proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
	if err := proxySrv.SetUpstreamProxy("socks5://" + socks.addr()); err != nil {
		t.Fatalf("SetUpstreamProxy failed: %v", err)
	}

	targetAddr := target.Listener.Addr().String()
	conn, err := proxySrv.dialTunnel("http", targetAddr)
	if err != nil {
		t.Fatalf("dialTunnel via SOCKS5 failed: %v", err)
	}
	defer conn.Close()

	if got := socks.served(); len(got) != 1 || got[0] != targetAddr {
		t.Fatalf("socks5 proxy saw %v, want [%s]", got, targetAddr)
	}

	if _, err := fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatalf("write over socks tunnel failed: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read over socks tunnel failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "socks-ok" {
		t.Fatalf("unexpected body %q", body)
	}
}

// ---------------------------------------------------------------------------
// WebSocket upgrades (plain path) are tunneled through the upstream proxy
// ---------------------------------------------------------------------------

func TestUpstreamProxy_WebSocketTunnelsThroughProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
		buf := make([]byte, 512)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			if _, err := conn.Write(append([]byte("echo:"), buf[:n]...)); err != nil {
				return
			}
		}
	}))
	defer target.Close()

	px := newMockHTTPProxy(t)
	proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
	if err := proxySrv.SetUpstreamProxy(px.URL()); err != nil {
		t.Fatalf("SetUpstreamProxy failed: %v", err)
	}
	devProxy := httptest.NewServer(http.HandlerFunc(proxySrv.ServeHTTP))
	defer devProxy.Close()

	targetAddr := target.Listener.Addr().String()
	conn, err := net.DialTimeout("tcp", devProxy.Listener.Addr().String(), 3*time.Second)
	if err != nil {
		t.Fatalf("dial DevProxy failed: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	req := "GET " + target.URL + "/ws HTTP/1.1\r\n" +
		"Host: " + targetAddr + "\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("write upgrade request failed: %v", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read upgrade response failed: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}

	if _, err := io.WriteString(conn, "ping"); err != nil {
		t.Fatalf("write over tunnel failed: %v", err)
	}
	got := make([]byte, 5)
	if _, err := io.ReadFull(br, got); err != nil {
		t.Fatalf("read over tunnel failed: %v", err)
	}
	if string(got) != "echo:" {
		t.Fatalf("tunneled payload %q, want %q", got, "echo:")
	}

	if seen := px.connected(); len(seen) != 1 || seen[0] != targetAddr {
		t.Fatalf("upstream proxy saw CONNECT %v, want [%s]", seen, targetAddr)
	}
}

// ---------------------------------------------------------------------------
// pipelined bytes that follow the CONNECT response stay readable
// ---------------------------------------------------------------------------

func TestUpstreamProxy_CONNECTPreservesPipelinedBytes(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	requestLine := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		// Drain the rest of the header block.
		for {
			l, err := br.ReadString('\n')
			if err != nil || l == "\r\n" {
				break
			}
		}
		requestLine <- strings.TrimSpace(line)
		// A single write, so the status line, headers and payload travel in
		// one segment and bufio reads past the end of the header block.
		_, _ = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\nPIPED-BYTES"))
	}()

	proxySrv := newUpstreamTestProxy(t, "127.0.0.1:0")
	if err := proxySrv.SetUpstreamProxy("http://" + ln.Addr().String()); err != nil {
		t.Fatalf("SetUpstreamProxy failed: %v", err)
	}

	conn, err := proxySrv.dialTunnel("https", "upstream.test:443")
	if err != nil {
		t.Fatalf("dialTunnel failed: %v", err)
	}
	defer conn.Close()

	select {
	case got := <-requestLine:
		if want := "CONNECT upstream.test:443 HTTP/1.1"; got != want {
			t.Fatalf("request line = %q, want %q", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("proxy never received a CONNECT request")
	}

	// Whatever bufio read past the CONNECT response must still come out of
	// the connection we hand back -- no bytes may be swallowed.
	got := make([]byte, len("PIPED-BYTES"))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("reading pipelined bytes failed: %v", err)
	}
	if string(got) != "PIPED-BYTES" {
		t.Fatalf("got %q, want %q", got, "PIPED-BYTES")
	}
}

// ---------------------------------------------------------------------------
// misc helpers
// ---------------------------------------------------------------------------

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

func mustDialTunnel(t *testing.T, p *ProxyServer, scheme, addr string) net.Conn {
	t.Helper()
	conn, err := p.dialTunnel(scheme, addr)
	if err != nil {
		t.Fatalf("dialTunnel(%s, %s) failed: %v", scheme, addr, err)
	}
	return conn
}
