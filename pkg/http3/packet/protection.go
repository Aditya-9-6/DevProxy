package packet

import (
	"crypto/aes"
	"errors"
)

// RemoveHeaderProtection unmasks the packet number and first byte using AES-ECB or ChaCha20.
func RemoveHeaderProtection(hpKey []byte, sample []byte, firstByte *byte, packetNumberBytes []byte) error {
	if len(hpKey) < 16 || len(sample) < 16 {
		return errors.New("invalid key or sample length for header protection")
	}

	block, err := aes.NewCipher(hpKey[:16])
	if err != nil {
		return err
	}

	mask := make([]byte, 16)
	block.Encrypt(mask, sample[:16])

	// Unmask first byte
	isLongHeader := (*firstByte & 0x80) != 0
	if isLongHeader {
		*firstByte ^= mask[0] & 0x0f
	} else {
		*firstByte ^= mask[0] & 0x1f
	}

	// Unmask packet number bytes
	pnLen := (int(*firstByte) & 0x03) + 1
	if len(packetNumberBytes) < pnLen {
		return errors.New("packet number buffer too short")
	}

	for i := 0; i < pnLen; i++ {
		packetNumberBytes[i] ^= mask[1+i]
	}

	return nil
}
