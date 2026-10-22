package proxyproto

import (
	"encoding/binary"
	"hash/crc32"
)

// Encoder writes checksummed PROXY protocol v2 frames into caller-owned buffers.
type Encoder struct{}

// NewEncoder returns a stateless frame encoder.
func NewEncoder() Encoder {
	return Encoder{}
}

// Encode writes a frame into dst and returns the number of bytes written.
// dst must be large enough for the complete frame. Successful calls do not
// allocate heap storage.
func (Encoder) Encode(dst []byte, frame Frame) (int, error) {
	if err := validateEncodeFrame(frame); err != nil {
		return 0, err
	}
	addressLength := addressLengthFor(Header{
		Command: frame.Command,
		Family:  frame.Family,
	})
	payloadLength := addressLength + len(frame.TLVData) + ChecksumSize
	if payloadLength > MaxPayloadSize {
		return 0, ErrLengthOverflow
	}
	total := HeaderSize + payloadLength
	if len(dst) < total {
		return 0, ErrBufferTooSmall
	}
	if err := validateTLVRegion(frame.TLVData); err != nil {
		return 0, err
	}
	copy(dst[:SignatureSize], proxyV2Signature[:])
	dst[12] = 0x20 | byte(frame.Command)
	dst[13] = byte(frame.Family)<<4 | byte(frame.Transport)
	binary.BigEndian.PutUint16(dst[14:16], uint16(payloadLength))
	if err := writeFrameEndpoints(dst[HeaderSize:], frame); err != nil {
		return 0, err
	}
	tlvStart := HeaderSize + addressLength
	copy(dst[tlvStart:tlvStart+len(frame.TLVData)], frame.TLVData)
	checksumOffset := total - ChecksumSize
	checksum := crc32.Checksum(dst[:checksumOffset], crc32cTable)
	binary.BigEndian.PutUint32(dst[checksumOffset:total], checksum)
	return total, nil
}

// Encode writes a frame into dst using a stateless encoder.
func Encode(dst []byte, frame Frame) (int, error) {
	return Encoder{}.Encode(dst, frame)
}

func validateEncodeFrame(frame Frame) error {
	if !frame.Command.valid() {
		return ErrInvalidCommand
	}
	if !frame.Family.valid() {
		return ErrInvalidAddressFamily
	}
	if !frame.Transport.valid() {
		return ErrInvalidTransportProtocol
	}
	if frame.Command == CommandLocal {
		if frame.Family != AddressFamilyUnspec || frame.Transport != TransportUnspec {
			return ErrInvalidLocalMetadata
		}
		return nil
	}
	if frame.Family == AddressFamilyUNIX {
		if err := validateUnixPath(frame.SourceUnixPath, frame.SourceUnixPathLength); err != nil {
			return err
		}
		if err := validateUnixPath(frame.DestinationUnixPath, frame.DestinationUnixPathLength); err != nil {
			return err
		}
	}
	return nil
}

func writeFrameEndpoints(dst []byte, frame Frame) error {
	switch frame.Family {
	case AddressFamilyINET:
		writeInetEndpoints(dst, frame)
	case AddressFamilyINET6:
		writeInet6Endpoints(dst, frame)
	case AddressFamilyUNIX:
		return writeUnixEndpoints(dst, frame)
	}
	return nil
}

func writeInetEndpoints(dst []byte, frame Frame) {
	copy(dst[0:4], frame.SourceAddress[:4])
	copy(dst[4:8], frame.DestinationAddress[:4])
	binary.BigEndian.PutUint16(dst[8:10], frame.SourcePort)
	binary.BigEndian.PutUint16(dst[10:12], frame.DestinationPort)
}

func writeInet6Endpoints(dst []byte, frame Frame) {
	copy(dst[0:16], frame.SourceAddress[:])
	copy(dst[16:32], frame.DestinationAddress[:])
	binary.BigEndian.PutUint16(dst[32:34], frame.SourcePort)
	binary.BigEndian.PutUint16(dst[34:36], frame.DestinationPort)
}

func writeUnixEndpoints(dst []byte, frame Frame) error {
	for index := range dst[:2*UnixAddressSize] {
		dst[index] = 0
	}
	copy(dst[:frame.SourceUnixPathLength], frame.SourceUnixPath[:frame.SourceUnixPathLength])
	destinationStart := UnixAddressSize
	copy(dst[destinationStart:destinationStart+int(frame.DestinationUnixPathLength)], frame.DestinationUnixPath[:frame.DestinationUnixPathLength])
	return nil
}

func validateUnixPath(path [UnixAddressSize]byte, length uint8) error {
	if length >= UnixAddressSize {
		return ErrInvalidUnixPath
	}
	for index := 0; index < int(length); index++ {
		if path[index] == 0 {
			return ErrInvalidUnixPath
		}
	}
	return nil
}
