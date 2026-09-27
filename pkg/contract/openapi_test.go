package contract

import (
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

const sampleSpecYAML = `
openapi: 3.0.0
info:
  title: Sample User API
  version: 1.0.0
paths:
  /api/v1/users:
    get:
      summary: List users
      responses:
        "200":
          description: OK
    post:
      summary: Create user
      parameters:
        - name: X-Auth-Token
          in: header
          required: true
      responses:
        "201":
          description: Created
  /api/v1/users/{id}:
    get:
      summary: Get user by ID
      responses:
        "200":
          description: OK
        "404":
          description: Not Found
`

func TestOpenAPIValidator(t *testing.T) {
	v := NewValidator()
	if err := v.LoadSpec([]byte(sampleSpecYAML)); err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	info := v.GetInfo()
	if info["title"] != "Sample User API" {
		t.Fatalf("expected title 'Sample User API', got %v", info["title"])
	}
	if info["total_paths"] != 2 {
		t.Fatalf("expected 2 paths, got %v", info["total_paths"])
	}

	// 1. Valid Request (GET /api/v1/users with 200)
	evtValid := &ringbuffer.TrafficEvent{
		ID:         "evt-1",
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/api/v1/users",
		URL:        "https://api.test/api/v1/users",
		StatusCode: 200,
	}
	findings := v.Evaluate(evtValid)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for valid traffic, got %d", len(findings))
	}

	// 2. Path Param Match (GET /api/v1/users/42 with 200)
	evtParam := &ringbuffer.TrafficEvent{
		ID:         "evt-param",
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/api/v1/users/42",
		URL:        "https://api.test/api/v1/users/42",
		StatusCode: 200,
	}
	findings = v.Evaluate(evtParam)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for path param match, got %d", len(findings))
	}

	// 3. Shadow API (Undocumented endpoint)
	evtShadow := &ringbuffer.TrafficEvent{
		ID:         "evt-2",
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/api/v1/internal/admin",
		URL:        "https://api.test/api/v1/internal/admin",
		StatusCode: 200,
	}
	findings = v.Evaluate(evtShadow)
	if len(findings) == 0 || findings[0].Title != "Undocumented API Endpoint (Shadow API)" {
		t.Fatalf("expected Shadow API finding, got %+v", findings)
	}

	// 4. Undocumented Method (DELETE /api/v1/users)
	evtMethod := &ringbuffer.TrafficEvent{
		ID:         "evt-3",
		Timestamp:  time.Now(),
		Method:     "DELETE",
		Path:       "/api/v1/users",
		URL:        "https://api.test/api/v1/users",
		StatusCode: 200,
	}
	findings = v.Evaluate(evtMethod)
	if len(findings) == 0 || findings[0].Title != "Undocumented HTTP Method 'DELETE'" {
		t.Fatalf("expected Undocumented HTTP Method finding, got %+v", findings)
	}

	// 5. Undocumented Status Code (GET /api/v1/users returning 503)
	evtStatus := &ringbuffer.TrafficEvent{
		ID:         "evt-4",
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/api/v1/users",
		URL:        "https://api.test/api/v1/users",
		StatusCode: 503,
	}
	findings = v.Evaluate(evtStatus)
	if len(findings) == 0 || findings[0].Title != "Undocumented Response Status 503" {
		t.Fatalf("expected Undocumented Response Status finding, got %+v", findings)
	}

	// 6. Missing Required Header (POST /api/v1/users without X-Auth-Token)
	evtHeader := &ringbuffer.TrafficEvent{
		ID:         "evt-5",
		Timestamp:  time.Now(),
		Method:     "POST",
		Path:       "/api/v1/users",
		URL:        "https://api.test/api/v1/users",
		StatusCode: 201,
		ReqHeaders: make(http.Header),
	}
	findings = v.Evaluate(evtHeader)
	if len(findings) == 0 || findings[0].Title != "Missing Required Header 'X-Auth-Token'" {
		t.Fatalf("expected Missing Required Header finding, got %+v", findings)
	}
}
