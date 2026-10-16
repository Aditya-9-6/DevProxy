package analysis

import (
	"net"
	"testing"
)

func TestNewRadixTrie(t *testing.T) {
	trie := NewRadixTrie()
	if trie == nil {
		t.Fatal("NewRadixTrie returned nil")
	}
	if trie.rootIPv4 == nil {
		t.Fatal("NewRadixTrie rootIPv4 is nil")
	}
	if trie.rootIPv6 == nil {
		t.Fatal("NewRadixTrie rootIPv6 is nil")
	}
}

func TestRadixTrie_Insert(t *testing.T) {
	trie := NewRadixTrie()

	entry := &BlocklistEntry{
		CIDR:        "192.168.1.0/24",
		Category:    "TEST",
		Description: "Test entry",
		Severity:    "LOW",
	}

	// Valid IPv4
	err := trie.Insert("192.168.1.0/24", entry)
	if err != nil {
		t.Fatalf("Insert failed for valid IPv4 CIDR: %v", err)
	}

	// Valid IPv6
	err = trie.Insert("2001:db8::/32", entry)
	if err != nil {
		t.Fatalf("Insert failed for valid IPv6 CIDR: %v", err)
	}

	// Invalid CIDR
	err = trie.Insert("invalid-cidr", entry)
	if err == nil {
		t.Fatal("Insert should fail for invalid CIDR")
	}
}

func TestRadixTrie_Lookup_IPv4(t *testing.T) {
	trie := NewRadixTrie()

	entry24 := &BlocklistEntry{Category: "NET24"}
	trie.Insert("192.168.1.0/24", entry24)

	entry32 := &BlocklistEntry{Category: "HOST32"}
	trie.Insert("10.0.0.1/32", entry32)

	tests := []struct {
		ip       string
		expected bool
		category string
	}{
		{"192.168.1.50", true, "NET24"},
		{"192.168.1.255", true, "NET24"},
		{"192.168.0.255", false, ""},
		{"192.168.2.0", false, ""},
		{"10.0.0.1", true, "HOST32"},
		{"10.0.0.2", false, ""},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		match, found := trie.Lookup(ip)
		if found != tt.expected {
			t.Errorf("Lookup(%s) found=%v, expected=%v", tt.ip, found, tt.expected)
		}
		if found && match.Category != tt.category {
			t.Errorf("Lookup(%s) category=%s, expected=%s", tt.ip, match.Category, tt.category)
		}
	}
}

func TestRadixTrie_Lookup_IPv6(t *testing.T) {
	trie := NewRadixTrie()

	entry32 := &BlocklistEntry{Category: "NET32"}
	trie.Insert("2001:db8::/32", entry32)

	entry128 := &BlocklistEntry{Category: "HOST128"}
	trie.Insert("fe80::1/128", entry128)

	tests := []struct {
		ip       string
		expected bool
		category string
	}{
		{"2001:db8::1", true, "NET32"},
		{"2001:db8:ffff:ffff:ffff:ffff:ffff:ffff", true, "NET32"},
		{"2001:db9::", false, ""},
		{"fe80::1", true, "HOST128"},
		{"fe80::2", false, ""},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		match, found := trie.Lookup(ip)
		if found != tt.expected {
			t.Errorf("Lookup(%s) found=%v, expected=%v", tt.ip, found, tt.expected)
		}
		if found && match.Category != tt.category {
			t.Errorf("Lookup(%s) category=%s, expected=%s", tt.ip, match.Category, tt.category)
		}
	}
}

func TestRadixTrie_Lookup_EdgeCases(t *testing.T) {
	trie := NewRadixTrie()
	trie.Insert("192.168.1.0/24", &BlocklistEntry{Category: "NET24"})

	// Nil IP
	if _, found := trie.Lookup(nil); found {
		t.Error("Lookup(nil) should return false")
	}

	// Empty IP slice
	if _, found := trie.Lookup(net.IP{}); found {
		t.Error("Lookup(empty IP) should return false")
	}

    // IP with wrong length
    if _, found := trie.Lookup(net.IP{1, 2, 3}); found {
		t.Error("Lookup(invalid length IP) should return false")
	}
}

func TestDefaultBlocklistTrie(t *testing.T) {
	trie := DefaultBlocklistTrie()

	tests := []struct {
		ip       string
		expected bool
		category string
	}{
		// Cloud Metadata
		{"169.254.169.254", true, "CLOUD_METADATA"},
		// Link-Local
		{"169.254.0.5", true, "LINK_LOCAL"},
		// RFC 1918 (10/8)
		{"10.240.0.5", true, "INTERNAL_NETWORK"},
		{"10.0.0.0", true, "INTERNAL_NETWORK"},
		{"10.255.255.255", true, "INTERNAL_NETWORK"},
		// RFC 1918 (172.16/12)
		{"172.16.0.1", true, "INTERNAL_NETWORK"},
		{"172.31.255.255", true, "INTERNAL_NETWORK"},
		{"172.32.0.1", false, ""},
		// RFC 1918 (192.168/16)
		{"192.168.1.1", true, "INTERNAL_NETWORK"},
		{"192.169.1.1", false, ""},
		// Loopback
		{"127.0.0.1", true, "LOOPBACK"},
		// Public IPs
		{"8.8.8.8", false, ""},
		{"1.1.1.1", false, ""},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		match, found := trie.Lookup(ip)
		if found != tt.expected {
			t.Errorf("DefaultTrie.Lookup(%s) found=%v, expected=%v", tt.ip, found, tt.expected)
		}
		if found && match.Category != tt.category {
			t.Errorf("DefaultTrie.Lookup(%s) category=%s, expected=%s", tt.ip, match.Category, tt.category)
		}
	}
}
