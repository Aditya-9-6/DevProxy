package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/net/proxy"
)

// upstreamDialTimeout bounds establishing an upstream connection: the TCP
// dial to the destination or proxy, and the proxy handshake itself. It
// matches the timeout the plain WebSocket path already used.
//
// It is a variable rather than a constant so tests can shorten it.
var upstreamDialTimeout = 10 * time.Second

// SetUpstreamProxy parses raw (the -upstream-proxy flag value) and routes
// DevProxy's egress through it. It rejects malformed URLs, unsupported
// schemes, credential-bearing URLs and addresses that would send traffic back
// into DevProxy itself. The parsed URL is never included in an error message.
func (p *ProxyServer) SetUpstreamProxy(raw string) error {
	u, err := parseUpstreamProxy(raw)
	if err != nil {
		return err
	}
	if p.upstreamLoop(u) {
		return fmt.Errorf("upstream proxy would loop back into DevProxy's own listen address; pick a different port")
	}
	p.upstreamProxyMu.Lock()
	p.upstreamProxy = u
	p.upstreamProxyMu.Unlock()
	return nil
}

// UpstreamProxyURL returns the string representation of the configured upstream proxy, or empty.
func (p *ProxyServer) UpstreamProxyURL() string {
	p.upstreamProxyMu.RLock()
	defer p.upstreamProxyMu.RUnlock()
	if p.upstreamProxy == nil {
		return ""
	}
	return p.upstreamProxy.String()
}

// ClearUpstreamProxy removes any explicitly configured upstream proxy.
func (p *ProxyServer) ClearUpstreamProxy() {
	p.upstreamProxyMu.Lock()
	defer p.upstreamProxyMu.Unlock()
	p.upstreamProxy = nil
}

func parseUpstreamProxy(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid -upstream-proxy value: URL could not be parsed")
	}
	switch u.Scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("unsupported upstream proxy scheme %q; use http, https, socks5 or socks5h", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("upstream proxy URL has no host")
	}
	if u.User != nil {
		return nil, fmt.Errorf("upstream proxy authentication is not supported yet; strip the credentials from the proxy URL")
	}
	return u, nil
}

// resolveUpstream returns the proxy to use when reaching host over scheme.
// A nil result means "connect directly".
func (p *ProxyServer) resolveUpstream(scheme, host string) (*url.URL, error) {
	p.upstreamProxyMu.RLock()
	configured := p.upstreamProxy
	p.upstreamProxyMu.RUnlock()

	if configured != nil {
		if p.upstreamLoop(configured) {
			return nil, fmt.Errorf("upstream proxy would loop back into DevProxy's own listen address")
		}
		return configured, nil
	}

	u := p.envUpstream(scheme, host)
	if u == nil {
		return nil, nil
	}
	if p.upstreamLoop(u) {
		p.loopWarnOnce.Do(func() {
			log.Printf("[proxy] ignoring upstream proxy that points at DevProxy's own listen address (%s); connecting directly", p.addr)
		})
		return nil, nil
	}
	return u, nil
}

// envUpstream resolves HTTPS_PROXY/HTTP_PROXY/NO_PROXY and falls back to
// ALL_PROXY when the scheme-specific variable is unset.
//
// The environment is re-read on every resolve: reading it once at startup
// would make a long-running DevProxy ignore later changes and would make the
// behaviour hard to test. It is a handful of environment lookups, so there is
// nothing to cache yet.
func (p *ProxyServer) envUpstream(scheme, host string) *url.URL {
	reqURL := &url.URL{Scheme: scheme, Host: host}
	cfg := httpproxy.FromEnvironment()
	if u, err := cfg.ProxyFunc()(reqURL); err == nil && u != nil {
		return u
	}

	all := os.Getenv("ALL_PROXY")
	if all == "" {
		all = os.Getenv("all_proxy")
	}
	if all == "" {
		return nil
	}
	// Only fall back when the scheme-specific proxy is unset, so that a
	// NO_PROXY decision is never overridden by ALL_PROXY.
	if scheme == "https" {
		if cfg.HTTPSProxy != "" {
			return nil
		}
	} else if cfg.HTTPProxy != "" {
		return nil
	}
	fallback := *cfg
	if scheme == "https" {
		fallback.HTTPSProxy = all
	} else {
		fallback.HTTPProxy = all
	}
	u, err := fallback.ProxyFunc()(reqURL)
	if err != nil {
		return nil
	}
	return u
}

// upstreamLoop reports whether using u would send traffic straight back into
// DevProxy's own listener.
func (p *ProxyServer) upstreamLoop(u *url.URL) bool {
	selfHost, selfPort, err := net.SplitHostPort(p.addr)
	if err != nil {
		return false
	}
	proxyPort := u.Port()
	if proxyPort == "" {
		proxyPort = defaultProxyPort(u.Scheme)
	}
	if proxyPort != selfPort {
		return false
	}
	switch selfHost {
	case "", "0.0.0.0", "::", "[::]":
		// Listening on every interface: any host on this port loops.
		return true
	}
	return sameHost(selfHost, u.Hostname())
}

func sameHost(a, b string) bool {
	if a == b {
		return true
	}
	return isLoopback(a) && isLoopback(b)
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defaultProxyPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default: // socks5, socks5h
		return "1080"
	}
}

// resolveForRequest adapts resolveUpstream to http.Transport's Proxy hook.
func (p *ProxyServer) resolveForRequest(r *http.Request) (*url.URL, error) {
	scheme := r.URL.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	return p.resolveUpstream(scheme, host)
}

// dialTunnel opens a raw connection to targetAddr, routed through the
// upstream proxy when one applies. It is what the TLS-bump and WebSocket
// paths use to reach their destination.
func (p *ProxyServer) dialTunnel(scheme, targetAddr string) (net.Conn, error) {
	u, err := p.resolveUpstream(scheme, targetAddr)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return net.DialTimeout("tcp", targetAddr, upstreamDialTimeout)
	}
	switch u.Scheme {
	case "http", "https":
		return dialHTTPConnect(u, targetAddr)
	case "socks5", "socks5h":
		return dialSOCKS5(u, targetAddr)
	default:
		return nil, fmt.Errorf("unsupported upstream proxy scheme %q", u.Scheme)
	}
}

// dialHTTPConnect opens a tunnel to targetAddr by issuing CONNECT against an
// HTTP(S) proxy.
func dialHTTPConnect(proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if _, _, err := net.SplitHostPort(proxyAddr); err != nil {
		proxyAddr = net.JoinHostPort(proxyAddr, defaultProxyPort(proxyURL.Scheme))
	}

	conn, err := net.DialTimeout("tcp", proxyAddr, upstreamDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to upstream proxy failed: %w", err)
	}
	// Bound the proxy handshake -- TLS to the proxy (when the proxy itself is
	// https), the CONNECT request and its response -- so a proxy that accepts
	// the connection and then stays silent cannot stall a bump goroutine.
	_ = conn.SetDeadline(time.Now().Add(upstreamDialTimeout))
	// The deadline covers the handshake only. Clear it before the tunnel is
	// handed back, so the connection the caller then streams through carries
	// no read/write deadline of its own. Note the closure reads `conn` at
	// return time, so for an https proxy it clears the TLS wrapper (and
	// through it the socket) as well.
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	if proxyURL.Scheme == "https" {
		host, _, err := net.SplitHostPort(proxyAddr)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("upstream proxy address %q is malformed", proxyAddr)
		}
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
		if err := tlsConn.HandshakeContext(context.Background()); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("TLS to upstream proxy failed: %w", err)
		}
		conn = tlsConn
	}

	// Mirrors how net/http builds its own CONNECT request.
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetAddr},
		Host:   targetAddr,
		Header: make(http.Header),
	}
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("writing CONNECT to upstream proxy failed: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("reading CONNECT response from upstream proxy failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("upstream proxy refused CONNECT with status %d", resp.StatusCode)
	}
	// The proxy may have pipelined bytes past the CONNECT response; only wrap
	// the connection when something was actually buffered.
	if br.Buffered() > 0 {
		return &bufferedConn{Conn: conn, r: br}, nil
	}
	return conn, nil
}

func dialSOCKS5(proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if _, _, err := net.SplitHostPort(proxyAddr); err != nil {
		proxyAddr = net.JoinHostPort(proxyAddr, defaultProxyPort(proxyURL.Scheme))
	}
	// Credentials are rejected by parseUpstreamProxy, so auth is always nil.
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("setting up SOCKS5 dialer failed: %w", err)
	}
	// Bound the SOCKS5 handshake with the same timeout the HTTP CONNECT path
	// uses. x/net derives a deadline from the context for the greeting and the
	// CONNECT exchange, and clears it again before returning the connection.
	ctx, cancel := context.WithTimeout(context.Background(), upstreamDialTimeout)
	defer cancel()
	cd, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("SOCKS5 dialer does not support contexts")
	}
	conn, err := cd.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		return nil, fmt.Errorf("dial through SOCKS5 upstream proxy failed: %w", err)
	}
	return conn, nil
}

// bufferedConn keeps bytes the CONNECT response reader pulled off the wire.
type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

// UpstreamPoolConfig holds connection pooling and active probe settings for upstream connections.
type UpstreamPoolConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	ProbeInterval       time.Duration
	KeepAlive           time.Duration
	Insecure            bool
}

// DefaultUpstreamPoolConfig returns default pooled transport settings.
func DefaultUpstreamPoolConfig() UpstreamPoolConfig {
	return UpstreamPoolConfig{
		MaxIdleConns:        500,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		ProbeInterval:       30 * time.Second,
		KeepAlive:           15 * time.Second,
		Insecure:            false,
	}
}

// PooledTransport wraps http.Transport with active keepalive health-check probes
// to eliminate 502 Bad Gateway race conditions from stale or closed connections.
type PooledTransport struct {
	*http.Transport
	probeInterval time.Duration
	stopChan      chan struct{}
	closeOnce     sync.Once
}

// NewPooledTransport creates an optimized upstream transport with active connection keepalive probes.
func NewPooledTransport(cfg UpstreamPoolConfig) *PooledTransport {
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 500
	}
	if cfg.MaxIdleConnsPerHost <= 0 {
		cfg.MaxIdleConnsPerHost = 100
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 15 * time.Second
	}
	if cfg.ProbeInterval <= 0 {
		cfg.ProbeInterval = 30 * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   upstreamDialTimeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}

	return &PooledTransport{
		Transport:     transport,
		probeInterval: cfg.ProbeInterval,
		stopChan:      make(chan struct{}),
	}
}

// StartProbes runs periodic health checks on idle connections to prevent silent TCP drops.
func (pt *PooledTransport) StartProbes(ctx context.Context) {
	ticker := time.NewTicker(pt.probeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pt.Transport.CloseIdleConnections()
		case <-ctx.Done():
			return
		case <-pt.stopChan:
			return
		}
	}
}

// Close stops the active probe goroutine and closes idle connections.
func (pt *PooledTransport) Close() {
	pt.closeOnce.Do(func() {
		close(pt.stopChan)
		pt.Transport.CloseIdleConnections()
	})
}
