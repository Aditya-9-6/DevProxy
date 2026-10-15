package quic

import (
	"sync"
	"testing"
)

func TestCreditManager_Basic(t *testing.T) {
	m := NewCreditManager(100, 200)

	if got := m.GetCredits(); got != 100 {
		t.Errorf("expected 100 credits, got %d", got)
	}

	if !m.Acquire(40) {
		t.Error(
			"expected acquisition of 40 credits to succeed",
		)
	}

	if got := m.GetCredits(); got != 60 {
		t.Errorf("expected 60 credits after acquire, got %d", got)
	}

	m.Release(50)
	if got := m.GetCredits(); got != 110 {
		t.Errorf("expected 110 credits after release, got %d", got)
	}
}

func TestCreditManager_BoundaryAndMax(t *testing.T) {
	m := NewCreditManager(50, 100)

	// Try to acquire more than available
	if m.Acquire(60) {
		t.Error(
			"expected acquisition of 60 credits with balance 50 to fail",
		)
	}
	if got := m.GetDroppedCount(); got != 1 {
		t.Errorf("expected dropped count 1, got %d", got)
	}

	// Release beyond maxWindow
	m.Release(200)
	if got := m.GetCredits(); got != 100 {
		t.Errorf(
			"expected credits to be capped at maxWindow (100), got %d",
			got,
		)
	}
}

func TestCreditManager_Concurrent(t *testing.T) {
	m := NewCreditManager(10000, 10000)
	var wg sync.WaitGroup

	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if m.Acquire(1) {
					m.Release(1)
				}
			}
		}()
	}

	wg.Wait()

	if got := m.GetCredits(); got != 10000 {
		t.Errorf(
			"expected final credits to balance back to 10000, got %d",
			got,
		)
	}
}
