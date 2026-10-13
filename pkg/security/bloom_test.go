package security

import (
	"net"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	bf := NewBloomFilter(1024, 3)
	ip := net.ParseIP("192.168.1.1").To4()

	if bf.Contains(ip) {
		t.Error("Expected false for non-existent IP")
	}

	bf.Add(ip)

	if !bf.Contains(ip) {
		t.Error("Expected true for added IP")
	}

	ip2 := net.ParseIP("10.0.0.1").To4()
	if bf.Contains(ip2) {
		// Bloom filters can have false positives, but unlikely with this small set
		t.Log("False positive detected (statistically possible)")
	}
}
