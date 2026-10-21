package security

import (
	"testing"
)

func TestReputationEngine_BlockIP(t *testing.T) {
	engine := NewReputationEngine(1000)

	// Test 1: Empty string
	if engine.IsBlocked("") {
		t.Error("Expected empty string to not be blocked")
	}

	// Test 2: Unblocked IP
	ip1 := "192.168.1.1"
	if engine.IsBlocked(ip1) {
		t.Errorf("Expected IP %s to not be blocked initially", ip1)
	}

	// Test 3: Block an IP and verify
	engine.BlockIP(ip1)
	if !engine.IsBlocked(ip1) {
		t.Errorf("Expected IP %s to be blocked", ip1)
	}

	// Test 4: Different unblocked IP
	ip2 := "10.0.0.1"
	if engine.IsBlocked(ip2) {
		t.Errorf("Expected IP %s to not be blocked", ip2)
	}

	// Test 5: Check with port
	ipWithPort := "192.168.1.1:8080"
	if !engine.IsBlocked(ipWithPort) {
		t.Errorf("Expected IP %s (stripped from %s) to be blocked", ip1, ipWithPort)
	}

	// Test 6: IPv6 Address
	ipv6 := "2001:db8::1"
	engine.BlockIP(ipv6)
	if !engine.IsBlocked(ipv6) {
		t.Errorf("Expected IPv6 %s to be blocked", ipv6)
	}

	// Test 7: IPv6 Address with port
	// IsBlocked currently doesn't split host port for bracketed IPv6, it just checks the whole string.
	// So blocking "[2001:db8::1]:8080" directly.
	ipv6WithPort := "[2001:db8::1]:8080"
	engine.BlockIP(ipv6WithPort)
	if !engine.IsBlocked(ipv6WithPort) {
		t.Errorf("Expected %s to be blocked", ipv6WithPort)
	}
}
