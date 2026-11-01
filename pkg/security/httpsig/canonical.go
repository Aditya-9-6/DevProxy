package httpsig

import (
	"fmt"
	"net/http"
	"strings"
)

// BuildCanonicalString constructs the RFC 9421 compliant canonical HTTP message string
// incorporating requested covered components, values, and signature parameters.
func BuildCanonicalString(req *http.Request, components []string, params map[string]string) (string, error) {
	var sb strings.Builder

	for _, comp := range components {
		compLower := strings.ToLower(strings.TrimSpace(comp))
		var val string

		switch compLower {
		case "@method":
			val = req.Method
		case "@target-uri":
			if req.URL != nil {
				val = req.URL.String()
			}
		case "@authority":
			val = req.Host
		case "@path":
			if req.URL != nil {
				val = req.URL.Path
			}
		default:
			if strings.HasPrefix(compLower, "@") {
				return "", fmt.Errorf("unsupported derived component: %s", comp)
			}
			// Standard HTTP header field lookup
			val = req.Header.Get(comp)
		}

		sb.WriteString(fmt.Sprintf("\"%s\": %s\n", compLower, val))
	}

	// Append signature parameters structure as required by RFC 9421
	for k, v := range params {
		sb.WriteString(fmt.Sprintf("\"@%s\": %s\n", strings.ToLower(k), v))
	}

	sb.WriteString("\"@signature-params\": ")
	paramList := make([]string, 0, len(components))
	for _, c := range components {
		paramList = append(paramList, fmt.Sprintf("\"%s\"", strings.ToLower(c)))
	}
	sb.WriteString(fmt.Sprintf("(%s)", strings.Join(paramList, " ")))

	return sb.String(), nil
}
