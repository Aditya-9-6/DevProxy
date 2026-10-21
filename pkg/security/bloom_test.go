package security

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter(1024, 3)
	ip := net.ParseIP("192.168.1.1").To4()

	if bf.Contains(ip) {
		t.Error("Expected false for non-existent IP")
	}

	bf.Add(ip)

	if !bf.Contains(ip) {
		t.Error("Expected true for added IP")
	}
}

func TestBlacklistManagerTable(t *testing.T) {
	t.Parallel()
	m := NewBlacklistManager()

	tests := []struct {
		name      string
		ipString  string
		shouldAdd bool
		expected  bool
	}{
		{"IPv4 Present", "10.0.0.1", true, true},
		{"IPv4 Absent", "10.0.0.2", false, false},
		{"IPv6 Present", "2001:db8::1", true, true},
		{"IPv6 Absent", "2001:db8::2", false, false},
		{"Nil IP", "", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ip net.IP
			if tc.ipString != "" {
				ip = net.ParseIP(tc.ipString)
			}
			if tc.shouldAdd {
				m.AddToBlacklist(ip)
			}
			got := m.IsBlacklisted(ip)
			if got != tc.expected {
				t.Errorf("IsBlacklisted(%s) = %v, want %v", tc.ipString, got, tc.expected)
			}
		})
	}
}

func TestBloomFilterConcurrency(t *testing.T) {
	t.Parallel()
	bf := NewBloomFilter(10000, 5)
	var wg sync.WaitGroup
	workers := 20
	iterations := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ip := net.ParseIP(fmt.Sprintf("192.168.%d.%d", workerID, j)).To4()
				bf.Add(ip)
			}
		}(i)
	}
	wg.Wait()

	for i := 0; i < workers; i++ {
		for j := 0; j < iterations; j++ {
			ip := net.ParseIP(fmt.Sprintf("192.168.%d.%d", i, j)).To4()
			if !bf.Contains(ip) {
				t.Fatalf("Expected concurrent inserted IP 192.168.%d.%d to be present", i, j)
			}
		}
	}
}

func BenchmarkBloomFilterContains(b *testing.B) {
	bf := NewBloomFilter(65536, 4)
	ip := net.ParseIP("192.168.1.100").To16()
	bf.Add(ip)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bf.Contains(ip)
	}
}
