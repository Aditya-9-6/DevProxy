package replay

import (
	"sync"
	"testing"
)

func TestAtomicBloomFilter_AddAndTest(t *testing.T) {
	size := uint64(1024)
	filter := NewAtomicBloomFilter(size)

	// Add an element
	hash := uint64(12345)
	filter.Add(hash)

	// Test if the element is present
	if !filter.Test(hash) {
		t.Errorf("Expected Test(%d) to return true, got false", hash)
	}

	// Test an element that hasn't been added (with a low probability of false positive)
	unaddedHash := uint64(54321)
	if filter.Test(unaddedHash) && unaddedHash%size != hash%size { // explicitly avoid accidental collision
		t.Errorf("Expected Test(%d) to return false, got true", unaddedHash)
	}
}

func TestAtomicBloomFilter_Reset(t *testing.T) {
	size := uint64(1024)
	filter := NewAtomicBloomFilter(size)

	hash := uint64(12345)
	filter.Add(hash)

	if !filter.Test(hash) {
		t.Errorf("Expected Test(%d) to return true before reset", hash)
	}

	filter.Reset()

	if filter.Test(hash) {
		t.Errorf("Expected Test(%d) to return false after reset", hash)
	}
}

func TestAtomicBloomFilter_ConcurrentAddAndTest(t *testing.T) {
	size := uint64(10000)
	filter := NewAtomicBloomFilter(size)
	var wg sync.WaitGroup

	numGoroutines := 100
	numOperations := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Generate a pseudo-random hash
				hash := uint64(routineID*numOperations + j)
				filter.Add(hash)
				if !filter.Test(hash) {
					t.Errorf("Concurrent add/test failed for hash %d", hash)
				}
			}
		}(i)
	}

	wg.Wait()
}
