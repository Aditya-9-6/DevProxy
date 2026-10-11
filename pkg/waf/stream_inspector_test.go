package waf

import (
	"context"
	"strings"
	"testing"
)

func TestInspectStream_Fragmented(t *testing.T) {
	inspector := &StreamInspector{}
	// Split "select" across two reads
	reader := strings.NewReader("sel" + "ect ")
	ctx := context.Background()

	err := inspector.InspectStream(ctx, reader, FrameText)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if inspector.BlockedCount.Load() == 0 {
		t.Error("Expected blocked count > 0 for fragmented payload")
	}
}

func TestHeuristicInspector_Encoding(t *testing.T) {
	h := &HeuristicInspector{}
	if !h.Inspect([]byte("%3cscript%3e")) {
		t.Error("Failed to detect URL encoded XSS")
	}
}
