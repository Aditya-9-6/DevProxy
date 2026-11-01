package dnssec

import (
	"bytes"
	"strings"
)

// CanonicalName converts a domain name to its canonical form (lowercase).
func CanonicalName(name string) string {
	return strings.ToLower(name)
}

// CanonicalRR encodes a resource record into canonical form for DNSSEC validation.
func CanonicalRR(name string, qtype, qclass uint16, ttl uint32, rdata []byte) []byte {
	var buf bytes.Buffer
	nameLower := CanonicalName(name)
	parts := strings.Split(nameLower, ".")
	for _, p := range parts {
		if p == "" {
			continue
		}
		buf.WriteByte(byte(len(p)))
		buf.WriteString(p)
	}
	buf.WriteByte(0)

	var header [10]byte
	// Type, Class, TTL, RData length
	// In actual RFC 4034, TTL in canonical form is original TTL or RRSIG original TTL, but passed here
	buf.Write(header[:])
	buf.Write(rdata)
	return buf.Bytes()
}
