package storage

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateHAR(t *testing.T) {
	records := []*RequestRecord{
		{
			ID:         "req-1",
			Timestamp:  time.Now(),
			DurationMs: 12.5,
			ClientIP:   "127.0.0.1",
			Host:       "api.example.com",
			Method:     "POST",
			URL:        "https://api.example.com/login?redirect=dashboard",
			Proto:      "HTTP/1.1",
			StatusCode: 200,
			ReqHeaders: map[string][]string{
				"Content-Type": {"application/json"},
				"User-Agent":   {"curl/8.4.0"},
			},
			ReqBody: `{"username": "dev"}`,
			RespHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			RespBody: `{"status": "ok"}`,
		},
	}

	harBytes, err := GenerateHAR(records)
	if err != nil {
		t.Fatalf("Failed to generate HAR: %v", err)
	}

	var har HAR
	if err := json.Unmarshal(harBytes, &har); err != nil {
		t.Fatalf("Failed to parse generated HAR as valid JSON: %v", err)
	}

	if har.Log.Version != "1.2" {
		t.Errorf("Expected HAR version 1.2, got %s", har.Log.Version)
	}
	if len(har.Log.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(har.Log.Entries))
	}

	entry := har.Log.Entries[0]
	if entry.Request.Method != "POST" {
		t.Errorf("Expected POST, got %s", entry.Request.Method)
	}
	if entry.Response.Status != 200 {
		t.Errorf("Expected status 200, got %d", entry.Response.Status)
	}
	if entry.Request.PostData == nil || entry.Request.PostData.Text != `{"username": "dev"}` {
		t.Errorf("PostData mismatch: %+v", entry.Request.PostData)
	}
}
