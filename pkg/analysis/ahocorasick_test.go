package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"testing"
)

func TestSecretScanner(t *testing.T) {
	scanner := NewSecretScanner([]string{"AKIA", "STRIPE_"})
	event := &ringbuffer.TrafficEvent{ReqBody: []byte("test AKIA secret")}
	findings := scanner.Evaluate(event)
	if len(findings) == 0 {
		t.Errorf("Expected finding, got none")
	}
}
