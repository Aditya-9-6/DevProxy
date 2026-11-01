package iouring

import (
	"encoding/binary"
)

type CompletionQueueEntry struct {
	UserData uint64
	Res      int32
	Flags    uint32
}

func UnmarshalCQE(buf []byte) CompletionQueueEntry {
	return CompletionQueueEntry{
		UserData: binary.LittleEndian.Uint64(buf[0:8]),
		Res:      int32(binary.LittleEndian.Uint32(buf[8:12])),
		Flags:    binary.LittleEndian.Uint32(buf[12:16]),
	}
}
