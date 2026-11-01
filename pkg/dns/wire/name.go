package wire

import (
	"bytes"
	"errors"
	"strings"
)

// ReadName reads a domain name from a wire-format message buffer.
func ReadName(buf []byte, offset int) (string, int, error) {
	var labels []string
	originalOffset := offset
	jumped := false
	jumps := 0

	for {
		if offset >= len(buf) {
			return "", 0, errors.New("dns wire: name offset out of bounds")
		}

		length := int(buf[offset])
		if length == 0 {
			offset++
			break
		}

		// Check for pointer (compression)
		if (length & 0xC0) == 0xC0 {
			if offset+1 >= len(buf) {
				return "", 0, errors.New("dns wire: truncated compression pointer")
			}
			pointer := int(buf[offset]&0x3F)<<8 | int(buf[offset+1])
			if !jumped {
				originalOffset = offset + 2
				jumped = true
			}
			offset = pointer
			jumps++
			if jumps > 100 {
				return "", 0, errors.New("dns wire: compression pointer loop detected")
			}
			continue
		}

		// Standard label
		offset++
		if offset+length > len(buf) {
			return "", 0, errors.New("dns wire: label length exceeds buffer")
		}

		labels = append(labels, string(buf[offset:offset+length]))
		offset += length
	}

	if !jumped {
		originalOffset = offset
	}

	return strings.Join(labels, "."), originalOffset, nil
}

// WriteName writes a domain name in wire format to a buffer, with basic compression support if map provided.
func WriteName(name string, buf *bytes.Buffer) error {
	if name == "" || name == "." {
		buf.WriteByte(0)
		return nil
	}

	parts := strings.Split(name, ".")
	for _, part := range parts {
		if part == "" {
			continue
		}
		length := len(part)
		if length > 63 {
			return errors.New("dns wire: label too long")
		}
		buf.WriteByte(byte(length))
		buf.WriteString(part)
	}
	buf.WriteByte(0)
	return nil
}
