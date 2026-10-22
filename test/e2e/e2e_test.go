package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/dashboard"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

type e2eBridge struct {
	store *storage.Store
	hub   *dashboard.Hub
}

func (b *e2eBridge) HandleResult(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	_ = b.store.SaveTransaction(event, findings)
	b.hub.BroadcastEvent(event, findings)
}

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func setupE2EEnvironment(t *testing.T) (*proxy.ProxyServer, *dashboard.Server, *storage.Store, *mock.Engine, string, string, func()) {
	t.Helper()

	proxyPort := getFreePort(t)
	webPort := getFreePort(t)
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	webAddr := fmt.Sprintf("127.0.0.1:%d", webPort)

	ca, err := certs.NewCertificateAuthority("", "")
	if err != nil {
		t.Fatalf("failed to create CA: %v", err)
	}
	certManager := certs.NewCertificateManager(ca)

	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ringBuf := ringbuffer.NewRingBuffer(1024)
	engine := analysis.NewSecurityRulesEngine()
	hub := dashboard.NewHub()
	go hub.Run()

	bridge := &e2eBridge{store: store, hub: hub}
	workerPool := analysis.NewAnalysisWorkerPool(ringBuf, engine, bridge, 2)
	workerPool.Start()

	mockEngine := mock.NewEngine()

	proxyServer := proxy.NewProxyServer(proxyAddr, certManager, ringBuf)
	proxyServer.SetMockEngine(mockEngine)

	dashServer := dashboard.NewServer(webAddr, store, hub, ca)
	dashServer.SetMockEngine(mockEngine)
	dashServer.SetRingBuffer(ringBuf)

	go func() {
		_ = proxyServer.Start()
	}()

	go func() {
		_ = dashServer.Start()
	}()

	// Wait for servers to be listening
	waitForPort(t, proxyAddr, 3*time.Second)
	waitForPort(t, webAddr, 3*time.Second)

	cleanup := func() {
		_ = proxyServer.Close()
		_ = dashServer.Close()
		workerPool.Stop()
		ringBuf.Close()
		store.Close()
	}

	return proxyServer, dashServer, store, mockEngine, proxyAddr, webAddr, cleanup
}

func waitForPort(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to listen", addr)
}

func TestE2E_FullProxyForwardingAndStorage(t *testing.T) {
	// 1. Upstream target HTTP server
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","source":"upstream"}`))
	}))
	defer upstream.Close()

	// 2. Start DevProxy stack
	_, _, store, _, proxyAddr, webAddr, cleanup := setupE2EEnvironment(t)
	defer cleanup()

	// 3. Configure HTTP client through DevProxy
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxyAddr))
	if err != nil {
		t.Fatalf("failed to parse proxy URL: %v", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	// 4. Send request to upstream via proxy
	req, err := http.NewRequestWithContext(context.Background(), "GET", upstream.URL+"/test/e2e/data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !upstreamCalled {
		t.Fatal("upstream server was never reached through proxy")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !strings.Contains(string(bodyBytes), "upstream") {
		t.Fatalf("unexpected response body: %s", string(bodyBytes))
	}

	// 5. Wait for asynchronous worker pool to persist to SQLite store
	var requests []*storage.RequestRecord
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		requests, err = store.GetRecentRequests(10, "")
		if err == nil && len(requests) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(requests) == 0 {
		t.Fatalf("expected request to be recorded in SQLite storage, got 0")
	}
	if requests[0].Method != "GET" || !strings.Contains(requests[0].URL, "/test/e2e/data") {
		t.Fatalf("recorded request data mismatch: %+v", requests[0])
	}

	// 6. Query Dashboard REST API
	dashResp, err := http.Get(fmt.Sprintf("http://%s/api/requests", webAddr))
	if err != nil {
		t.Fatalf("failed to query dashboard API: %v", err)
	}
	defer dashResp.Body.Close()

	if dashResp.StatusCode != http.StatusOK {
		t.Fatalf("expected dashboard status 200, got %d", dashResp.StatusCode)
	}

	var reqList []*storage.RequestRecord
	if err := json.NewDecoder(dashResp.Body).Decode(&reqList); err != nil {
		t.Fatalf("failed to decode dashboard response: %v", err)
	}
	if len(reqList) < 1 {
		t.Fatalf("dashboard API reported no requests in list")
	}
}

func TestE2E_MapLocalMockOverride(t *testing.T) {
	upstreamHit := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upstream": true}`))
	}))
	defer upstream.Close()

	_, _, _, mockEngine, proxyAddr, _, cleanup := setupE2EEnvironment(t)
	defer cleanup()

	// Add Map Local rule for /mocked-resource
	mockRule := &mock.MapLocalRule{
		ID:          "rule-mock-e2e",
		Enabled:     true,
		Pattern:     ".*/mocked-resource.*",
		StatusCode:  http.StatusCreated,
		ContentType: "application/json",
		Headers: map[string]string{
			"X-Mock-Engine": "DevProxy-E2E",
		},
		InlineBody: `{"mocked": true, "source": "devproxy-local-mock"}`,
	}
	if err := mockEngine.AddMapLocal(mockRule); err != nil {
		t.Fatalf("failed to add map local rule: %v", err)
	}

	proxyURL, _ := url.Parse(fmt.Sprintf("http://%s", proxyAddr))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(upstream.URL + "/mocked-resource")
	if err != nil {
		t.Fatalf("mocked request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected mocked status 201, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Mock-Engine") != "DevProxy-E2E" {
		t.Fatalf("expected X-Mock-Engine header, got %s", resp.Header.Get("X-Mock-Engine"))
	}
	if upstreamHit {
		t.Fatal("upstream server was called even though request was intercepted by Map Local")
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "devproxy-local-mock") {
		t.Fatalf("unexpected mocked body: %s", string(body))
	}
}

func TestE2E_PassiveSecurityFindingDetection(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Upstream leaks valid Luhn credit card: 4532-0151-1283-0366
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"account":"user-1","card":"4532-0151-1283-0366"}`))
	}))
	defer upstream.Close()

	_, _, store, _, proxyAddr, _, cleanup := setupE2EEnvironment(t)
	defer cleanup()

	proxyURL, _ := url.Parse(fmt.Sprintf("http://%s", proxyAddr))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(upstream.URL + "/api/v1/leak")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Wait for worker pool to analyze payload and store finding
	var findings []*analysis.Finding
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		findings, err = store.GetRecentFindings(10, "")
		if err == nil && len(findings) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(findings) == 0 {
		t.Fatal("expected passive security finding for leaked card, got none")
	}

	foundPCI := false
	for _, f := range findings {
		if f.Category == "PCI_COMPLIANCE" && f.Severity == analysis.SeverityCritical {
			foundPCI = true
		}
	}
	if !foundPCI {
		t.Fatalf("expected critical PCI_COMPLIANCE finding, got %+v", findings)
	}
}
