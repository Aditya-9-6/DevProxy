package proxy

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

type testBridge struct {
	store    *storage.Store
	findings chan *analysis.Finding
}

func (b *testBridge) HandleResult(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	_ = b.store.SaveTransaction(event, findings)
	for _, f := range findings {
		b.findings <- f
	}
}

func TestProxy_EndToEnd_AnalysisDecoupled(t *testing.T) {
	// 1. Upstream target HTTP server returning an insecure cookie and stack trace
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session_id=secret123; Path=/") // Insecure
		w.Header().Set("Server", "Apache/2.4.41 (Ubuntu)")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Traceback (most recent call last):
  File "app.py", line 10, in index
    raise ValueError("unexpected database failure")`))
	}))
	defer upstream.Close()

	// 2. Setup CA & CertManager
	tempDir := t.TempDir()
	ca, err := certs.NewCertificateAuthority(filepath.Join(tempDir, "ca.crt"), filepath.Join(tempDir, "ca.key"))
	if err != nil {
		t.Fatalf("CA init failed: %v", err)
	}
	cm := certs.NewCertificateManager(ca)

	// 3. RingBuffer & Storage
	rb := ringbuffer.NewRingBuffer(256)
	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("Store init failed: %v", err)
	}
	defer store.Close()

	// 4. Worker Pool
	engine := analysis.NewSecurityRulesEngine()
	findingChan := make(chan *analysis.Finding, 10)
	bridge := &testBridge{store: store, findings: findingChan}
	workerPool := analysis.NewAnalysisWorkerPool(rb, engine, bridge, 2)
	workerPool.Start()
	defer workerPool.Stop()

	// 5. Proxy Server
	proxySrv := NewProxyServer("127.0.0.1:0", cm, rb)
	testProxy := httptest.NewServer(http.HandlerFunc(proxySrv.ServeHTTP))
	defer testProxy.Close()

	// 6. Send client request configured with proxy
	proxyURL, _ := url.Parse(testProxy.URL)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(upstream.URL + "/test-endpoint")
	if err != nil {
		t.Fatalf("Client request through proxy failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	duration := time.Since(start)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected 500 status code, got %d", resp.StatusCode)
	}
	if len(bodyBytes) == 0 {
		t.Fatalf("Expected response body, got empty")
	}

	t.Logf("Response received through proxy in %v (Zero-latency forwarding)", duration)

	// 7. Verify asynchronous analysis captured the finding
	select {
	case f := <-findingChan:
		t.Logf("Captured Security Finding: [%s] %s (%s)", f.Severity, f.Title, f.RuleName)
	case <-time.After(3 * time.Second):
		t.Fatalf("Timed out waiting for asynchronous analysis result")
	}

	// Verify store has saved transaction
	requests, err := store.GetRecentRequests(10, "")
	if err != nil || len(requests) == 0 {
		t.Fatalf("Expected request to be stored in SQLite, got %d records", len(requests))
	}
}
