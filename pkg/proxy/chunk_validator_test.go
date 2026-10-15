package proxy

import (
	"testing"
)

func TestChunkValidator(t *testing.T) {
	tests := []struct {
		name     string
		maxDepth uint64
		maxExt   uint64
		steps    []struct {
			depth int
			ext   uint64
			want  bool
		}
	}{{"Normal", 5, 10, []struct {
		depth int
		ext   uint64
		want  bool
	}{{1, 1, true}, {1, 1, true}, {-1, 0, true}}},
		{"DepthOverflow", 2, 10, []struct {
			depth int
			ext   uint64
			want  bool
		}{{1, 0, true}, {2, 0, false}}},
		{"ExtOverflow", 2, 1, []struct {
			depth int
			ext   uint64
			want  bool
		}{{0, 2, false}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewChunkValidator(tt.maxDepth, tt.maxExt)
			for _, step := range tt.steps {
				err := v.ValidateChunk(step.depth, step.ext)
				if (err == nil) != step.want {
					t.Errorf("ValidateChunk() error = %v, want %v", err, step.want)
				}
			}
		})
	}
}
