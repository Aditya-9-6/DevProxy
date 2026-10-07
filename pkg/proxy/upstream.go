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
	"time"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/net/proxy"
)

var upstreamDialTimeout = 10 * time.Second

func (p *ProxyServer) SetUpstreamProxy(raw string) error {
	u, err := parseUpstreamProxy(raw)
	if err != nil {
		return err
	}
	if p.upstreamLoop(u) {
		return fmt.Errorf("upstream proxy would loop back into DevProxy's own listen address; pick a different port")
	}
	p.upstreamProxy = u
	return nil
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

func (p *ProxyServer) resolveUpstream(scheme, host string) (*url.URL, error) {
	if p.upstreamProxy != nil {
		if p.upstreamLoop(p.upstreamProxy) {
			return nil, fmt.Errorf("upstream proxy would loop back into DevProxy's own listen address")
		}
		return p.upstreamProxy, nil
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
	default:
		return "1080"
	}
}

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

func dialHTTPConnect(proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if _, _, err := net.SplitHostPort(proxyAddr); err != nil {
		proxyAddr = net.JoinHostPort(proxyAddr, defaultProxyPort(proxyURL.Scheme))
	}

	conn, err := net.DialTimeout("tcp", proxyAddr, upstreamDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to upstream proxy failed: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(upstreamDialTimeout))
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
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("setting up SOCKS5 dialer failed: %w", err)
	}
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

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.r.Read(p) }
