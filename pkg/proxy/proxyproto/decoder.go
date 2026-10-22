package proxyproto

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

var (
	proxyV2Signature = [...]byte{0x0d, 0x0a, 0x0d, 0x0a, 0x00, 0x0d, 0x0a, 0x51, 0x55, 0x49, 0x54, 0x0a}
	crc32cTable      = crc32.MakeTable(crc32.Castagnoli)
)

// DecodeHeader validates and returns fixed-size metadata from a v2 header.
func DecodeHeader(frame []byte) (Header, error) {
	if len(frame) < HeaderSize {
		return Header{}, ErrTruncatedFrame
	}
	if !bytes.Equal(frame[:SignatureSize], proxyV2Signature[:]) {
		return Header{}, ErrInvalidSignature
	}
	if frame[12]>>4 != 2 {
		return Header{}, ErrUnsupportedVersion
	}
	command := Command(frame[12] & 0x0f)
	if !command.valid() {
		return Header{}, ErrInvalidCommand
	}
	family := AddressFamily(frame[13] >> 4)
	if !family.valid() {
		return Header{}, ErrInvalidAddressFamily
	}
	transport := TransportProtocol(frame[13] & 0x0f)
	if !transport.valid() {
		return Header{}, ErrInvalidTransportProtocol
	}
	length := binary.BigEndian.Uint16(frame[14:16])
	if length < ChecksumSize {
		return Header{}, ErrInvalidLength
	}
	return Header{Command: command, Family: family, Transport: transport, Length: length}, nil
}

// Decode validates and decodes one complete PROXY protocol v2 frame.
// The input is never modified, and TLVData borrows the input storage.
func Decode(frame []byte) (Frame, error) {
	header, err := DecodeHeader(frame)
	if err != nil {
		return Frame{}, err
	}
	total := HeaderSize + int(header.Length)
	if len(frame) < total {
		return Frame{}, ErrTruncatedFrame
	}
	if len(frame) > total {
		return Frame{}, ErrTrailingBytes
	}
	if err := validateFrameHeader(header); err != nil {
		return Frame{}, err
	}
	addressLength := addressLengthFor(header)
	if int(header.Length) < addressLength+ChecksumSize {
		return Frame{}, ErrInvalidLength
	}
	tlvStart := HeaderSize + addressLength
	checksumOffset := total - ChecksumSize
	if err := validateTLVRegion(frame[tlvStart:checksumOffset]); err != nil {
		return Frame{}, err
	}
	declared := binary.BigEndian.Uint32(frame[checksumOffset:total])
	actual := crc32.Checksum(frame[:checksumOffset], crc32cTable)
	if declared != actual {
		return Frame{}, ErrChecksumMismatch
	}
	result, err := decodeFrameFields(frame, header)
	if err != nil {
		return Frame{}, err
	}
	result.TLVData = frame[tlvStart:checksumOffset]
	result.Checksum = declared
	result.Length = total
	return result, nil
}

// Decoder is a stateless frame decoder.
type Decoder struct{}

// NewDecoder returns a stateless frame decoder.
func NewDecoder() Decoder {
	return Decoder{}
}

// Decode validates and decodes one complete frame.
func (Decoder) Decode(frame []byte) (Frame, error) {
	return Decode(frame)
}

func validateFrameHeader(header Header) error {
	if header.Command == CommandLocal {
		if header.Family != AddressFamilyUnspec || header.Transport != TransportUnspec {
			return ErrInvalidLocalMetadata
		}
	}
	return nil
}

func addressLengthFor(header Header) int {
	if header.Command == CommandLocal {
		return 0
	}
	switch header.Family {
	case AddressFamilyINET:
		return 12
	case AddressFamilyINET6:
		return 36
	case AddressFamilyUNIX:
		return 2 * UnixAddressSize
	default:
		return 0
	}
}

func decodeFrameFields(frame []byte, header Header) (Frame, error) {
	result := Frame{Command: header.Command, Family: header.Family, Transport: header.Transport}
	if header.Command != CommandProxy {
		return result, nil
	}
	switch header.Family {
	case AddressFamilyINET:
		decodeInetFrame(&result, frame)
	case AddressFamilyINET6:
		decodeInet6Frame(&result, frame)
	case AddressFamilyUNIX:
		return decodeUnixFrame(result, frame)
	}
	return result, nil
}

func decodeInetFrame(result *Frame, frame []byte) {
	copy(result.SourceAddress[:4], frame[HeaderSize:HeaderSize+4])
	copy(result.DestinationAddress[:4], frame[HeaderSize+4:HeaderSize+8])
	result.SourceAddressLength = 4
	result.DestinationAddressLength = 4
	result.SourcePort = binary.BigEndian.Uint16(frame[HeaderSize+8 : HeaderSize+10])
	result.DestinationPort = binary.BigEndian.Uint16(frame[HeaderSize+10 : HeaderSize+12])
}

func decodeInet6Frame(result *Frame, frame []byte) {
	copy(result.SourceAddress[:], frame[HeaderSize:HeaderSize+16])
	copy(result.DestinationAddress[:], frame[HeaderSize+16:HeaderSize+32])
	result.SourceAddressLength = 16
	result.DestinationAddressLength = 16
	result.SourcePort = binary.BigEndian.Uint16(frame[HeaderSize+32 : HeaderSize+34])
	result.DestinationPort = binary.BigEndian.Uint16(frame[HeaderSize+34 : HeaderSize+36])
}

func decodeUnixFrame(result Frame, frame []byte) (Frame, error) {
	sourceLength, ok := decodeUnixPath(frame[HeaderSize : HeaderSize+UnixAddressSize])
	if !ok {
		return Frame{}, ErrMalformedUnixPath
	}
	destinationLength, ok := decodeUnixPath(frame[HeaderSize+UnixAddressSize : HeaderSize+2*UnixAddressSize])
	if !ok {
		return Frame{}, ErrMalformedUnixPath
	}
	copy(result.SourceUnixPath[:], frame[HeaderSize:HeaderSize+UnixAddressSize])
	copy(result.DestinationUnixPath[:], frame[HeaderSize+UnixAddressSize:HeaderSize+2*UnixAddressSize])
	result.SourceUnixPathLength = uint8(sourceLength)
	result.DestinationUnixPathLength = uint8(destinationLength)
	return result, nil
}

func decodeUnixPath(path []byte) (int, bool) {
	for index, value := range path {
		if value != 0 {
			continue
		}
		for trailing := index + 1; trailing < len(path); trailing++ {
			if path[trailing] != 0 {
				return 0, false
			}
		}
		return index, true
	}
	return 0, false
}

func validateTLVRegion(region []byte) error {
	iterator := TLVIterator{remaining: region}
	for {
		_, ok := iterator.Next()
		if !ok {
			return iterator.Err()
		}
	}
}

// DecodeState identifies the current state of a StreamDecoder.
type DecodeState uint8

const (
	// DecodeStateHeader indicates that the fixed header is being read.
	DecodeStateHeader DecodeState = iota
	// DecodeStatePayload indicates that the declared payload is being read.
	DecodeStatePayload
	// DecodeStateComplete indicates that a complete frame was decoded.
	DecodeStateComplete
	// DecodeStateFailed indicates a terminal decoding failure.
	DecodeStateFailed
)

// String returns a stable state name.
func (s DecodeState) String() string {
	switch s {
	case DecodeStateHeader:
		return "HEADER"
	case DecodeStatePayload:
		return "PAYLOAD"
	case DecodeStateComplete:
		return "COMPLETE"
	case DecodeStateFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// ContextReader is an optional reader interface that can interrupt a blocked
// read when the supplied context is canceled.
type ContextReader interface {
	ReadContext(context.Context, []byte) (int, error)
}

// StreamDecoder incrementally reads frames into caller-owned storage.
// A canceled read preserves the current offset so a later call can resume.
type StreamDecoder struct {
	state         DecodeState
	header        [HeaderSize]byte
	headerOffset  int
	frame         []byte
	payloadOffset int
	err           error
}

// NewStreamDecoder returns a decoder in its header-reading state.
func NewStreamDecoder() *StreamDecoder {
	return &StreamDecoder{state: DecodeStateHeader}
}

// State returns the decoder's current state.
func (d *StreamDecoder) State() DecodeState {
	return d.state
}

// Reset returns the decoder to its initial state.
func (d *StreamDecoder) Reset() {
	d.state = DecodeStateHeader
	d.headerOffset = 0
	d.payloadOffset = 0
	d.frame = nil
	d.err = nil
}

// Read reads the next frame into dst. dst must have capacity for the largest
// accepted frame. Successful calls do not allocate heap storage.
func (d *StreamDecoder) Read(ctx context.Context, src io.Reader, dst []byte) (Frame, error) {
	if ctx == nil {
		return Frame{}, ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return Frame{}, err
	}
	if d.state == DecodeStateComplete {
		d.Reset()
	} else if d.state == DecodeStateFailed {
		return Frame{}, d.err
	}
	if d.state == DecodeStateHeader {
		if err := d.readHeader(ctx, src); err != nil {
			return Frame{}, err
		}
		if err := d.prepareFrame(dst); err != nil {
			return Frame{}, err
		}
	}
	return d.readPayload(ctx, src)
}

func (d *StreamDecoder) readHeader(ctx context.Context, src io.Reader) error {
	read, err := readFull(ctx, src, d.header[d.headerOffset:])
	d.headerOffset += read
	if err != nil {
		if ctx.Err() == nil {
			d.fail(err)
		}
		return err
	}
	return nil
}

func (d *StreamDecoder) prepareFrame(dst []byte) error {
	header, err := DecodeHeader(d.header[:])
	if err != nil {
		d.fail(err)
		return err
	}
	total := HeaderSize + int(header.Length)
	if len(dst) < total {
		err := ErrBufferTooSmall
		d.fail(err)
		return err
	}
	d.frame = dst[:total]
	copy(d.frame[:HeaderSize], d.header[:])
	d.payloadOffset = HeaderSize
	d.state = DecodeStatePayload
	return nil
}

func (d *StreamDecoder) readPayload(ctx context.Context, src io.Reader) (Frame, error) {
	read, err := readFull(ctx, src, d.frame[d.payloadOffset:])
	d.payloadOffset += read
	if err != nil {
		if ctx.Err() == nil {
			d.fail(err)
		}
		return Frame{}, err
	}
	frame, err := Decode(d.frame)
	if err != nil {
		d.fail(err)
		return Frame{}, err
	}
	d.state = DecodeStateComplete
	return frame, nil
}

func (d *StreamDecoder) fail(err error) {
	d.state = DecodeStateFailed
	d.err = err
}

func readFull(ctx context.Context, src io.Reader, dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	total := 0
	for total < len(dst) {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		var read int
		var err error
		if contextReader, ok := src.(ContextReader); ok {
			read, err = contextReader.ReadContext(ctx, dst[total:])
		} else {
			read, err = src.Read(dst[total:])
		}
		if read < 0 || read > len(dst)-total {
			return total, io.ErrShortBuffer
		}
		total += read
		if contextErr := ctx.Err(); contextErr != nil {
			return total, contextErr
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if total == len(dst) {
					return total, nil
				}
				return total, io.ErrUnexpectedEOF
			}
			return total, err
		}
		if read == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}
