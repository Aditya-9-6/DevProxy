package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

func setupTestServer(t *testing.T) *Server {
	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	hub := NewHub()
	return NewServer("127.0.0.1:0", store, hub, nil)
}

func TestServer_MocksAPI(t *testing.T) {
	srv := setupTestServer(t)

	// 1. Add Map Local Rule
	localRuleJSON := `{"pattern": ".*/api/v1/mock", "status_code": 200, "inline_body": "{\"mocked\": true}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mocks/local", bytes.NewReader([]byte(localRuleJSON)))
	w := httptest.NewRecorder()
	srv.handleAddMapLocal(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created mock.MapLocalRule
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("failed to decode created rule: %v", err)
	}

	// 2. Add Map Remote Rule
	remoteRuleJSON := `{"pattern": ".*/api/v1/forward", "target_host": "localhost:9000"}`
	req = httptest.NewRequest(http.MethodPost, "/api/mocks/remote", bytes.NewReader([]byte(remoteRuleJSON)))
	w = httptest.NewRecorder()
	srv.handleAddMapRemote(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// 3. Add Chaos Rule
	chaosRuleJSON := `{"pattern": ".*/api/v1/flaky", "delay_ms": 100, "error_rate": 0.5, "error_status": 503}`
	req = httptest.NewRequest(http.MethodPost, "/api/mocks/chaos", bytes.NewReader([]byte(chaosRuleJSON)))
	w = httptest.NewRecorder()
	srv.handleAddChaos(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// 4. GET /api/mocks
	req = httptest.NewRequest(http.MethodGet, "/api/mocks", nil)
	w = httptest.NewRecorder()
	srv.handleMocks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &summary); err != nil {
		t.Fatalf("failed to unmarshal summary: %v", err)
	}
	if len(summary["map_local"].([]interface{})) != 1 {
		t.Fatalf("expected 1 map_local rule, got %v", summary["map_local"])
	}

	// 5. DELETE /api/mocks?id=...
	req = httptest.NewRequest(http.MethodDelete, "/api/mocks?id="+created.ID, nil)
	w = httptest.NewRecorder()
	srv.handleMocks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", w.Code)
	}

	// Verify rule deleted
	req = httptest.NewRequest(http.MethodGet, "/api/mocks", nil)
	w = httptest.NewRecorder()
	srv.handleMocks(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &summary)
	if len(summary["map_local"].([]interface{})) != 0 {
		t.Fatalf("expected 0 map_local rules after delete")
	}
}

func TestServer_ContractOpenAPI(t *testing.T) {
	srv := setupTestServer(t)

	spec := `
openapi: 3.0.0
info:
  title: Test Service
  version: 2.1.0
paths:
  /status:
    get:
      responses:
        "200":
          description: OK
`

	// 1. Upload spec
	req := httptest.NewRequest(http.MethodPost, "/api/contract/openapi", bytes.NewReader([]byte(spec)))
	w := httptest.NewRecorder()
	srv.handleContractOpenAPI(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 2. GET spec info
	req = httptest.NewRequest(http.MethodGet, "/api/contract/openapi", nil)
	w = httptest.NewRecorder()
	srv.handleContractOpenAPI(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var info map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &info)
	if info["title"] != "Test Service" || info["loaded"] != true {
		t.Fatalf("unexpected spec info: %+v", info)
	}

	// 3. DELETE /api/contract/openapi
	req = httptest.NewRequest(http.MethodDelete, "/api/contract/openapi", nil)
	w = httptest.NewRecorder()
	srv.handleContractOpenAPI(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", w.Code)
	}
}

func TestServer_JWTInspect(t *testing.T) {
	srv := setupTestServer(t)

	// Insecure alg none token
	tokenBody := `{"token": "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkFkaXR5YSJ9."}`
	req := httptest.NewRequest(http.MethodPost, "/api/jwt/inspect", bytes.NewReader([]byte(tokenBody)))
	w := httptest.NewRecorder()
	srv.handleJWTInspect(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["algorithm"] != "none" {
		t.Fatalf("expected algorithm 'none', got %v", res["algorithm"])
	}
	findings := res["findings"].([]interface{})
	if len(findings) == 0 {
		t.Fatal("expected security finding for alg none")
	}
}
