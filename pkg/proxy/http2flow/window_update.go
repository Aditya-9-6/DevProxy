package http2flow

import (
	"encoding/binary"
)

// CreateWindowUpdateFrame generates a raw HTTP/2 WINDOW_UPDATE frame (type 0x8).
func CreateWindowUpdateFrame(streamID uint32, increment uint32) []byte {
	buf := make([]byte, 13)

	// Length: 4 bytes (always 4 for WINDOW_UPDATE)
	buf[0] = 0
	buf[1] = 0
	buf[2] = 4

	// Type: WINDOW_UPDATE (0x8)
	buf[3] = 0x8

	// Flags: 0
	buf[4] = 0

	// Stream Identifier: 4 bytes
	binary.BigEndian.PutUint32(buf[5:9], streamID & 0x7FFFFFFF)

	// Increment: 4 bytes (Reserved bit + 31-bit unsigned integer)
	binary.BigEndian.PutUint32(buf[9:13], increment & 0x7FFFFFFF)

	return buf
}
