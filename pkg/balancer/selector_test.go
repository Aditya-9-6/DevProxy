package balancer

import (
	"testing"
)

func TestSelector_Select(t *testing.T) {
	addrs := []string{"10.0.0.1:80", "10.0.0.2:80"}
	s := NewSelector(addrs)

	// Test round-robin distribution
	seen := make(map[string]int)
	for i := 0; i < 10; i++ {
		t := s.Select()
		seen[t.Addr]++
	}

	if seen["10.0.0.1:80"] != 5 || seen["10.0.0.2:80"] != 5 {
		t.Errorf("Expected even distribution, got %v", seen)
	}
}

func TestSelector_Concurrency(t *testing.T) {
	s := NewSelector([]string{"127.0.0.1:8080"})
	// Ensure no data races under load
	for i := 0; i < 100; i++ {
		go func() {
			s.Select()
		}()
	}
}
