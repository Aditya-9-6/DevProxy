package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// ResourceRecord represents a standard DNS resource record.
type ResourceRecord struct {
	Name     string
	Type     uint16
	Class    uint16
	TTL      uint32
	RData    []byte
}

// Serialize encodes a ResourceRecord into wire format.
func (rr *ResourceRecord) Serialize(buf *bytes.Buffer) error {
	if err := WriteName(rr.Name, buf); err != nil {
		return err
	}
	b := make([]byte, 10)
	binary.BigEndian.PutUint16(b[0:2], rr.Type)
	binary.BigEndian.PutUint16(b[2:4], rr.Class)
	binary.BigEndian.PutUint32(b[4:8], rr.TTL)
	binary.BigEndian.PutUint16(b[8:10], uint16(len(rr.RData)))
	buf.Write(b)
	buf.Write(rr.RData)
	return nil
}

// ParseResourceRecord parses a ResourceRecord from a wire-format message buffer.
func ParseResourceRecord(buf []byte, offset int) (*ResourceRecord, int, error) {
	name, newOffset, err := ReadName(buf, offset)
	if err != nil {
		return nil, 0, err
	}

	if newOffset+10 > len(buf) {
		return nil, 0, errors.New("dns wire: truncated resource record header")
	}

	typeVal := binary.BigEndian.Uint16(buf[newOffset : newOffset+2])
	classVal := binary.BigEndian.Uint16(buf[newOffset+2 : newOffset+4])
	ttlVal := binary.BigEndian.Uint32(buf[newOffset+4 : newOffset+8])
	rdataLen := binary.BigEndian.Uint16(buf[newOffset+8 : newOffset+10])
	newOffset += 10

	if newOffset+int(rdataLen) > len(buf) {
		return nil, 0, errors.New("dns wire: truncated rdata")
	}

	rdata := make([]byte, rdataLen)
	copy(rdata, buf[newOffset:newOffset+int(rdataLen)])
	newOffset += int(rdataLen)

	return &ResourceRecord{
		Name:  name,
		Type:  typeVal,
		Class: classVal,
		TTL:   ttlVal,
		RData: rdata,
	}, newOffset, nil
}
