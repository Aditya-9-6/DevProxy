package quic

import (
	"bytes"
	"testing"
)

func TestCoalesceAndParsePackets(t *testing.T) {
	packet1 := []byte{0xC0, 0x00, 0x00, 0x00, 0x01} // Long header / Initial mock
	packet2 := []byte{0x40, 0x02}                   // Short header mock

	originalPackets := [][]byte{packet1, packet2}

	datagram, err := CoalescePackets(originalPackets)
	if err != nil {
		t.Fatalf("failed to coalesce packets: %v", err)
	}

	parsedPackets, err := ParseCoalescedPackets(datagram)
	if err != nil {
		t.Fatalf("failed to parse coalesced datagram: %v", err)
	}

	if len(parsedPackets) == 0 {
		t.Fatalf("expected parsed packets, got none")
	}

	// Verify roundtrip fidelity
	joinedOriginal := bytes.Join(originalPackets, nil)
	if !bytes.Equal(datagram, joinedOriginal) {
		t.Errorf("datagram does not match joined original packets")
	}
}
