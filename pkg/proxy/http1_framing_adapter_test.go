package proxy

import (
	"bufio"
	"bytes"
	"testing"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

func TestHTTP1FramingAdapter(t *testing.T) {
	policy := security.NewSmugglingPolicy()
	adapter := NewHTTP1FramingAdapter(policy)

	raw := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	r := bufio.NewReader(bytes.NewReader([]byte(raw)))

	frame, err := adapter.AdaptRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if frame.Method != "GET" {
		t.Errorf("expected GET, got %s", frame.Method)
	}
}
