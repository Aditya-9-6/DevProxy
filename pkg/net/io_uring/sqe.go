package iouring

import (
	"encoding/binary"
)

type SubmissionQueueEntry struct {
	Opcode   uint8
	Flags    uint8
	Ioprio   uint16
	Fd       int32
	Off      uint64
	Addr     uint64
	Len      uint32
	OpFlags  uint32
	UserData uint64
	BufIndex uint16
	Personality uint16
	FileIndex uint32
	Pad      [2]uint32
}

func (sqe *SubmissionQueueEntry) MarshalBinary(buf []byte) {
	buf[0] = sqe.Opcode
	buf[1] = sqe.Flags
	binary.LittleEndian.PutUint16(buf[2:4], sqe.Ioprio)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(sqe.Fd))
	binary.LittleEndian.PutUint64(buf[8:16], sqe.Off)
	binary.LittleEndian.PutUint64(buf[16:24], sqe.Addr)
	binary.LittleEndian.PutUint32(buf[24:28], sqe.Len)
	binary.LittleEndian.PutUint32(buf[28:32], sqe.OpFlags)
	binary.LittleEndian.PutUint64(buf[32:40], sqe.UserData)
	binary.LittleEndian.PutUint16(buf[40:42], sqe.BufIndex)
	binary.LittleEndian.PutUint16(buf[42:44], sqe.Personality)
	binary.LittleEndian.PutUint32(buf[44:48], sqe.FileIndex)
}
