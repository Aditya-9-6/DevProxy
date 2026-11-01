package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractor(t *testing.T) {
	extractor := NewExtractor()

	// Test missing header -> fresh span
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	sc1 := extractor.Extract(req1)
	if !sc1.IsValid || sc1.TraceID == "" || sc1.ParentID == "" {
		t.Fatalf("expected valid fresh span context, got %+v", sc1)
	}

	// Test valid incoming header
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	validParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	req2.Header.Set(TraceParentHeader, validParent)
	req2.Header.Set(TraceStateHeader, "congo=t61rcWkgMzE")

	sc2 := extractor.Extract(req2)
	if !sc2.IsValid {
		t.Fatal("expected extracted span context to be valid")
	}
	if sc2.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("unexpected trace ID: %s", sc2.TraceID)
	}
	if sc2.ParentID != "00f067aa0ba902b7" {
		t.Errorf("unexpected parent ID: %s", sc2.ParentID)
	}
	if sc2.TraceState != "congo=t61rcWkgMzE" {
		t.Errorf("unexpected tracestate: %s", sc2.TraceState)
	}
}
