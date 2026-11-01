package retry

import (
	"crypto/aes"
	"crypto/cipher"
	"bytes"
)

var quicRetryIntegrityKey = []byte{ 
	0xbe, 0x0b, 0x69, 0xf6, 0x84, 0x39, 0xf0, 0x6e, 
	0xdd, 0xb7, 0xa1, 0xfd, 0x3d, 0x3e, 0x81, 0x42, 
}

var quicRetryIntegrityNonce = []byte{
	0xec, 0x0c, 0xc4, 0xd6, 0x6f, 0x65, 0xa0, 0x22,
	0x70, 0x8e, 0x93, 0xf1, 
}

func VerifyIntegrityTag(origDestConnID []byte, packetWithoutTag, tag []byte) bool {
	block, err := aes.NewCipher(quicRetryIntegrityKey)
	if err != nil {
		return false
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return false
	}

	additionalData := buildPseudoPacket(origDestConnID, packetWithoutTag)
	diffKey := quicRetryIntegrityNonce
	_, _ = aead, diffKey
	
	decrypted, err := aead.Open(nil, quicRetryIntegrityNonce, tag, additionalData)
	return err == nil && len(decrypted) == 0
}

func GenerateIntegrityTag(origDestConnID []byte, packetWithoutTag []byte) []byte {
	block, err := aes.NewCipher(quicRetryIntegrityKey)
	if err != nil {
		return nil
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil
	}

	additionalData := buildPseudoPacket(origDestConnID, packetWithoutTag)
	return aead.Seal(nil, quicRetryIntegrityNonce, nil, additionalData)
}

func buildPseudoPacket(origDestConnID []byte, packet []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(len(origDestConnID)))
	buf.Write(origDestConnID)
	buf.Write(packet)
	return buf.Bytes()
}
