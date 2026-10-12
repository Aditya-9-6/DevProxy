package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"testing"
)

func TestGraphQLGuard_Evaluate(t *testing.T) {
	guard := NewGraphQLGuard(5, 100)
	// Depth is 5: { user { posts { comments { author { name } } } } }
	event := &ringbuffer.TrafficEvent{
		ID:      "test-1",
		ReqBody: []byte("{ user { posts { comments { author { name } } } } }"),
	}

	findings := guard.Evaluate(event)
	if len(findings) != 0 {
		t.Errorf("Expected no findings for depth 5, got %d", len(findings))
	}

	// Depth is 7: { a { b { c { d { e { f { g } } } } } } }
	event.ReqBody = []byte("{ a { b { c { d { e { f { g } } } } } } }")
	findings = guard.Evaluate(event)
	if len(findings) == 0 {
		t.Error("Expected finding for excessive depth")
	}
}
