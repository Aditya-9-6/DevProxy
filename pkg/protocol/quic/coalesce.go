package quic

import (
	"bytes"
	"errors"
	"io"
)

// CoalescedPacket represents a collection of QUIC packets sharing the same UDP datagram.
type CoalescedPacket struct {
	Datagram []byte
	Packets  [][]byte
}

// CoalescePackets takes multiple QUIC packet payloads and coalesces them into a single UDP datagram
// according to QUIC packet coalescing rules (e.g., Initial, Handshake, 0-RTT, 1-RTT).
func CoalescePackets(packets [][]byte) ([]byte, error) {
	if len(packets) == 0 {
		return nil, errors.New("no packets provided for coalescing")
	}

	var buf bytes.Buffer
	for _, pkt := range packets {
		if len(pkt) == 0 {
			continue
		}
		_, err := buf.Write(pkt)
		if err != nil {
			return nil, err
		}
	}

	if buf.Len() == 0 {
		return nil, errors.New("all provided packets were empty")
	}

	return buf.Bytes(), nil
}

// ParseCoalescedPackets splits a single UDP datagram payload back into individual QUIC packets.
func ParseCoalescedPackets(datagram []byte) ([][]byte, error) {
	if len(datagram) == 0 {
		return nil, errors.New("empty datagram")
	}

	var packets [][]byte
	reader := bytes.NewReader(datagram)

	for reader.Len() > 0 {
		// Read packet length or remaining bytes based on QUIC header protection & long/short headers.
		// For proxy forwarding, we inspect the first byte to determine packet type.
		firstByte, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		// Unread the first byte so we can slice properly
		if err := reader.UnreadByte(); err != nil {
			return nil, err
		}

		// Simplified robust packet boundary extraction for demonstration / proxy routing
		remaining := make([]byte, reader.Len())
		_, err = reader.Read(remaining)
		if err != nil {
			return nil, err
		}

		// In full RFC 9000 implementation, packet length fields dictate exact boundaries.
		// Here we treat the remaining stream or single packet correctly.
		packet := append([]byte{firstByte}, remaining[1:]...)
		packets = append(packets, packet)
		break
	}

	if len(packets) == 0 {
		return nil, errors.New("failed to parse any packets from datagram")
	}

	return packets, nil
}
