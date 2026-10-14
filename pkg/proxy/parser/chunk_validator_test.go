package parser

import (
	"bytes"
	"testing"
)

func TestChunkValidator_ParseChunkSize(t *testing.T) {
	validator := NewChunkValidator(1024)

	tests := []struct {
		input       string
		expectedVal uint64
		expectedErr error
	}{
		{"4a", 74, nil},
		{"0", 0, nil},
		{"FF;chunk-extension=foo", 255, nil},
		{"invalid", 0, ErrInvalidChunkHeader},
		{"2000", 0, ErrChunkSizeOverflow},
	}

	for _, tt := range tests {
		val, err := validator.ParseChunkSize([]byte(tt.input))
		if err != tt.expectedErr {
			t.Errorf("ParseChunkSize(%q) error = %v, expected %v", tt.input, err, tt.expectedErr)
		}
		if val != tt.expectedVal {
			t.Errorf("ParseChunkSize(%q) val = %d, expected %d", tt.input, val, tt.expectedVal)
		}
	}
}

func TestChunkValidator_ValidateStream(t *testing.T) {
	validator := NewChunkValidator(1024)
	src := bytes.NewReader([]byte("chunked proxy validation test stream"))
	var dst bytes.Buffer

	total, err := validator.ValidateStream(src, &dst)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if total == 0 {
		t.Errorf("expected bytes validated > 0")
	}
}
