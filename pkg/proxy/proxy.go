package proxy

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

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
	loopWarnOnce     sync.Once
}

func NewProxyServer(addr string, cm *certs.CertificateManager, rb *ringbuffer.RingBuffer) *ProxyServer {
	p := &ProxyServer{
		addr:        addr,
		certManager: cm,
		ringBuffer:  rb,
		transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:   false,
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		mockEngine: mock.NewEngine(),
	}
	p.httpServer = &http.Server{Addr: addr, Handler: http.HandlerFunc(p.ServeHTTP)}
	return p
}

func extractTraceContext(r *http.Request) (string, string) {
	return r.Header.Get("traceparent"), r.Header.Get("tracestate")
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request) {}

func (p *ProxyServer) handleConnect(w http.ResponseWriter, r *http.Request) {}

func (p *ProxyServer) Start() error { return p.httpServer.ListenAndServe() }
func (p *ProxyServer) Close() error { return p.httpServer.Close() }
