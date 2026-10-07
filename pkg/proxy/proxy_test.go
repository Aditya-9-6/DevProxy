package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestUpstreamProxyChaining(t *testing.T) {
	// Mock upstream proxy
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// Setup DevProxy
	ca, _ := certs.NewCertificateAuthority("", "")
	rb := ringbuffer.NewRingBuffer(1024)
	p := NewProxyServer(":0", certs.NewCertificateManager(ca), rb)

	// Configure upstream
	err := p.SetUpstreamProxy(upstream.URL)
	if err != nil {
		t.Fatalf("Failed to set upstream proxy: %v", err)
	}

	// Verify resolution
	proxyURL, err := p.resolveUpstream("http", "example.com")
	if err != nil {
		t.Fatalf("Resolution error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != upstream.URL {
		t.Errorf("Expected proxy URL %s, got %v", upstream.URL, proxyURL)
	}
}

func TestUpstreamLoopDetection(t *testing.T) {
	ca, _ := certs.NewCertificateAuthority("", "")
	rb := ringbuffer.NewRingBuffer(1024)
	p := NewProxyServer(":8080", certs.NewCertificateManager(ca), rb)

	// Test loop detection
	loopURL, _ := url.Parse("http://127.0.0.1:8080")
	if !p.upstreamLoop(loopURL) {
		t.Error("Expected loop detection for 127.0.0.1:8080")
	}

	// Test valid proxy
	validURL, _ := url.Parse("http://127.0.0.1:9090")
	if p.upstreamLoop(validURL) {
		t.Error("Did not expect loop detection for 127.0.0.1:9090")
	}
}
