package spiffe

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

const backslashByte = 0x5c

var (
	ErrInvalidURI                  = errors.New("invalid SPIFFE URI")
	ErrInvalidScheme               = errors.New("invalid SPIFFE URI scheme")
	ErrInvalidURIForm              = errors.New("invalid SPIFFE URI form")
	ErrInvalidUserInfo             = errors.New("SPIFFE URI must not contain user information")
	ErrInvalidPort                 = errors.New("SPIFFE URI must not contain a port")
	ErrInvalidQuery                = errors.New("SPIFFE URI must not contain a query")
	ErrInvalidFragment             = errors.New("SPIFFE URI must not contain a fragment")
	ErrInvalidTrustDomain          = errors.New("invalid SPIFFE trust domain")
	ErrInvalidPath                 = errors.New("invalid SPIFFE path")
	ErrDotSegment                  = errors.New("SPIFFE path must not contain dot segments")
	ErrEncodedPathSeparator        = errors.New("SPIFFE path must not contain encoded separators")
	ErrNonCanonicalPercentEncoding = errors.New("non-canonical percent encoding in SPIFFE path")
)

// ID identifies a workload within a SPIFFE trust domain.
type ID struct {
	TrustDomain string
	Path        string
}

// Parse parses a canonical SPIFFE URI.
func Parse(raw string) (ID, error) {
	if raw == "" {
		return ID{}, ErrInvalidURI
	}

	scheme, ok := rawScheme(raw)
	if !ok || scheme != "spiffe" {
		return ID{}, ErrInvalidScheme
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		if strings.Contains(err.Error(), "invalid port") {
			return ID{}, ErrInvalidPort
		}
		return ID{}, ErrInvalidURI
	}

	// url.URL cannot distinguish an absent fragment from an empty fragment.
	if strings.ContainsRune(raw, '#') {
		return ID{}, ErrInvalidFragment
	}

	return ParseURL(parsed)
}

// ParseURL validates an already parsed SPIFFE URL. The valid path does not
// allocate, making it suitable for hot proxy request-processing paths.
func ParseURL(uri *url.URL) (ID, error) {
	if uri == nil {
		return ID{}, ErrInvalidURI
	}
	if uri.Scheme != "spiffe" {
		return ID{}, ErrInvalidScheme
	}
	if uri.Opaque != "" {
		return ID{}, ErrInvalidURIForm
	}
	if uri.User != nil {
		return ID{}, ErrInvalidUserInfo
	}
	if hasInvalidPort(uri.Host) {
		return ID{}, ErrInvalidPort
	}
	if uri.ForceQuery || uri.RawQuery != "" {
		return ID{}, ErrInvalidQuery
	}
	if uri.Fragment != "" || uri.RawFragment != "" {
		return ID{}, ErrInvalidFragment
	}
	if err := ValidateTrustDomain(uri.Host); err != nil {
		return ID{}, err
	}

	path := uri.Path
	if path == "" {
		path = "/"
	}
	if err := validateDecodedPath(path); err != nil {
		return ID{}, err
	}
	if uri.RawPath != "" {
		if err := validateRawPath(uri.RawPath, path); err != nil {
			return ID{}, err
		}
	}

	return ID{TrustDomain: uri.Host, Path: path}, nil
}

// Validate validates the trust domain and path of an ID.
func (id ID) Validate() error {
	if err := ValidateTrustDomain(id.TrustDomain); err != nil {
		return err
	}
	return validateDecodedPath(id.Path)
}

// String returns the canonical SPIFFE URI representation.
func (id ID) String() string {
	return "spiffe://" + id.TrustDomain + id.Path
}

func rawScheme(raw string) (string, bool) {
	index := strings.IndexByte(raw, ':')
	if index <= 0 {
		return "", false
	}
	return raw[:index], true
}

func hasInvalidPort(host string) bool {
	if host == "" {
		return false
	}
	if strings.HasPrefix(host, "[") {
		return strings.Contains(host, "]:")
	}
	return strings.ContainsRune(host, ':')
}

func validateDecodedPath(path string) error {
	if path == "" || path[0] != '/' {
		return ErrInvalidPath
	}
	if !utf8.ValidString(path) {
		return ErrInvalidPath
	}
	if strings.IndexByte(path, backslashByte) >= 0 {
		return ErrInvalidPath
	}
	if hasDotSegment(path) {
		return ErrDotSegment
	}
	for index := 0; index < len(path); index++ {
		if path[index] < 0x20 || path[index] == 0x7f {
			return ErrInvalidPath
		}
	}
	return nil
}

func hasDotSegment(path string) bool {
	start := 0
	for index := 0; index <= len(path); index++ {
		if index != len(path) && path[index] != '/' {
			continue
		}
		segment := path[start:index]
		if segment == "." || segment == ".." {
			return true
		}
		start = index + 1
	}
	return false
}

func validateRawPath(raw, decoded string) error {
	if raw == "" {
		return nil
	}
	if raw[0] != '/' {
		return ErrInvalidPath
	}

	decodedIndex := 0
	for rawIndex := 0; rawIndex < len(raw); {
		value, consumed, err := rawPathByte(raw, rawIndex)
		if err != nil {
			return err
		}
		if decodedIndex >= len(decoded) || decoded[decodedIndex] != value {
			return ErrInvalidPath
		}
		decodedIndex++
		rawIndex += consumed
	}
	if decodedIndex != len(decoded) {
		return ErrInvalidPath
	}
	return nil
}

func rawPathByte(raw string, index int) (byte, int, error) {
	character := raw[index]
	if character != '%' {
		if character == backslashByte || character == '?' || character == '#' {
			return 0, 0, ErrInvalidPath
		}
		if character < 0x20 || character == 0x7f {
			return 0, 0, ErrInvalidPath
		}
		if character >= 0x80 {
			return 0, 0, ErrNonCanonicalPercentEncoding
		}
		return character, 1, nil
	}

	if index+2 >= len(raw) {
		return 0, 0, ErrInvalidURI
	}
	high, highOK, highLower := hexValue(raw[index+1])
	low, lowOK, lowLower := hexValue(raw[index+2])
	if !highOK || !lowOK {
		return 0, 0, ErrInvalidURI
	}
	if highLower || lowLower {
		return 0, 0, ErrNonCanonicalPercentEncoding
	}

	value := high<<4 | low
	if value == '/' {
		return 0, 0, ErrEncodedPathSeparator
	}
	if value == backslashByte {
		return 0, 0, ErrInvalidPath
	}
	if isUnreservedByte(value) {
		return 0, 0, ErrNonCanonicalPercentEncoding
	}
	if value < 0x20 || value == 0x7f {
		return 0, 0, ErrInvalidPath
	}
	return value, 3, nil
}

func hexValue(character byte) (byte, bool, bool) {
	switch {
	case character >= '0' && character <= '9':
		return character - '0', true, false
	case character >= 'A' && character <= 'F':
		return character - 'A' + 10, true, false
	case character >= 'a' && character <= 'f':
		return character - 'a' + 10, true, true
	default:
		return 0, false, false
	}
}

func isUnreservedByte(value byte) bool {
	return value >= 'A' && value <= 'Z' ||
		value >= 'a' && value <= 'z' ||
		value >= '0' && value <= '9' ||
		value == '-' || value == '.' || value == '_' || value == '~'
}

// ValidatePath validates a decoded or percent-encoded SPIFFE path.
func ValidatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	decoded, err := url.PathUnescape(path)
	if err != nil {
		return ErrInvalidPath
	}
	if err := validateDecodedPath(decoded); err != nil {
		return err
	}
	if path != decoded {
		if err := validateRawPath(path, decoded); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTrustDomain validates a lowercase DNS-style SPIFFE trust domain.
func ValidateTrustDomain(domain string) error {
	if domain == "" || len(domain) > 255 {
		return ErrInvalidTrustDomain
	}

	start := 0
	for index := 0; index <= len(domain); index++ {
		if index != len(domain) && domain[index] != '.' {
			continue
		}
		if err := validateTrustDomainLabel(domain[start:index]); err != nil {
			return err
		}
		start = index + 1
	}
	return nil
}

func validateTrustDomainLabel(label string) error {
	if label == "" || len(label) > 63 {
		return ErrInvalidTrustDomain
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return ErrInvalidTrustDomain
	}
	for index := 0; index < len(label); index++ {
		character := label[index]
		if character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' || character == '-' {
			continue
		}
		return ErrInvalidTrustDomain
	}
	return nil
}
