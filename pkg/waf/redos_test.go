package waf

import (
	"testing"
)

func TestReDoSGuard(t *testing.T) {
	guard := NewReDoSGuard(10)
	for i := 0; i < 10; i++ {
		if !guard.Increment() {
			t.Errorf("Guard failed at step %d", i)
		}
	}
	if guard.Increment() {
		t.Error("Guard allowed steps beyond limit")
	}
	guard.Reset()
	if !guard.Increment() {
		t.Error("Guard failed to reset")
	}
}

func TestRedosDetector(t *testing.T) {
	detector, err := NewRedosDetector("a+b", 100)
	if err != nil {
		t.Fatal(err)
	}
	if !detector.Match([]byte("aaab")) {
		t.Error("Detector failed to match valid input")
	}
}
