package balancer

import (
	"testing"
)

func TestSelector_Select(t *testing.T) {
	addrs := []string{"10.0.0.1:80", "10.0.0.2:80"}
	s := NewSelector(addrs)

	// Test weighted distribution
	s.UpdateTargetWeight("10.0.0.1:80", 200)
	s.UpdateTargetWeight("10.0.0.2:80", 100)

	seen := make(map[string]int)
	for i := 0; i < 1000; i++ {
		target := s.Select()
		seen[target.Addr]++
	}

	// Target 1 should have roughly double the traffic of Target 2
	if seen["10.0.0.1:80"] < seen["10.0.0.2:80"] {
		t.Errorf("Expected weighted distribution, got %v", seen)
	}
}

func TestSelector_Concurrency(t *testing.T) {
	s := NewSelector([]string{"127.0.0.1:8080", "127.0.0.1:8081"})
	for i := 0; i < 100; i++ {
		go func() {
			s.Select()
		}()
		go func() {
			s.UpdateTargetWeight("127.0.0.1:8080", 100)
		}()
	}
}
