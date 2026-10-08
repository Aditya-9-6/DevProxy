package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestUpstreamProxyConfiguration(t *testing.T) {
	ca, _ := certs.NewCertificateAuthority("", "")
	cm := certs.NewCertificateManager(ca)
	rb := ringbuffer.NewRingBuffer(1024)
	p := NewProxyServer(":0", cm, rb)

	// Test valid proxy URL
	proxyURL := "http://proxy.example.com:8080"
	err := p.SetUpstreamProxy(proxyURL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify resolution
	req, _ := http.NewRequest("GET", "http://google.com", nil)
	resolved, err := p.resolveForRequest(req)
	if err != nil || resolved.String() != proxyURL {
		t.Errorf("Expected %s, got %v", proxyURL, resolved)
	}

	// Test SOCKS5
	socksURL := "socks5://127.0.0.1:1080"
	p.SetUpstreamProxy(socksURL)
	resolved, _ = p.resolveForRequest(req)
	if resolved.String() != socksURL {
		t.Errorf("Expected %s, got %v", socksURL, resolved)
	}
}
