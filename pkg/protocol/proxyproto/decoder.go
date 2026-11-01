package proxyproto

import (
	"errors"
	"io"
	"net/netip"
	"sync"
)

var (
	ErrInvalidSignature = errors.New("proxyproto: invalid v2 signature")
	ErrInvalidVersion   = errors.New("proxyproto: invalid v2 version")
	ErrUnsupportedCmd   = errors.New(
		"proxyproto: unsupported command or family",
	)
	ErrTruncatedHeader  = errors.New("proxyproto: truncated header")
	ErrBufferTooSmall   = errors.New("proxyproto: buffer too small")
)

var v2Signature = [12]byte{
	0x0D,
	0x0A,
	0x0D,
	0x0A,
	0x00,
	0x0D,
	0x0A,
	0x51,
	0x55,
	0x49,
	0x54,
	0x0A,
}

// Header contains parsed metadata from a PROXY protocol v2 header.
type Header struct {
	Command   byte
	Family    byte
	Protocol  byte
	Source    netip.AddrPort
	Destination netip.AddrPort
}

var headerPool = sync.Pool{
	New: func() any {
		b := make([]byte, 65535)
		return &b
	},
}

// Decoder decodes PROXY protocol v2 headers.
type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode reads and parses a PROXY protocol v2 header from r.
func (d *Decoder) Decode(r io.Reader) (Header, int, error) {
	bufPtr := headerPool.Get().(*[]byte)
	defer headerPool.Put(bufPtr)
	buf := *bufPtr

	if len(buf) < 16 {
		return Header{}, 0, ErrBufferTooSmall
	}

	// Read fixed 16-byte header prefix
	_, err := io.ReadFull(r, buf[:16])
	if err != nil {
		return Header{}, 0, ErrTruncatedHeader
	}

	for i := 0; i < 12; i++ {
		if buf[i] != v2Signature[i] {
			return Header{}, 0, ErrInvalidSignature
	}
	}

	versionAndCmd := buf[12]
	version := versionAndCmd >> 4
	if version != 0x02 {
		return Header{}, 0, ErrInvalidVersion
	}

	command := versionAndCmd & 0x0F
	familyAndProto := buf[13]
	family := familyAndProto >> 4
	proto := familyAndProto & 0x0F

	// 16-bit payload length
	length := uint16(buf[14])<<8 | uint16(buf[15])

	totalHeaderLen := 16 + int(length)
	if len(buf) < totalHeaderLen {
		return Header{}, 0, ErrBufferTooSmall
	}

	if length > 0 {
		_, err = io.ReadFull(r, buf[16:totalHeaderLen])
		if err != nil {
			return Header{}, 0, ErrTruncatedHeader
		}
	}

	var src, dst netip.AddrPort

	switch command {
	case 0x01: // PROXY
		switch family {
		case 0x1: // AF_INET
			if length < 12 {
				return Header{}, 0, ErrTruncatedHeader
			}
			srcIP := netip.AddrFrom4([4]byte{
				buf[16],
				buf[17],
				buf[18],
				buf[19],
			})
			dstIP := netip.AddrFrom4([4]byte{
				buf[20],
				buf[21],
				buf[22],
				buf[23],
			})
			srcPort := uint16(buf[24])<<8 | uint16(buf[25])
			dstPort := uint16(buf[26])<<8 | uint16(buf[27])
			src = netip.AddrPortFrom(srcIP, srcPort)
			dst = netip.AddrPortFrom(dstIP, dstPort)

		case 0x2: // AF_INET6
			if length < 36 {
				return Header{}, 0, ErrTruncatedHeader
			}
			var srcBytes, dstBytes [16]byte
			copy(srcBytes[:], buf[16:32])
			copy(dstBytes[:], buf[32:48])
			srcIP := netip.AddrFrom16(srcBytes)
			dstIP := netip.AddrFrom16(dstBytes)
			srcPort := uint16(buf[48])<<8 | uint16(buf[49])
			dstPort := uint16(buf[50])<<8 | uint16(buf[51])
			src = netip.AddrPortFrom(srcIP, srcPort)
			dst = netip.AddrPortFrom(dstIP, dstPort)
		}
	case 0x00: // LOCAL
		// LOCAL command can have empty or proxy addresses, ignored or passed through
	default:
		return Header{}, 0, ErrUnsupportedCmd
	}

	return Header{
		Command:   command,
		Family:    family,
		Protocol:  proto,
		Source:    src,
		Destination: dst,
	}, totalHeaderLen, nil
}
