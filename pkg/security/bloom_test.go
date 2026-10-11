package security

import "testing"

func TestBloomFilter(t *testing.T) {
	bf := NewBloomFilter(1024, 3)
	bf.Add([]byte("1.2.3.4"))

	if !bf.Contains([]byte("1.2.3.4")) {
		t.Error("Expected 1.2.3.4 to be in filter")
	}
	if bf.Contains([]byte("5.6.7.8")) {
		// False positive is possible but unlikely with this small set
	}
}
