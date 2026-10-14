package waf

import (
	"sync"
	"testing"
)

func TestRevocationEpoch(t *testing.T) {
	epochState := NewRevocationEpoch()
	epoch := epochState.GetCurrentEpoch()
	epochState.RevokeAll()
	if epochState.GetCurrentEpoch() != epoch+1 {
		t.Errorf("Expected epoch increment, got %d", epochState.GetCurrentEpoch())
	}
}

func TestCookieValidator(t *testing.T) {
	cv := NewCookieValidator(1024)
	cookie := []byte("session_id=123")

	if !cv.IsTampered(cookie) {
		t.Error("Expected cookie to be marked as tampered initially")
	}

	cv.MarkValid(cookie)
	if cv.IsTampered(cookie) {
		t.Error("Expected cookie to be marked as valid after MarkValid")
	}
}

func TestCookieValidatorConcurrency(t *testing.T) {
	cv := NewCookieValidator(1024)
	cookie := []byte("concurrent_test")
	var wg sync.WaitGroup

	// Stress test: concurrent reads and writes
	for i := 0; i < 1000; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			cv.MarkValid(cookie)
		}()
		go func() {
			defer wg.Done()
			_ = cv.IsTampered(cookie)
		}()
	}
	wg.Wait()
}

func BenchmarkCookieValidator(b *testing.B) {
	cv := NewCookieValidator(1024)
	cookie := []byte("bench_test")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cv.MarkValid(cookie)
			cv.IsTampered(cookie)
		}
	})
}
