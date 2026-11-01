package security

import (
	"testing"
	"github.com/Aditya-9-6/DevProxy/pkg/httpwire"
)

func TestSmugglingPolicyInspect(t *testing.T) {
	policy := NewSmugglingPolicy()

	// Frame with both CL and TE should fail
	frame := &httpwire.Frame{
		Headers: [][2]string{
			{"Content-Length", "10"},
			{"Transfer-Encoding", "chunked"},
		},
	}

	err := policy.Inspect(frame)
	if err == nil {
		t.Errorf("expected smuggling detection error for conflicting headers, got nil")
	}
}
