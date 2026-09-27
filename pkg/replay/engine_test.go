package replay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

func TestReplay_ExecutionAndDiff(t *testing.T) {
	// Mock server that returns custom response based on query
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", r.Header.Get("X-Custom"))
		if r.Method == "POST" {
			b, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"received": "` + string(b) + `"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer srv.Close()

	replayer := NewReplayer(5*time.Second, true)

	orig := &storage.RequestRecord{
		ID:         "orig-123",
		Timestamp:  time.Now(),
		DurationMs: 15.0,
		Method:     "GET",
		URL:        srv.URL + "/api/test",
		StatusCode: http.StatusOK,
		ReqHeaders: map[string][]string{
			"X-Custom": {"test-val"},
		},
		RespHeaders: map[string][]string{
			"X-Custom": {"test-val"},
		},
		RespBody: `{"status": "ok"}`,
	}

	// 1. Replay with identical request
	res, err := replayer.Replay(orig, nil)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if res.Diff.StatusChanged {
		t.Fatalf("expected status not changed")
	}
	if res.Diff.BodyChanged {
		t.Fatalf("expected body not changed")
	}

	// 2. Replay with method & body override
	override := &ReplayOverride{
		Method: "POST",
		Body:   "hello-world",
		Headers: map[string][]string{
			"X-Custom": {"new-val"},
		},
	}
	res2, err := replayer.Replay(orig, override)
	if err != nil {
		t.Fatalf("replay with override failed: %v", err)
	}
	if !res2.Diff.StatusChanged {
		t.Fatalf("expected status changed to 201 Created")
	}
	if res2.Replayed.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res2.Replayed.StatusCode)
	}
	if !res2.Diff.BodyChanged {
		t.Fatalf("expected body changed")
	}
}

func TestComputeDiff_JSONSemanticEquivalence(t *testing.T) {
	orig := &storage.RequestRecord{
		StatusCode:  200,
		RespHeaders: map[string][]string{"Content-Type": {"application/json"}},
		RespBody:    `{"a": 1, "b": 2}`,
	}
	replayed := &ReplayedResponse{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       "{\n  \"b\": 2,\n  \"a\": 1\n}", // same JSON, different formatting
	}

	diff := ComputeDiff(orig, replayed)
	if diff.StatusChanged {
		t.Fatal("expected status unchanged")
	}
	if diff.BodyChanged {
		t.Fatal("expected JSON to be marked semantically equivalent")
	}

	// Verify Header diffs
	replayed.Headers = map[string][]string{
		"Content-Type": {"application/json"},
		"X-New-Header": {"123"},
	}
	diff2 := ComputeDiff(orig, replayed)
	if len(diff2.HeadersAdded) != 1 || diff2.HeadersAdded[0] != "X-New-Header" {
		t.Fatalf("expected X-New-Header added: %+v", diff2.HeadersAdded)
	}
}
