package wire

import (
	"errors"
	"io"
)

var (
	ErrInvalidVarInt = errors.New("qpack: invalid variable-length integer encoding")
	ErrBufferTooSmall = errors.New("qpack: buffer too small for varint encoding")
)

// ReadVarInt reads a QPACK variable-length integer per RFC 9000 / RFC 9204 section 4.1.1.
func ReadVarInt(r io.Reader, prefixBits byte) (uint64, int, error) {
	buf := [1]byte{}
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, 0, err
	}

	mask := byte((1 << prefixBits) - 1)
	val := uint64(buf[0] & mask)

	if val < uint64(mask) {
		return val, 1, nil
	}

	multiplier := uint64(1)
	nBytes := 1
	for {
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			return 0, nBytes, err
		}
	nBytes++
		
		b := buf[0]
		val += uint64(b&0x7f) * multiplier
		multiplier *= 128

		if (b & 0x80) == 0 {
			break
		}
		if nBytes > 8 {
			return 0, nBytes, ErrInvalidVarInt
		}
	}

	return val, nBytes, nil
}

// AppendVarInt appends a QPACK variable-length integer to dst.
func AppendVarInt(dst []byte, prefixBits byte, val uint64) ([]byte, error) {
	mask := uint64((1 << prefixBits) - 1)
	if val < mask {
		if len(dst) == 0 {
			return nil, ErrBufferTooSmall
		}
		dst[len(dst)-1] |= byte(val)
		return dst, nil
	}

	if len(dst) == 0 {
		return nil, ErrBufferTooSmall
	}
	dst[len(dst)-1] |= byte(mask)
	val -= mask

	for val >= 128 {
		dst = append(dst, byte(val&0x7f)|0x80)
		val >>= 7
	}
	dst = append(dst, byte(val))
	return dst, nil
}
