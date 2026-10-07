package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type ProxyServer struct {
	addr          string
	certManager   *certs.CertificateManager
	ringBuf       *ringbuffer.RingBuffer
	mockEngine    *mock.Engine
	upstreamProxy *url.URL
	loopWarnOnce  sync.Once
}

func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	return &ProxyServer{addr: addr, certManager: cm, ringBuf: rb}
}

func (p *ProxyServer) SetMockEngine(me *mock.Engine) { p.mockEngine = me }
func (p *ProxyServer) SetInsecureUpstreamTLS(b bool) {}
func (p *ProxyServer) Start() error                  { return nil }
func (p *ProxyServer) Close() error                  { return nil }

func IsGRPC(contentType string) bool {
	return strings.Contains(contentType, "application/grpc")
}
