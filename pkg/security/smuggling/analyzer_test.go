package smuggling

import (
	"testing"
)

func TestSmugglingAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()
	state := analyzer.GetState()
	defer analyzer.PutState(state)

	state.ObserveHeader("Content-Length", "100")
	state.ObserveHeader("Transfer-Encoding", "chunked")

	if !state.IsSmugglingRisk() {
		t.Error("Expected smuggling risk detected")
	}

	state.Reset()
	if state.IsSmugglingRisk() {
		t.Error("Expected state to be clean after reset")
	}
}
