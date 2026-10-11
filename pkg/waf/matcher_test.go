package waf

import (
	"testing"
)

func TestAutomaton(t *testing.T) {
	patterns := [][]byte{[]byte("admin"), []byte("root")}
	a := NewAutomaton(patterns)
	data := []byte("login as admin or root")
	matches := make(map[int]int)
	a.Scan(data, func(pIdx, pos int) {
		matches[pIdx] = pos
	})
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
}
