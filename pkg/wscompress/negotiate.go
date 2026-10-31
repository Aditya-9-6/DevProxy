package wscompress

import (
	"strconv"
	"strings"
)

// ExtensionParams holds parsed permessage-deflate extension parameters.
type ExtensionParams struct {
	ServerNoContextTakeover bool
	ClientNoContextTakeover bool
	ServerMaxWindowBits     int
	ClientMaxWindowBits     int
}

// ParseOffer parses the Sec-WebSocket-Extensions header value for permessage-deflate.
func ParseOffer(headerVal string) (bool, ExtensionParams, error) {
	parts := strings.Split(headerVal, ";")
	found := false
	params := ExtensionParams{
		ServerMaxWindowBits: 15,
		ClientMaxWindowBits: 15,
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "permessage-deflate" {
			found = true
			continue
		}

		subParts := strings.SplitN(p, "=", 2)
		key := strings.TrimSpace(subParts[0])
		val := ""
		if len(subParts) > 1 {
			val = strings.TrimSpace(subParts[1])
		}

		switch key {
		case "server_no_context_takeover":
			params.ServerNoContextTakeover = true
		case "client_no_context_takeover":
			params.ClientNoContextTakeover = true
		case "server_max_window_bits":
			if val != "" {
				n, err := strconv.Atoi(val)
				if err != nil || n < 8 || n > 15 {
					return false, params, ErrInvalidParameter
				}
				params.ServerMaxWindowBits = n
			}
		case "client_max_window_bits":
			if val != "" {
				n, err := strconv.Atoi(val)
				if err != nil || n < 8 || n > 15 {
					return false, params, ErrInvalidParameter
				}
				params.ClientMaxWindowBits = n
			}
		}
	}

	return found, params, nil
}

// FormatResponse serializes accepted parameters back into a response extension header.
func FormatResponse(params ExtensionParams) string {
	var sb strings.Builder
	sb.WriteString("permessage-deflate")
	if params.ServerNoContextTakeover {
		sb.WriteString("; server_no_context_takeover")
	}
	if params.ClientNoContextTakeover {
		sb.WriteString("; client_no_context_takeover")
	}
	if params.ServerMaxWindowBits != 15 {
		sb.WriteString("; server_max_window_bits=")
		sb.WriteString(strconv.Itoa(params.ServerMaxWindowBits))
	}
	if params.ClientMaxWindowBits != 15 {
		sb.WriteString("; client_max_window_bits=")
		sb.WriteString(strconv.Itoa(params.ClientMaxWindowBits))
	}
	return sb.String()
}
