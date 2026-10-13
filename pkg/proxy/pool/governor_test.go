package pool

import (
	"testing"
)

func TestGovernor(t *testing.T) {
	g := NewGovernor()

	g.RecordDialSuccess()
	g.RecordDialSuccess()
	g.RecordDialFailure()
	g.RecordReuse()
	g.RecordSocketExhaust()

	s := g.GetSnapshot()

	if s.DialSuccess != 2 {
		t.Errorf("expected 2 successes, got %d", s.DialSuccess)
	}
	if s.DialFailure != 1 {
		t.Errorf("expected 1 failure, got %d", s.DialFailure)
	}
	if s.ReuseCount != 1 {
		t.Errorf("expected 1 reuse, got %d", s.ReuseCount)
	}
	if s.SocketExhaust != 1 {
		t.Errorf("expected 1 exhaustion, got %d", s.SocketExhaust)
	}
}
