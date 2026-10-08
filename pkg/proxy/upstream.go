package proxy

import (
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
	u, err := url.Parse(raw)
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
