package proxy

import (
	"encoding/binary"
	"io"
)

// Deframer parses gRPC length-prefixed messages from a stream.
type Deframer struct {
	reader io.Reader
}

func NewDeframer(r io.Reader) *Deframer {
	return &Deframer{reader: r}
}

// Next reads the next gRPC message from the stream.
func (d *Deframer) Next() (*GRPCMessage, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(d.reader, header); err != nil {
		return nil, err
	}

	compressed := header[0] == 1
	length := binary.BigEndian.Uint32(header[1:])

	data := make([]byte, length)
	if _, err := io.ReadFull(d.reader, data); err != nil {
		return nil, err
	}

	return &GRPCMessage{
		Compressed: compressed,
		Data:       data,
	}, nil
}
