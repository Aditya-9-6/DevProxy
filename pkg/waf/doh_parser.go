package waf

import (
	"encoding/binary"
	"errors"
)

// ParseDNSMessage inspects a DoH wire-format payload for security violations.
func ParseDNSMessage(data []byte) error {
	if len(data) < 12 {
		return errors.New("payload too short")
	}

	qdCount := binary.BigEndian.Uint16(data[4:6])
	if qdCount == 0 {
		return nil
	}

	offset := 12
	for i := 0; i < int(qdCount); i++ {
		newOffset, err := skipName(data, offset)
		if err != nil {
			return err
		}
		offset = newOffset + 4
		if offset > len(data) {
			return errors.New("out of bounds")
		}
		qType := binary.BigEndian.Uint16(data[offset-4 : offset-2])
		GlobalDoHMetrics.RecordRecordType(qType)
	}
	return nil
}

func skipName(data []byte, offset int) (int, error) {
	for offset < len(data) {
		b := data[offset]
		if b == 0 {
			return offset + 1, nil
		}
		if (b & 0xC0) == 0xC0 {
			return offset + 2, nil
		}
		offset += int(b) + 1
	}
	return 0, errors.New("invalid label")
}
