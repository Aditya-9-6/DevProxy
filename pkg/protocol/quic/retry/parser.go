package retry

import (
	"bytes"
	"errors"
	"io"
)

var (
	ErrInvalidPacket = errors.New("quic retry: invalid packet")
	ErrShortBuffer   = errors.New("quic retry: short buffer")
)

type RetryPacket struct {
	HeaderForm    bool
	FixedBit      bool
	LongPacketType byte
	Version       uint32
	DestConnID    []byte
	SrcConnID     []byte
	RetryToken    []byte
	RetryIntegrityTag []byte
}

func ParseRetryPacket(data []byte) (*RetryPacket, error) {
	if len(data) < 17 {
		return nil, ErrShortBuffer
	}
	if (data[0] & 0x80) == 0 {
		return nil, ErrInvalidPacket
	}

	p := &RetryPacket{
		HeaderForm:     true,
		FixedBit:       (data[0] & 0x40) != 0,
		LongPacketType: (data[0] & 0x30) >> 4,
	}

	r := bytes.NewReader(data[1:])
	var version uint32
	if err := readUint32(r, &version); err != nil {
		return nil, err
	}
	p.Version = version

	dcidLenByte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	dcidLen := int(dcidLenByte)
	if r.Len() < dcidLen {
		return nil, ErrShortBuffer
	}
	p.DestConnID = make([]byte, dcidLen)
	if _, err := io.ReadFull(r, p.DestConnID); err != nil {
		return nil, err
	}

	scidLenByte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	scidLen := int(scidLenByte)
	if r.Len() < scidLen + 16 {
		return nil, ErrShortBuffer
	}
	p.SrcConnID = make([]byte, scidLen)
	if _, err := io.ReadFull(r, p.SrcConnID); err != nil {
		return nil, err
	}

	rem := r.Len()
	if rem < 16 {
		return nil, ErrShortBuffer
	}
	tokenLen := rem - 16
	p.RetryToken = make([]byte, tokenLen)
	if _, err := io.ReadFull(r, p.RetryToken); err != nil {
		return nil, err
	}

	p.RetryIntegrityTag = make([]byte, 16)
	if _, err := io.ReadFull(r, p.RetryIntegrityTag); err != nil {
		return nil, err
	}

	return p, nil
}

func readUint32(r *bytes.Reader, v *uint32) error {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	*v = uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	return nil
}
