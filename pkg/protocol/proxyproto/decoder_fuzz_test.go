package proxyproto

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func FuzzDecoder(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte(Signature))
	f.Add(fuzzHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)))
	f.Add(fuzzHeader(byte(CommandProxy), byte(FamilyINET6), byte(ProtocolStream), make([]byte, INET6PayloadLength)))
	f.Add(fuzzHeader(byte(CommandProxy), byte(FamilyUNIX), byte(ProtocolStream), make([]byte, UNIXPayloadLength)))
	f.Add(fuzzHeader(byte(CommandProxy), byte(FamilyUnspecified), byte(ProtocolUnspecified), nil))
	f.Add(fuzzHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength+1)))
	f.Add(fuzzHeader(byte(CommandProxy), byte(FamilyUnspecified), byte(ProtocolStream), nil))

	f.Fuzz(func(t *testing.T, data []byte) {
		reader := bytes.NewReader(data)
		header, err := (V2Decoder{}).Decode(reader)
		if err != nil {
			return
		}

		consumed := HeaderSize + int(header.PayloadLength)
		if consumed > len(data) {
			t.Fatalf("decoder consumed %d bytes from %d-byte input", consumed, len(data))
		}
		if reader.Len() != len(data)-consumed {
			t.Fatalf("remaining bytes = %d, want %d", reader.Len(), len(data)-consumed)
		}
	})
}

func fuzzHeader(command, family, protocol byte, payload []byte) []byte {
	frame := make([]byte, HeaderSize+len(payload))
	copy(frame[:12], Signature)
	frame[12] = 0x20 | (command & 0x0f)
	frame[13] = (family << 4) | (protocol & 0x0f)
	binary.BigEndian.PutUint16(frame[14:16], uint16(len(payload)))
	copy(frame[HeaderSize:], payload)
	return frame
}
