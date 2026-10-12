package security

import (
	"sync"
	"testing"
)

func TestCacheAlignedBitVector_Basic(f *testing.T) {
	bv := NewCacheAlignedBitVector(128)

	if bv.Test(10) {
		f.Errorf("expected bit 10 to be false")
	}

	bv.Set(10)
	if !bv.Test(10) {
		f.Errorf("expected bit 10 to be true after Set")
	}

	bv.Clear(10)
	if bv.Test(10) {
		f.Errorf("expected bit 10 to be false after Clear")
	}
}

func TestCacheAlignedBitVector_Concurrency(t *testing.T) {
	bv := NewCacheAlignedBitVector(1000)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			idx := val % 1000
			bv.Set(idx)
			bv.Test(idx)
			bv.Clear(idx)
		}(i)
	}
	wg.Wait()
}
