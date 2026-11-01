package httpsig

import (
	"errors"
	"strings"
)

var (
	ErrInvalidSignature = errors.New("invalid signature format")
)

// ParseSignatureInput parses an HTTP signature input string.
func ParseSignatureInput(input string) (map[string]string, error) {
	parsed := make(map[string]string)
	parts := strings.Split(input, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			parsed[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
		}
	}
	return parsed, nil
}
