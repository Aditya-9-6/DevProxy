package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"net/http"
	"testing"
)

func TestUpstreamProxyConfiguration(t *testing.T) {
	ca, _ := certs.NewCertificateAuthority("", "")
	cm := certs.NewCertificateManager(ca)
	rb := ringbuffer.NewRingBuffer(1024)
	p := NewProxyServer(":0", cm, rb)

	tests := []struct {
		name     string
		proxyURL string
		want     string
	}{
		{"HTTP", "http://proxy.example.com:8080", "http://proxy.example.com:8080"},
		{"SOCKS5", "socks5://127.0.0.1:1080", "socks5://127.0.0.1:1080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.SetUpstreamProxy(tt.proxyURL)
			req, _ := http.NewRequest("GET", "http://google.com", nil)
			resolved, err := p.resolveForRequest(req)
			if err != nil || resolved.String() != tt.want {
				t.Errorf("Expected %s, got %v", tt.want, resolved)
			}
		})
	}
}
