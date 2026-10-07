package proxy

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// TLS Record Layer and Handshake constants.
const (
	recordTypeHandshake      = 0x16
	handshakeTypeClientHello = 0x01
	maxClientHelloSize       = 65535 // Bounded size to eliminate memory exhaustion attacks
)

// ClientHelloData captures parsed TLS ClientHello components for fingerprinting.
type ClientHelloData struct {
	Version         uint16
	CipherSuites    []uint16
	Extensions      []uint16
	SupportedGroups []uint16
	ECPointFormats  []uint8
	ALPNProtocols   []string
	SNI             string
	RawDataSize     int
}

var chPool = sync.Pool{
	New: func() interface{} {
		return &ClientHelloData{
			CipherSuites:    make([]uint16, 0, 64),
			Extensions:      make([]uint16, 0, 32),
			SupportedGroups: make([]uint16, 0, 16),
			ECPointFormats:  make([]uint8, 0, 4),
			ALPNProtocols:   make([]string, 0, 4),
		}
	},
}

// IsGREASE determines whether a 16-bit TLS value is a GREASE value per RFC 8701.
func IsGREASE(val uint16) bool {
	return (val & 0x0f0f) == 0x0a0a
}

// ParseClientHello parses a raw TLS Record containing a ClientHello message.
// Enforces strict bounds checking and returns an error on malformed/truncated data.
func ParseClientHello(raw []byte) (*ClientHelloData, error) {
	if len(raw) < 5 {
		return nil, fmt.Errorf("packet too short for TLS record header: %d bytes", len(raw))
	}
	if len(raw) > maxClientHelloSize {
		return nil, fmt.Errorf("TLS record exceeds maximum permissible size (%d > %d)", len(raw), maxClientHelloSize)
	}
	if raw[0] != recordTypeHandshake {
		return nil, fmt.Errorf("not a TLS handshake record (type 0x%02x)", raw[0])
	}

	recordLen := binary.BigEndian.Uint16(raw[3:5])
	if int(recordLen) > len(raw)-5 {
		return nil, fmt.Errorf("truncated TLS record payload")
	}

	payload := raw[5 : 5+recordLen]
	if len(payload) < 4 || payload[0] != handshakeTypeClientHello {
		return nil, fmt.Errorf("not a TLS ClientHello message")
	}

	msgLen := int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	if msgLen > len(payload)-4 {
		return nil, fmt.Errorf("truncated ClientHello body")
	}

	body := payload[4 : 4+msgLen]
	return parseClientHelloBody(body)
}

func parseClientHelloBody(body []byte) (*ClientHelloData, error) {
	if len(body) < 34 { // 2 bytes version + 32 bytes random
		return nil, fmt.Errorf("body too short for TLS version and random")
	}

	data := chPool.Get().(*ClientHelloData)
	data.CipherSuites = data.CipherSuites[:0]
	data.Extensions = data.Extensions[:0]
	data.SupportedGroups = data.SupportedGroups[:0]
	data.ECPointFormats = data.ECPointFormats[:0]
	data.ALPNProtocols = data.ALPNProtocols[:0]
	data.SNI = ""
	data.RawDataSize = len(body)
	data.Version = binary.BigEndian.Uint16(body[0:2])

	offset := 34
	if offset >= len(body) {
		return data, nil
	}

	// Session ID
	sessionIDLen := int(body[offset])
	offset += 1 + sessionIDLen
	if offset+2 > len(body) {
		return data, nil
	}

	// Cipher Suites
	cipherLen := int(binary.BigEndian.Uint16(body[offset : offset+2]))
	offset += 2
	if offset+cipherLen > len(body) {
		return data, nil
	}
	for i := 0; i < cipherLen; i += 2 {
		cs := binary.BigEndian.Uint16(body[offset+i : offset+i+2])
		if !IsGREASE(cs) {
			data.CipherSuites = append(data.CipherSuites, cs)
		}
	}
	offset += cipherLen
	if offset >= len(body) {
		return data, nil
	}

	// Compression methods
	compLen := int(body[offset])
	offset += 1 + compLen
	if offset+2 > len(body) {
		return data, nil
	}

	// Extensions
	extTotalLen := int(binary.BigEndian.Uint16(body[offset : offset+2]))
	offset += 2
	if offset+extTotalLen > len(body) {
		return data, nil
	}

	parseExtensions(body[offset:offset+extTotalLen], data)
	return data, nil
}

func parseExtensions(extBytes []byte, data *ClientHelloData) {
	idx := 0
	for idx+4 <= len(extBytes) {
		extType := binary.BigEndian.Uint16(extBytes[idx : idx+2])
		extLen := int(binary.BigEndian.Uint16(extBytes[idx+2 : idx+4]))
		idx += 4

		if idx+extLen > len(extBytes) {
			break
		}

		if !IsGREASE(extType) {
			data.Extensions = append(data.Extensions, extType)
		}

		extVal := extBytes[idx : idx+extLen]
		parseExtensionPayload(extType, extVal, data)
		idx += extLen
	}
}

func parseExtensionPayload(extType uint16, val []byte, data *ClientHelloData) {
	switch extType {
	case 0x0000: // Server Name Indication (SNI)
		if len(val) >= 5 {
			sniLen := int(binary.BigEndian.Uint16(val[3:5]))
			if 5+sniLen <= len(val) {
				data.SNI = string(val[5 : 5+sniLen])
			}
		}
	case 0x000a: // Supported Groups (Elliptic Curves)
		if len(val) >= 2 {
			listLen := int(binary.BigEndian.Uint16(val[0:2]))
			for j := 2; j < 2+listLen && j+2 <= len(val); j += 2 {
				grp := binary.BigEndian.Uint16(val[j : j+2])
				if !IsGREASE(grp) {
					data.SupportedGroups = append(data.SupportedGroups, grp)
				}
			}
		}
	case 0x000b: // EC Point Formats
		if len(val) >= 1 {
			pLen := int(val[0])
			for j := 1; j < 1+pLen && j < len(val); j++ {
				data.ECPointFormats = append(data.ECPointFormats, val[j])
			}
		}
	case 0x0010: // Application Layer Protocol Negotiation (ALPN)
		if len(val) >= 2 {
			alpnLen := int(binary.BigEndian.Uint16(val[0:2]))
			pIdx := 2
			for pIdx < 2+alpnLen && pIdx < len(val) {
				protoLen := int(val[pIdx])
				pIdx++
				if pIdx+protoLen <= len(val) {
					data.ALPNProtocols = append(data.ALPNProtocols, string(val[pIdx:pIdx+protoLen]))
					pIdx += protoLen
				}
			}
		}
	}
}

// ComputeJA3 computes standard JA3 fingerprint and MD5 digest string.
func ComputeJA3(data *ClientHelloData) (string, string) {
	if data == nil {
		return "", ""
	}

	var sb strings.Builder
	sb.WriteString(strconv.Itoa(int(data.Version)))
	sb.WriteByte(',')

	joinUint16(&sb, data.CipherSuites, '-')
	sb.WriteByte(',')

	joinUint16(&sb, data.Extensions, '-')
	sb.WriteByte(',')

	joinUint16(&sb, data.SupportedGroups, '-')
	sb.WriteByte(',')

	joinUint8(&sb, data.ECPointFormats, '-')

	rawStr := sb.String()
	hasher := md5.New()
	hasher.Write([]byte(rawStr))
	md5Digest := hex.EncodeToString(hasher.Sum(nil))

	return rawStr, md5Digest
}

// ComputeJA4 generates an RFC-aligned JA4 client fingerprint string.
func ComputeJA4(data *ClientHelloData, isTCP bool) string {
	if data == nil {
		return ""
	}

	protoPrefix := "t"
	if !isTCP {
		protoPrefix = "q"
	}

	versionStr := "13"
	if data.Version == 0x0303 {
		versionStr = "12"
	} else if data.Version == 0x0302 {
		versionStr = "11"
	} else if data.Version == 0x0301 {
		versionStr = "10"
	}

	hasSNI := "d"
	if data.SNI != "" {
		hasSNI = "d"
	} else {
		hasSNI = "i"
	}

	numCiphers := len(data.CipherSuites)
	numExtensions := len(data.Extensions)

	alpnPart := "00"
	if len(data.ALPNProtocols) > 0 {
		firstProto := data.ALPNProtocols[0]
		if len(firstProto) >= 2 {
			alpnPart = firstProto[0:2]
		}
	}

	partA := fmt.Sprintf("%s%s%s%02d%02d%s", protoPrefix, versionStr, hasSNI, numCiphers, numExtensions, alpnPart)

	// Part B: Sorted 12-char hex truncated SHA-256 of ciphers
	sortedCiphers := make([]uint16, len(data.CipherSuites))
	copy(sortedCiphers, data.CipherSuites)
	sort.Slice(sortedCiphers, func(i, j int) bool { return sortedCiphers[i] < sortedCiphers[j] })

	var cipherHex strings.Builder
	for _, c := range sortedCiphers {
		cipherHex.WriteString(fmt.Sprintf("%04x,", c))
	}
	partB := hexShortHash(cipherHex.String(), 12)

	// Part C: Sorted 12-char hex truncated SHA-256 of extensions
	sortedExts := make([]uint16, len(data.Extensions))
	copy(sortedExts, data.Extensions)
	sort.Slice(sortedExts, func(i, j int) bool { return sortedExts[i] < sortedExts[j] })

	var extHex strings.Builder
	for _, e := range sortedExts {
		extHex.WriteString(fmt.Sprintf("%04x,", e))
	}
	partC := hexShortHash(extHex.String(), 12)

	return fmt.Sprintf("%s_%s_%s", partA, partB, partC)
}

func hexShortHash(s string, length int) string {
	h := md5.Sum([]byte(s))
	full := hex.EncodeToString(h[:])
	if len(full) > length {
		return full[:length]
	}
	return full
}

// ClassifyClient assigns a high-confidence bot or browser label based on JA3 hash and UA.
func ClassifyClient(ja3Hash string, userAgent string) string {
	uaLower := strings.ToLower(userAgent)

	// Known standard automated scripts and HTTP tooling
	switch {
	case strings.Contains(uaLower, "curl/"):
		return "bot/curl"
	case strings.Contains(uaLower, "python-requests"), strings.Contains(uaLower, "aiohttp"):
		return "bot/python-requests"
	case strings.Contains(uaLower, "go-http-client"):
		return "bot/golang"
	case strings.Contains(uaLower, "postman"):
		return "tool/postman"
	case strings.Contains(uaLower, "wget/"):
		return "bot/wget"
	}

	// Browser family heuristic matching
	switch {
	case strings.Contains(uaLower, "edg/"):
		return "browser/edge"
	case strings.Contains(uaLower, "chrome/") && !strings.Contains(uaLower, "edg/"):
		return "browser/chrome"
	case strings.Contains(uaLower, "firefox/"):
		return "browser/firefox"
	case strings.Contains(uaLower, "safari/") && !strings.Contains(uaLower, "chrome/"):
		return "browser/safari"
	}

	if ja3Hash != "" {
		return "client/custom-tls"
	}
	return "client/generic"
}

func joinUint16(sb *strings.Builder, items []uint16, delim byte) {
	for i, val := range items {
		if i > 0 {
			sb.WriteByte(delim)
		}
		sb.WriteString(strconv.Itoa(int(val)))
	}
}

func joinUint8(sb *strings.Builder, items []uint8, delim byte) {
	for i, val := range items {
		if i > 0 {
			sb.WriteByte(delim)
		}
		sb.WriteString(strconv.Itoa(int(val)))
	}
}
