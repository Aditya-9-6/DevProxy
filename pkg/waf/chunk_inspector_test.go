package waf

import (
	"testing"
)

func TestChunkInspector_InspectChunk(t *testing.T) {
	signatures := [][]byte{
		[]byte("malicious_payload"),
		[]byte("<script>"),
	}

	inspector := NewChunkInspector(signatures)

	tests := []struct {
		name  string
		chunk []byte
		want  bool
	}{
		{
			name:  "clean chunk",
			chunk: []byte("hello world"),
			want:  false,
		},
		{
			name:  "malicious chunk",
			chunk: []byte("some malicious_payload here"),
			want:  true,
		},
		{
			name:  "xss chunk",
			chunk: []byte("<script>alert(1)</script>"),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inspector.InspectChunk(tt.chunk)
			if got != tt.want {
				t.Errorf("InspectChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}
