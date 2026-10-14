package parser

import (
	"bytes"
	"sync"
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
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			val, err := validator.ParseChunkSize([]byte(tt.input))
			if err != tt.expectedErr {
				t.Errorf("ParseChunkSize(%q) error = %v, expected %v", tt.input, err, tt.expectedErr)
			}
			if val != tt.expectedVal {
				t.Errorf("ParseChunkSize(%q) val = %d, expected %d", tt.input, val, tt.expectedVal)
			}
		})
	}
}

func TestChunkValidator_ValidateStream(t *testing.T) {
	t.Parallel()
	validator := NewChunkValidator(1024)
	rawChunked := "5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n"
	src := bytes.NewReader([]byte(rawChunked))
	var dst bytes.Buffer

	total, err := validator.ValidateStream(src, &dst)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if total != 11 {
		ErrorfTotal := total
		t.Errorf("expected 11 bytes validated, got %d", ErrorfTotal)
	}

	if dst.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", dst.String())
	}
}

func TestChunkValidator_ConcurrentValidation(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			validator := NewChunkValidator(1024)
			rawChunked := "3\r\nabc\r\n0\r\n\r\n"
			_, _ = validator.ValidateStream(bytes.NewReader([]byte(rawChunked)), nil)
		}()
	}
	wg.Wait()
}
