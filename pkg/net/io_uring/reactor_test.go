package iouring

import (
	"context"
	"testing"
)

func TestNewReactor(t *testing.T) {
	r, err := NewReactor(64)
	if err != nil {
		t.Fatalf("failed to create reactor: %v", err)
	}
	defer r.Close()

	ctx := context.Background()
	_, err = r.Submit(ctx, SubmissionQueueEntry{UserData: 42})
	if err != nil {
		t.Fatalf("failed to submit: %v", err)
	}
}
