package waf

import (
	"net/netip"
	"sync/atomic"
)

// node represents a bitwise Radix Trie node.
type node struct {
	children [2]*node
	isLeaf   bool
}

// SSRFMatcher implements a lock-free, cache-aligned bitwise Radix Trie for IP/CIDR matching.
// It is designed for high-throughput security checks with zero allocations during lookup.
type SSRFMatcher struct {
	_    [64]byte // Padding to prevent false sharing
	root atomic.Pointer[node]
	_    [64]byte // Padding to prevent false sharing
}

// NewSSRFMatcher creates a new empty matcher.
func NewSSRFMatcher() *SSRFMatcher {
	m := &SSRFMatcher{}
	m.root.Store(&node{})
	return m
}

// UpdateBlocklist rebuilds the trie with the provided CIDR prefixes and swaps it atomically.
func (m *SSRFMatcher) UpdateBlocklist(prefixes []netip.Prefix) {
	newRoot := &node{}
	for _, p := range prefixes {
		insert(newRoot, p)
	}
	m.root.Store(newRoot)
}

// IsForbidden checks if the given IP address matches any forbidden CIDR range.
// This method is zero-allocation and lock-free.
func (m *SSRFMatcher) IsForbidden(addr netip.Addr) bool {
	curr := m.root.Load()
	if curr == nil {
		return false
	}

	// Use stack-allocated byte array for IP bits
	bits := addr.As16()
	maxBits := 128
	if addr.Is4() {
		maxBits = 32
		// For IPv4, we only care about the last 4 bytes of the 16-byte array
		// but netip.Addr.As16() returns IPv4-mapped IPv6 or just the 4 bytes at the end.
		// Actually, As16() for IPv4 returns [12]0 + [4]byte.
	}

	for i := 0; i < maxBits; i++ {
		if curr.isLeaf {
			return true
		}

		byteIdx := i / 8
		if addr.Is4() {
			byteIdx += 12 // Offset for IPv4 in 16-byte array
		}
		bitIdx := 7 - (i % 8)
		bit := (bits[byteIdx] >> bitIdx) & 1

		curr = curr.children[bit]
		if curr == nil {
			return false
		}
	}

	return curr.isLeaf
}

func insert(root *node, prefix netip.Prefix) {
	addr := prefix.Addr()
	ones := prefix.Bits()
	bits := addr.As16()

	curr := root
	for i := 0; i < ones; i++ {
		byteIdx := i / 8
		if addr.Is4() {
			byteIdx += 12
		}
		bitIdx := 7 - (i % 8)
		bit := (bits[byteIdx] >> bitIdx) & 1

		if curr.children[bit] == nil {
			curr.children[bit] = &node{}
		}
		curr = curr.children[bit]
	}
	curr.isLeaf = true
}
