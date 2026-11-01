package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// JA4 fingerprint specification constants and structures.
// A JA4 fingerprint is a 36-character canonical string structured as:
// [protocol][version][sni][cipher_count][extension_count][alpn]_[hash_ciphers]_[hash_extensions]

var ja4BufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

// ClientHelloRecord holds parsed TLS ClientHello metadata needed for JA4 generation.
type ClientHelloRecord struct {
	Version     uint16
	CipherSuites []uint16
	Extensions   []uint16
	ServerName   string
	ALPN         string
}

// ServerHelloRecord holds parsed TLS ServerHello metadata needed for JA4S generation.
type ServerHelloRecord struct {
	Version      uint16
	CipherSuite  uint16
	Extensions   []uint16
	ALPN         string
}

// GenerateJA4 computes the JA4 TLS client fingerprint.
func GenerateJA4(ch *ClientHelloRecord) string {
	// 1. Protocol part: 't' for TCP, 'q' for QUIC (default 't')
	proto := 't'

	// 2. TLS Version abbreviation
	versionStr := tlsVersionToString(ch.Version)

	// 3. SNI status: 'd' if SNI is present, 'i' otherwise
	sni := 'i'
	if ch.ServerName != "" {
		sni = 'd'
	}

	// 4. Cipher count (2 digits, zero-padded)
	cipherCount := len(ch.CipherSuites)
	cipherCountStr := strconv.Itoa(cipherCount)
	if cipherCount < 10 {
		cipherCountStr = "0" + cipherCountStr
	}
	if cipherCount > 99 {
		cipherCountStr = "99"
	}

	// 5. Extension count (2 digits, zero-padded)
	extCount := len(ch.Extensions)
	extCountStr := strconv.Itoa(extCount)
	if extCount < 10 {
		extCountStr = "0" + extCountStr
	}
	if extCount > 99 {
		extCountStr = "99"
	}

	// 6. ALPN first and last character
	alpnStr := "00"
	if len(ch.ALPN) >= 2 {
		alpnStr = string([]byte{ch.ALPN[0], ch.ALPN[len(ch.ALPN)-1]})
	} else if len(ch.ALPN) == 1 {
		alpnStr = string([]byte{ch.ALPN[0], '0'})
	}

	part1 := string([]byte{byte(proto)}) + versionStr + string([]byte{byte(sni)}) + cipherCountStr + extCountStr + alpnStr

	// Hash ciphers (sorted, excluding GREASE)
	hashCiphers := hashSortedCiphers(ch.CipherSuites)
	// Hash extensions (sorted, excluding SNI, ALPN, and GREASE)
	hashExts := hashSortedExtensions(ch.Extensions)

	return part1 + "_" + hashCiphers + "_" + hashExts
}

// GenerateJA4S computes the JA4S TLS server fingerprint.
func GenerateJA4S(sh *ServerHelloRecord) string {
	// Protocol part
	proto := 't'
	versionStr := tlsVersionToString(sh.Version)

	// Extension count
	extCount := len(sh.Extensions)
	extCountStr := strconv.Itoa(extCount)
	if extCount < 10 {
		extCountStr = "0" + extCountStr
	}
	if extCount > 99 {
		extCountStr = "99"
	}

	// ALPN
	alpnStr := "00"
	if len(sh.ALPN) >= 2 {
		alpnStr = string([]byte{sh.ALPN[0], sh.ALPN[len(sh.ALPN)-1]})
	} else if len(sh.ALPN) == 1 {
		alpnStr = string([]byte{sh.ALPN[0], '0'})
	}

	part1 := string([]byte{byte(proto)}) + versionStr + extCountStr + alpnStr

	// Cipher suite single hex representation or hash
	cipherHex := strconv.FormatUint(uint64(sh.CipherSuite), 16)
	if len(cipherHex) < 4 {
		cipherHex = strings.Repeat("0", 4-len(cipherHex)) + cipherHex
	}

	hashExts := hashSortedExtensions(sh.Extensions)

	return part1 + "_" + cipherHex + "_" + hashExts
}

func tlsVersionToString(v uint16) string {
	switch v {
	case 0x0304:
		return "13"
	case 0x0303:
		return "12"
	case 0x0302:
		return "11"
	case 0x0301:
		return "10"
	default:
		return "00"
	}
}

func isGrease(v uint16) bool {
	// GREASE values take the form of 0x?a?a where the nibbles match
	return (v & 0x0f0f) == 0x0a0a && (v&0xff) == ((v >> 8) & 0xff)
}

func hashSortedCiphers(ciphers []uint16) string {
	bufPtr := ja4BufferPool.Get().(*[]byte)
	defer ja4BufferPool.Put(bufPtr)
	buf := (*bufPtr)[:0]

	var filtered []uint16
	for _, c := range ciphers {
		if !isGrease(c) {
			filtered = append(filtered, c)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i] < filtered[j]
	});

	for i, c := range filtered {
		if i > 0 {
			buf = append(buf, ',')
		}
		hx := strconv.FormatUint(uint64(c), 16)
		if len(hx) < 4 {
			for k := 0; k < 4-len(hx); k++ {
				buf = append(buf, '0')
			}
		}
		buf = append(buf, hx...)
	}

	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])[:12]
}

func hashSortedExtensions(exts []uint16) string {
	bufPtr := ja4BufferPool.Get().(*[]byte)
	defer ja4BufferPool.Put(bufPtr)
	buf := (*bufPtr)[:0]

	var filtered []uint16
	for _, e := range exts {
		// Exclude SNI (0x0000), ALPN (0x0010), and GREASE
		if e == 0x0000 || e == 0x0010 || isGrease(e) {
			continue
		}
		filtered = append(filtered, e)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i] < filtered[j]
	});

	for i, e := range filtered {
		if i > 0 {
			buf = append(buf, ',')
		}
		hx := strconv.FormatUint(uint64(e), 16)
		if len(hx) < 4 {
			for k := 0; k < 4-len(hx); k++ {
				buf = append(buf, '0')
			}
		}
		buf = append(buf, hx...)
	}

	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])[:12]
}
