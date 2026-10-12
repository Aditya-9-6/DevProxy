package proxy

import (
	"encoding/binary"
)

// PQC Hybrid Group IDs (IANA registered/draft)
const (
	GroupX25519MLKEM768    uint16 = 0x11ec
	GroupSecP256r1MLKEM768 uint16 = 0x11eb
)

// DetectPQCHybridHandshake scans TLS 1.3 ClientHello extensions for PQC hybrid groups.
// It returns true if a known PQC hybrid group is present in the supported_groups extension.
func DetectPQCHybridHandshake(extensions []byte) bool {
	for len(extensions) >= 4 {
		extType := binary.BigEndian.Uint16(extensions[0:2])
		extLen := int(binary.BigEndian.Uint16(extensions[2:4]))
		extData := extensions[4 : 4+extLen]

		if extType == 10 && len(extData) >= 2 {
			// supported_groups extension (type 10)
			listLen := int(binary.BigEndian.Uint16(extData[0:2]))
			groups := extData[2:]
			for i := 0; i+2 <= listLen && i+2 <= len(groups); i += 2 {
				group := binary.BigEndian.Uint16(groups[i : i+2])
				if group == GroupX25519MLKEM768 || group == GroupSecP256r1MLKEM768 {
					return true
				}
			}
		}
		extensions = extensions[4+extLen:]
	}
	return false
}
