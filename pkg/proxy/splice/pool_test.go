package splice

import "testing"

func TestPoolAcquireRelease(t *testing.T) {
	p, err := NewPool(2)
	if err != nil {
		t.Fatal(err)
	}

	pipe1, ok1 := p.Acquire()
	if !ok1 {
		t.Error("expected to acquire pipe")
	}

	_, ok2 := p.Acquire()
	if !ok2 {
		t.Error("expected to acquire second pipe")
	}

	_, ok3 := p.Acquire()
	if ok3 {
		t.Error("expected pool to be exhausted")
	}

	p.Release(pipe1)
	_, ok4 := p.Acquire()
	if !ok4 {
		t.Error("expected to acquire after release")
	}
}
