package replay

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

func TestHARPlaybackServer_ServeHTTP(t *testing.T) {
	sampleHAR := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "DevProxy", "version": "1.0"},
			"entries": [
				{
					"startedDateTime": "2026-10-08T00:00:00Z",
					"time": 10.0,
					"request": {
						"method": "GET",
						"url": "http://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": []
					},
					"response": {
						"status": 200,
						"statusText": "OK",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"},
							{"name": "X-Custom", "value": "devproxy"}
						],
						"content": {
							"size": 17,
							"mimeType": "application/json",
							"text": "{\"users\":[\"alice\"]}"
						}
					}
				},
				{
					"startedDateTime": "2026-10-08T00:00:00Z",
					"time": 5.0,
					"request": {
						"method": "POST",
						"url": "http://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": []
					},
					"response": {
						"status": 201,
						"statusText": "Created",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"content": {
							"size": 12,
							"mimeType": "application/json",
							"text": "{\"created\":true}"
						}
					}
				},
				{
					"startedDateTime": "2026-10-08T00:00:00Z",
					"time": 2.0,
					"request": {
						"method": "GET",
						"url": "http://api.example.com/binary",
						"httpVersion": "HTTP/1.1",
						"headers": []
					},
					"response": {
						"status": 200,
						"statusText": "OK",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/octet-stream"}
						],
						"content": {
							"size": 4,
							"mimeType": "application/octet-stream",
							"text": "AQIDBA==",
							"encoding": "base64"
						}
					}
				}
			]
		}
	}`)

	server, err := NewHARPlaybackServer(":0", sampleHAR, false)
	if err != nil {
		t.Fatalf("failed to create playback server: %v", err)
	}

	// 1. Test GET /users
	req1 := httptest.NewRequest("GET", "http://localhost/users", nil)
	rec1 := httptest.NewRecorder()
	server.ServeHTTP(rec1, req1)
	res1 := rec1.Result()

	if res1.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res1.StatusCode)
	}
	if res1.Header.Get("X-Custom") != "devproxy" {
		t.Errorf("expected header X-Custom: devproxy, got %s", res1.Header.Get("X-Custom"))
	}
	body1, _ := io.ReadAll(res1.Body)
	if string(body1) != "{\"users\":[\"alice\"]}" {
		t.Errorf("unexpected body: %s", string(body1))
	}

	// 2. Test POST /users
	req2 := httptest.NewRequest("POST", "http://localhost/users", nil)
	rec2 := httptest.NewRecorder()
	server.ServeHTTP(rec2, req2)
	res2 := rec2.Result()

	if res2.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", res2.StatusCode)
	}

	// 3. Test Base64 Binary /binary
	req3 := httptest.NewRequest("GET", "http://localhost/binary", nil)
	rec3 := httptest.NewRecorder()
	server.ServeHTTP(rec3, req3)
	res3 := rec3.Result()

	body3, _ := io.ReadAll(res3.Body)
	expectedBytes, _ := base64.StdEncoding.DecodeString("AQIDBA==")
	if string(body3) != string(expectedBytes) {
		t.Errorf("binary body mismatch: got %v, expected %v", body3, expectedBytes)
	}

	// 4. Test 404 Route Not In HAR
	req4 := httptest.NewRequest("GET", "http://localhost/not-found", nil)
	rec4 := httptest.NewRecorder()
	server.ServeHTTP(rec4, req4)
	res4 := rec4.Result()

	if res4.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown path, got %d", res4.StatusCode)
	}
}

func TestExportTrafficEventsToHAR(t *testing.T) {
	events := []*ringbuffer.TrafficEvent{
		{
			ID:          "ev-1",
			Timestamp:   time.Now(),
			Duration:    25 * time.Millisecond,
			Host:        "localhost",
			Method:      "GET",
			URL:         "http://localhost/test",
			Proto:       "HTTP/1.1",
			ReqHeaders:  http.Header{"Accept": []string{"text/html"}},
			ReqBody:     []byte(""),
			StatusCode:  200,
			RespHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			RespBody:    []byte("hello world"),
		},
	}

	harBytes, err := ExportTrafficEventsToHAR(events)
	if err != nil {
		t.Fatalf("failed to export HAR: %v", err)
	}

	parsedEvents, err := storage.ParseHAR(harBytes)
	if err != nil {
		t.Fatalf("failed to re-parse exported HAR: %v", err)
	}

	if len(parsedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(parsedEvents))
	}
	if parsedEvents[0].StatusCode != 200 {
		t.Errorf("expected status 200, got %d", parsedEvents[0].StatusCode)
	}
	if string(parsedEvents[0].RespBody) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(parsedEvents[0].RespBody))
	}
}
