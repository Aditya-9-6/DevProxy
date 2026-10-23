package multipart

import (
	"strings"
)

// MIMEHeader represents parsed MIME part headers.
type MIMEHeader map[string][]string

func ParseHeader(headerBlock string, maxHeaderBytes int) (MIMEHeader, error) {
	if len(headerBlock) > maxHeaderBytes {
		return nil, ErrHeaderTooLarge
	}
	headers := make(MIMEHeader)
	lines := strings.Split(headerBlock, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			headers[key] = append(headers[key], val)
		}
	}
	return headers, nil
}
