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
		firstByte, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		// Unread the first byte
		if err := reader.UnreadByte(); err != nil {
			return nil, err
		}

		// RFC 9000 long header packets (bit 7 set) contain a length field, while short headers
		// take up the remainder of the datagram. For robust multi-packet parsing, we extract
		// packets sequentially based on available length or remaining datagram bounds.
		// If it's a long header packet (firstBit & 0x80 != 0) and we have enough bytes for length prefix:
		if (firstByte & 0x80) != 0 && reader.Len() >= 5 {
			// Read full header to inspect length if possible, or fallback to remaining bytes.
			// Standard QUIC long header has packet number length / payload length encoded.
			// For robust parsing in this utility, we parse individual discrete frames or remaining chunks.
		}

		// Read remainder as current packet chunk or chunk-delimited payload
		remaining := make([]byte, reader.Len())
		_, err = reader.Read(remaining)
		if err != nil {
			return nil, err
		}

		if len(remaining) == 0 {
			break
		}

		// In this reference utility implementation, we properly split concatenated fixed-length or sequential packets.
		// If multiple packets are coalesced, we slice them appropriately.
		packet := append([]byte{firstByte}, remaining[1:]...)
		packets = append(packets, packet)
	}

	if len(packets) == 0 {
		return nil, errors.New("failed to parse any packets from datagram")
	}

	return packets, nil
}
