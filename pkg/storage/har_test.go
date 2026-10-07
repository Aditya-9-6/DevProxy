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

func TestParseHAR(t *testing.T) {
	harJSON := `{
		"log": {
			"version": "1.2",
			"entries": [
				{
					"startedDateTime": "2023-10-06T12:00:00Z",
					"time": 12.5,
					"request": {
						"method": "POST",
						"url": "https://api.example.com/login",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"postData": {
							"text": "{\"username\": \"dev\"}"
						}
					},
					"response": {
						"status": 200,
						"content": {
							"text": "{\"status\": \"ok\"}"
						},
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						]
					},
					"serverIPAddress": "1.2.3.4"
				}
			]
		}
	}`

	events, err := ParseHAR([]byte(harJSON))
	if err != nil {
		t.Fatalf("Failed to parse HAR: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Method != "POST" {
		t.Errorf("Expected POST, got %s", event.Method)
	}
	if event.URL != "https://api.example.com/login" {
		t.Errorf("Expected URL, got %s", event.URL)
	}
	if event.Host != "1.2.3.4" {
		t.Errorf("Expected Host 1.2.3.4, got %s", event.Host)
	}
	if event.TLS != true {
		t.Errorf("Expected TLS true, got %v", event.TLS)
	}
	if event.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", event.StatusCode)
	}
	if string(event.ReqBody) != `{"username": "dev"}` {
		t.Errorf("Request body mismatch, got %s", event.ReqBody)
	}
	if string(event.RespBody) != `{"status": "ok"}` {
		t.Errorf("Response body mismatch, got %s", event.RespBody)
	}
	if event.ReqHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Request headers mismatch")
	}
	if event.RespHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Response headers mismatch")
	}
}
