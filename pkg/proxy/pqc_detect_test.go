package proxy

import (
	"encoding/binary"
	"testing"
)

func TestDetectPQCHybridHandshake(t *testing.T) {
	// Construct a mock supported_groups extension
	// Type 10, Length 6, ListLen 4, Groups: 0x0017, 0x11ec
	ext := make([]byte, 10)
	binary.BigEndian.PutUint16(ext[0:2], 10)
	binary.BigEndian.PutUint16(ext[2:4], 6)
	binary.BigEndian.PutUint16(ext[4:6], 4)
	binary.BigEndian.PutUint16(ext[6:8], 0x0017)
	binary.BigEndian.PutUint16(ext[8:10], GroupX25519MLKEM768)

	if !DetectPQCHybridHandshake(ext) {
		t.Error("Expected PQC detection for X25519MLKEM768")
	}

	// Test negative case
	ext2 := make([]byte, 8)
	binary.BigEndian.PutUint16(ext2[0:2], 10)
	binary.BigEndian.PutUint16(ext2[2:4], 4)
	binary.BigEndian.PutUint16(ext2[4:6], 2)
	binary.BigEndian.PutUint16(ext2[6:8], 0x0017)

	if DetectPQCHybridHandshake(ext2) {
		t.Error("Unexpected PQC detection for standard group")
	}
}
