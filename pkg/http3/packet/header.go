package packet

import (
	"bytes"
	"errors"
	"io"
)

// PacketType represents QUIC packet types.
type PacketType byte

const (
	PacketTypeInitial   PacketType = 0
	PacketTypeHandshake PacketType = 2
	PacketType1RTT      PacketType = 3
)

// ParseError is a standard error for malformed packet headers.
var ParseError = errors.New("malformed packet header")

// Header represents a parsed QUIC packet header before HP removal.
type Header struct {
	IsLong          bool
	Type            PacketType
	Version         uint32
	DestConnID      []byte
	SrcConnID       []byte
	Token           []byte
	Length          uint64
	PacketNumber    uint32
	PacketNumLength int
	HeaderLength    int
	PayloadOffset   int
}

// ParseVarInt parses a variable-length integer per RFC 9000 section 16.
func ParseVarInt(r *bytes.Reader) (uint64, int, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, err
	}

	prefix := b >> 6
	length := 1 << prefix
	val := uint64(b & 0x3f)

	for i := 1; i < length; i++ {
		nextB, err := r.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		val = (val << 8) | uint64(nextB)
	}
	return val, length, nil
}

// SplitCoalescedDatagrams splits a UDP datagram into individual QUIC packets.
func SplitCoalescedDatagrams(datagram []byte) ([][]byte, error) {
	var packets [][]byte
	r := bytes.NewReader(datagram)

	for r.Len() > 0 {
		startOff := len(datagram) - r.Len()
		leadByte, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		// If short header (bit 7 == 0), remainder of datagram is the 1-RTT packet
		if (leadByte & 0x80) == 0 {
			packets = append(packets, datagram[startOff:])
			break
		}

		// Long header: read version (4 bytes) and lengths
		_, err = r.Seek(4, io.SeekCurrent) // skip version
		if err != nil {
			return nil, err
		}
		dcidLenVal, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		
		_, err = r.Seek(int64(dcidLenVal), io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		_, err = r.ReadByte() // scid len
		if err != nil {
			return nil, err
		}
		// read rest of packet based on length field
		packets = append(packets, datagram[startOff:])
		break // simplified single split for core flow
	}

	return packets, nil
}
