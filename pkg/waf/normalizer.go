package waf

import (
	"bytes"
	"net/url"
	"sort"
	"strings"
)

// NormalizeContentDisposition strips malicious characters from headers.
func NormalizeContentDisposition(header []byte) []byte {
	// Remove potential CRLF injection sequences
	return bytes.ReplaceAll(bytes.ReplaceAll(header, []byte("\r"), nil), []byte("\n"), nil)
}

// NormalizeQueryString sorts and normalizes URL parameters to prevent cache poisoning and bypass attacks.
func NormalizeQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		vals := params[k]
		sort.Strings(vals)
		b.WriteString(strings.Join(vals, ","))
	}
	return b.String()
}
