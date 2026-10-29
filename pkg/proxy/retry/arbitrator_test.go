package retry

import (
	"testing"
)

func TestArbitratorRetry(t *testing.T) {
	arb := NewArbitrator()

	if !arb.CanRetry(true) {
		t.Fatalf("expected idle replay-safe request to be retryable")
	}

	arb.MarkHeadersVisible()
	if !arb.CanRetry(true) {
		t.Fatalf("expected request with headers visible but uncommitted to be retryable")
	}

	arb.MarkCommitted()
	if arb.CanRetry(true) {
		t.Fatalf("expected committed request NOT to be retryable")
	}

	arb2 := NewArbitrator()
	if arb2.CanRetry(false) {
		t.Fatalf("expected non-replay-safe request NOT to be retryable")
	}
}
