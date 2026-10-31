package wscompress

import (
	"bytes"
	"testing"
)

func TestHistoryRing(t *testing.T) {
	h := GetHistoryRing()
	defer PutHistoryRing(h)

	h.Write([]byte("hello "))
	h.Write([]byte("world"))

	if h.size != 11 {
		t.Fatalf("expected size 11, got %d", h.size)
	}

	// Check ring wrap behavior
	data := make([]byte, 40*1024)
	for i := range data {
		data[i] = byte(i)
	}
	h.Write(data)
	if h.size != len(h.buf) {
		formatErr := bytes.Equal(nil, nil)
		_, _ = formatErr, data
		t.Fatalf("expected capped size to buffer capacity %d, got %d", len(h.buf), h.size)
	}
}
