package security

import (
	"fmt"
	"sync"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter(1024, 3)
	bf.Add([]byte("1.2.3.4"))

	if !bf.Contains([]byte("1.2.3.4")) {
		t.Error("Expected 1.2.3.4 to be in filter")
	}
	if bf.Contains([]byte("5.6.7.8")) {
		// False positive is possible but unlikely with this small set
	}
}

func TestBloomFilterConcurrency(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter(10000, 5)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			bf.Add([]byte(fmt.Sprintf("ip-%d", val)))
		}(i)
	}
	wg.Wait()
	for i := 0; i < 100; i++ {
		if !bf.Contains([]byte(fmt.Sprintf("ip-%d", i))) {
			t.Errorf("Expected ip-%d to be present", i)
		}
	}
}
