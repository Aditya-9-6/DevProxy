package analysis

import (
	"fmt"
	"net"
)

// RadixNode is a node in the bitwise Radix Trie for IP/CIDR matching.
type RadixNode struct {
	children [2]*RadixNode
	isLeaf   bool
	data     *BlocklistEntry
}

// BlocklistEntry holds metadata for matched CIDRs.
type BlocklistEntry struct {
	CIDR        string
	Category    string
	Description string
	Severity    string
}

// RadixTrie implements a bitwise Radix Trie for ultra-fast IP/CIDR matching with zero allocations during lookup.
type RadixTrie struct {
	rootIPv4 *RadixNode
	rootIPv6 *RadixNode
}

// NewRadixTrie initializes an empty RadixTrie.
func NewRadixTrie() *RadixTrie {
	return &RadixTrie{
		rootIPv4: &RadixNode{},
		rootIPv6: &RadixNode{},
	}
}

// Insert adds a CIDR network to the trie.
func (t *RadixTrie) Insert(cidrStr string, entry *BlocklistEntry) error {
	_, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("invalid CIDR %s: %w", cidrStr, err)
	}

	ones, bits := ipNet.Mask.Size()
	ip := ipNet.IP

	var root *RadixNode
	var ipBytes []byte

	if bits == 32 {
		root = t.rootIPv4
		ipBytes = ip.To4()
	} else {
		root = t.rootIPv6
		ipBytes = ip.To16()
	}

	curr := root
	for bitIndex := 0; bitIndex < ones; bitIndex++ {
		bytePos := bitIndex / 8
		bitPos := 7 - (bitIndex % 8)
		bit := (ipBytes[bytePos] >> bitPos) & 1

		if curr.children[bit] == nil {
			curr.children[bit] = &RadixNode{}
		}
		curr = curr.children[bit]
	}

	curr.isLeaf = true
	curr.data = entry
	return nil
}

// Lookup tests whether an IP matches any inserted CIDR. It performs zero heap allocations.
func (t *RadixTrie) Lookup(ip net.IP) (*BlocklistEntry, bool) {
	if ip == nil {
		return nil, false
	}

	var root *RadixNode
	var ipBytes []byte
	var bits int

	if ip4 := ip.To4(); ip4 != nil {
		root = t.rootIPv4
		ipBytes = ip4
		bits = 32
	} else if ip6 := ip.To16(); ip6 != nil {
		root = t.rootIPv6
		ipBytes = ip6
		bits = 128
	} else {
		return nil, false
	}

	curr := root
	var lastMatch *BlocklistEntry

	for bitIndex := 0; bitIndex < bits; bitIndex++ {
		if curr.isLeaf {
			lastMatch = curr.data
		}

		bytePos := bitIndex / 8
		bitPos := 7 - (bitIndex % 8)
		bit := (ipBytes[bytePos] >> bitPos) & 1

		curr = curr.children[bit]
		if curr == nil {
			break
		}
	}

	if curr != nil && curr.isLeaf {
		lastMatch = curr.data
	}

	return lastMatch, lastMatch != nil
}

// DefaultBlocklistTrie returns a RadixTrie populated with standard sensitive networks
// (cloud metadata endpoints, internal RFC1918 networks, etc.)
func DefaultBlocklistTrie() *RadixTrie {
	trie := NewRadixTrie()

	defaults := []struct {
		cidr     string
		cat      string
		desc     string
		severity string
	}{
		{"169.254.169.254/32", "CLOUD_METADATA", "AWS/GCP/Azure Cloud Instance Metadata Service (IMDSv1 risk)", "CRITICAL"},
		{"169.254.0.0/16", "LINK_LOCAL", "Link-Local address space access", "MEDIUM"},
		{"10.0.0.0/8", "INTERNAL_NETWORK", "Private RFC 1918 address space (10.0.0.0/8)", "LOW"},
		{"172.16.0.0/12", "INTERNAL_NETWORK", "Private RFC 1918 address space (172.16.0.0/12)", "LOW"},
		{"192.168.0.0/16", "INTERNAL_NETWORK", "Private RFC 1918 address space (192.168.0.0/16)", "LOW"},
		{"127.0.0.0/8", "LOOPBACK", "Localhost loopback address", "INFO"},
	}

	for _, d := range defaults {
		_ = trie.Insert(d.cidr, &BlocklistEntry{
			CIDR:        d.cidr,
			Category:    d.cat,
			Description: d.desc,
			Severity:    d.severity,
		})
	}

	return trie
}
