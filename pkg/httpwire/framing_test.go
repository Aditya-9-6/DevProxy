package httpwire

import (
	"bufio"
	"bytes"
	"testing"
)

func TestParseRequestFraming(t *testing.T) {
	raw := "GET /path HTTP/1.1\r\nHost: example.com\r\nTransfer-Encoding: chunked\r\n\r\n"
	r := bufio.NewReader(bytes.NewReader([]byte(raw)))

	frame, err := ParseRequestFraming(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !frame.IsRequest {
		t.Errorf("expected request frame")
	}
	if frame.Method != "GET" {
		t.Errorf("expected GET, got %s", frame.Method)
	}
	if frame.URI != "/path" {
		t.Errorf("expected /path, got %s", frame.URI)
	}
	if !frame.Chunked {
		t.Errorf("expected chunked encoding flag to be true")
	}
}

func TestParseResponseFraming(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nhi"
	r := bufio.NewReader(bytes.NewReader([]byte(raw)))

	frame, err := ParseResponseFraming(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if frame.IsRequest {
		t.Errorf("expected response frame")
	}
	if frame.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", frame.StatusCode)
	}
}
