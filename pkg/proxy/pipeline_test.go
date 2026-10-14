package proxy

import (
	"context"
	"net/http"
	"testing"
)

func TestPipelineQueue(t *testing.T) {
	q := NewPipelineQueue(2)
	req := &http.Request{}

	if !q.Enqueue(req) {
		t.Fatal("failed to enqueue")
	}

	if !q.Enqueue(req) {
		t.Fatal("failed to enqueue")
	}

	if q.Enqueue(req) {
		t.Fatal("should have blocked/dropped")
	}

	ctx := context.Background()
	_, ok := q.Dequeue(ctx)
	if !ok {
		t.Fatal("failed to dequeue")
	}

	if !q.Enqueue(req) {
		t.Fatal("failed to enqueue after dequeue")
	}
}
