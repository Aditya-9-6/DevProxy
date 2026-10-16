package protocol

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	ProxyV2Signature = "\x0D\x0A\x0D\x0A\x00\x0D\x0A\x51\x55\x49\x54\x0A"
	ProxyV2MinLen    = 16
)

var (
	ErrInvalidSignature = errors.New("invalid proxy protocol v2 signature")
	ErrBufferTooSmall   = errors.New("buffer too small for proxy protocol v2 header")
)

type ProxyV2Header struct {
	Command uint8
	Family  uint8
	Len     uint16
	SrcAddr net.IP
	DstAddr net.IP
	SrcPort uint16
	DstPort uint16
}

func ParseProxyV2(buf []byte) (*ProxyV2Header, int, error) {
	if len(buf) < ProxyV2MinLen {
		return nil, 0, ErrBufferTooSmall
	}
	if string(buf[:12]) != ProxyV2Signature {
		return nil, 0, ErrInvalidSignature
	}

	verCmd := buf[12]
	fam := buf[13]
	dataLen := binary.BigEndian.Uint16(buf[14:16])
	totalLen := int(ProxyV2MinLen + dataLen)

	if len(buf) < totalLen {
		return nil, 0, ErrBufferTooSmall
	}

	h := &ProxyV2Header{Command: verCmd, Family: fam, Len: dataLen}
	addrData := buf[ProxyV2MinLen : ProxyV2MinLen+dataLen]

	if fam == 0x11 { // IPv4
		h.SrcAddr = net.IP(addrData[0:4])
		h.DstAddr = net.IP(addrData[4:8])
		h.SrcPort = binary.BigEndian.Uint16(addrData[8:10])
		h.DstPort = binary.BigEndian.Uint16(addrData[10:12])
	} else if fam == 0x21 { // IPv6
		h.SrcAddr = net.IP(addrData[0:16])
		h.DstAddr = net.IP(addrData[16:32])
		h.SrcPort = binary.BigEndian.Uint16(addrData[32:34])
		h.DstPort = binary.BigEndian.Uint16(addrData[34:36])
	}

	return h, totalLen, nil
}
