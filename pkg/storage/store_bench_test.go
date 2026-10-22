package storage

import (
	"net/http"
	"testing"
	"time"
	"encoding/json"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func BenchmarkStore_GetRequestByID(b *testing.B) {
	store, err := NewStore()
	if err != nil {
		b.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	reqH := make(http.Header)
	reqH.Set("User-Agent", "Mozilla/5.0")
	respH := make(http.Header)
	respH.Set("Server", "nginx/1.24.0")

	event := &ringbuffer.TrafficEvent{
		ID:          "bench-req-123",
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

	err = store.SaveTransaction(event, []*analysis.Finding{})
	if err != nil {
		b.Fatalf("Failed to save transaction: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := store.GetRequestByID("bench-req-123")
		if err != nil {
			b.Fatalf("Failed to get request: %v", err)
		}

		// Encode to JSON like the server API does to measure full impact
		_, err = json.Marshal(r)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}
