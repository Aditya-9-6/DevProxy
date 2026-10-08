package mock

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandaloneServer_ServeHTTP(t *testing.T) {
	eng := NewEngine()
	rule := &MapLocalRule{
		Enabled:     true,
		Pattern:     "^/api/v1/status$",
		StatusCode:  200,
		ContentType: "application/json",
		InlineBody:  `{"status":"ok"}`,
	}
	_ = eng.AddMapLocal(rule)

	server := NewStandaloneServer(":0", eng)

	// Test intercepted rule
	req1 := httptest.NewRequest("GET", "http://localhost/api/v1/status", nil)
	rec1 := httptest.NewRecorder()
	server.ServeHTTP(rec1, req1)
	res1 := rec1.Result()

	if res1.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res1.StatusCode)
	}
	body1, _ := io.ReadAll(res1.Body)
	if string(body1) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", string(body1))
	}

	// Test default fallback
	req2 := httptest.NewRequest("GET", "http://localhost/unknown", nil)
	rec2 := httptest.NewRecorder()
	server.ServeHTTP(rec2, req2)
	res2 := rec2.Result()

	if res2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 fallback, got %d", res2.StatusCode)
	}
}
