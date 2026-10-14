package waf

import (
	"sync"
	"testing"
)

func TestRevocationEpoch(t *testing.T) {
	epoch := GlobalRevocationState.GetCurrentEpoch()
	GlobalRevocationState.RevokeAll()
	if GlobalRevocationState.GetCurrentEpoch() != epoch+1 {
		t.Errorf("Expected epoch increment, got %d", GlobalRevocationState.GetCurrentEpoch())
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

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cv.MarkValid(cookie)
			if cv.IsTampered(cookie) {
				t.Errorf("Race condition detected: cookie marked valid but reported tampered")
			}
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
