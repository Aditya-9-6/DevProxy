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

var upstreamDialTimeout = 10 * time.Second

type UpstreamTransport struct {
	*http.Transport
	probeInterval time.Duration
	stopChan      chan struct{}
	closeOnce     sync.Once
}

func NewUpstreamTransport(insecure bool) *UpstreamTransport {
	return &UpstreamTransport{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   upstreamDialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: insecure},
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		probeInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

func (u *UpstreamTransport) StartProbes(ctx context.Context) {
	ticker := time.NewTicker(u.probeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			u.Transport.CloseIdleConnections()
		case <-ctx.Done():
			return
		case <-u.stopChan:
			return
		}
	}
}

func (u *UpstreamTransport) Close() {
	u.closeOnce.Do(func() {
		close(u.stopChan)
	})
}

func ParseUpstreamProxy(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" && u.Scheme != "socks5h") {
		return nil, fmt.Errorf("invalid or unsupported proxy scheme")
	}
	return u, nil
}

func DialTunnel(u *url.URL, targetAddr string) (net.Conn, error) {
	if u == nil {
		return net.DialTimeout("tcp", targetAddr, upstreamDialTimeout)
	}
	switch u.Scheme {
	case "http", "https":
		return dialHTTPConnect(u, targetAddr)
	case "socks5", "socks5h":
		return dialSOCKS5(u, targetAddr)
	}
	return nil, fmt.Errorf("unsupported scheme")
}

func dialHTTPConnect(proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyURL.Host, upstreamDialTimeout)
	if err != nil {
		return nil, err
	}
	req := &http.Request{Method: "CONNECT", URL: &url.URL{Opaque: targetAddr}, Host: targetAddr}
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil || resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("proxy error")
	}
	return conn, nil
}

func dialSOCKS5(proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, nil, nil)
	if err != nil {
		return nil, err
	}
	return dialer.Dial("tcp", targetAddr)
}
