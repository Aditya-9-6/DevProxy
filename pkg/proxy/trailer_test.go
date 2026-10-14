package proxy

import (
	"testing"
)

func TestTrailerValidator(t *testing.T) {
	validator := NewTrailerValidator([]string{"Content-MD5", "X-Checksum"})

	tests := []struct {
		name     string
		data     []byte
		checksum []byte
		want     bool
	}{
		{
			name:     "Valid Trailer",
			data:     []byte("Content-MD5: abc\r\nX-Checksum: 123\r\n"),
			checksum: []byte("abc"),
			want:     true,
		},
		{
			name:     "Injection Attempt",
			data:     []byte("Key: val\r\n\r\nInjected: Header\r\n"),
			checksum: nil,
			want:     false,
		},
		{
			name:     "Invalid Checksum",
			data:     []byte("Content-MD5: abc\r\n"),
			checksum: []byte("wrong"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.ValidateTrailer(tt.data, tt.checksum)
			if got != tt.want {
				t.Errorf("ValidateTrailer() = %v, want %v", got, tt.want)
			}
		})
	}
}
