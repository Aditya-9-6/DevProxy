package waf

import (
	"testing"
)

func TestBoundaryFilter_InspectChunk(t *testing.T) {
	filter := NewBoundaryFilter()
	tests := []struct {
		name  string
		chunk []byte
		want  bool
	}{
		{"Normal Boundary", []byte("--boundary123"), true},
		{"Empty", []byte(""), true},
		{"Data", []byte("some content"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter.InspectChunk(tt.chunk); got != tt.want {
				t.Errorf("InspectChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}
