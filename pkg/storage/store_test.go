package storage

import (
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestStore_SaveAndRetrieve(t *testing.T) {
	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	reqH := make(http.Header)
	reqH.Set("User-Agent", "DevProxy-Tester")
	respH := make(http.Header)
	respH.Set("Server", "nginx/1.24.0")

	event := &ringbuffer.TrafficEvent{
		ID:          "test-req-123",
		Timestamp:   time.Now(),
		Duration:    1500 * time.Microsecond,
		ClientIP:    "127.0.0.1",
		Scheme:      "https",
		Host:        "api.github.com",
		Method:      "GET",
		Path:        "/user",
		URL:         "https://api.github.com/user",
		Proto:       "HTTP/1.1",
		StatusCode:  200,
		ReqHeaders:  reqH,
		ReqBody:     []byte(""),
		RespHeaders: respH,
		RespBody:    []byte(`{"login": "developer"}`),
		TLS:         true,
	}

	finding := &analysis.Finding{
		ID:          "f-1",
		RequestID:   "test-req-123",
		Timestamp:   time.Now(),
		Severity:    analysis.SeverityCritical,
		Category:    "EXPOSED_SECRET",
		RuleName:    "GitHub Token Leak",
		Title:       "Detected GitHub Token",
		Description: "Token was found in body",
		Evidence:    "ghp_test...",
		URL:         event.URL,
		Method:      event.Method,
	}

	err = store.SaveTransaction(event, []*analysis.Finding{finding})
	if err != nil {
		t.Fatalf("Failed to save transaction: %v", err)
	}

	// Retrieve list
	records, err := store.GetRecentRequests(10, "")
	if err != nil {
		t.Fatalf("Failed to get recent requests: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}
	if records[0].ID != "test-req-123" {
		t.Fatalf("Expected record ID test-req-123, got %s", records[0].ID)
	}
	if records[0].FindingCount != 1 {
		t.Fatalf("Expected FindingCount=1, got %d", records[0].FindingCount)
	}

	// Retrieve by ID
	full, err := store.GetRequestByID("test-req-123")
	if err != nil {
		t.Fatalf("Failed to get request by ID: %v", err)
	}
	if len(full.Findings) != 1 {
		t.Fatalf("Expected 1 finding on full request, got %d", len(full.Findings))
	}
	if full.Findings[0].RuleName != "GitHub Token Leak" {
		t.Fatalf("Finding mismatch: %v", full.Findings[0])
	}

	// Verify stats
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalRequests != 1 {
		t.Fatalf("Expected 1 total request in stats, got %d", stats.TotalRequests)
	}
	if stats.TotalFindings != 1 {
		t.Fatalf("Expected 1 total finding in stats, got %d", stats.TotalFindings)
	}
	if stats.SeverityCounts[analysis.SeverityCritical] != 1 {
		t.Fatalf("Expected 1 Critical finding, got %d", stats.SeverityCounts[analysis.SeverityCritical])
	}
}
