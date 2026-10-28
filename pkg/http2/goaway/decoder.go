package goaway

import (
	"io"
)

// Decoder reads and decodes GOAWAY frames from an HTTP/2 connection stream.
type Decoder struct {
	reader io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: r}
}

func (d *Decoder) Decode() (*Frame, error) {
	// Reads frame header (9 bytes)
	header := make([]byte, 9)
	if _, err := io.ReadFull(d.reader, header);
	err != nil {
		return nil, err
	}

	length := uint32(header[0])<<16 | uint32(header[1])<<8 | uint32(header[2])
	frameType := header[3]
	
	if frameType != 0x7 { // 0x7 is GOAWAY
		// Skip non-GOAWAY frames or handle accordingly
		drop := make([]byte, length)
		if _, err := io.ReadFull(d.reader, drop); err != nil {
			return nil, err
		}
		return d.Decode()
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(d.reader, payload); err != nil {
		return nil, err
	}

	return ParseFrame(payload)
}
