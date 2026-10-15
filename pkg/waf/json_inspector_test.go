package waf

import (
	"sync"
	"testing"
)

func TestJSONInspector_InspectStream(t *testing.T) {
	tests := []struct {
		name     string
		maxDepth uint64
		input    string
		want     bool
	}{
		{"valid", 3, `{"a":{"b":{"c":1}}}`, true},
		{"invalid", 3, `{"a":{"b":{"c":{"d":1}}}}`, false},
		{"escaped_quotes", 2, `{"a": "{\"}"}`, true},
		{"empty", 2, `{}`, true},
		{"malformed", 2, `{"a": {`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := NewJSONInspector(tt.maxDepth)
			if got := inspector.InspectStream([]byte(tt.input)); got != tt.want {
				t.Errorf("InspectStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONInspector_Concurrency(t *testing.T) {
	inspector := NewJSONInspector(1)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inspector.InspectStream([]byte(`{"a":1}`)) // Should not race
		}()
	}
	wg.Wait()
}
