package analysis

import (
	"fmt"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// SmugglingRule inspects HTTP requests for Content-Length and Transfer-Encoding desynchronization headers (CL.TE, TE.CL, multiple Content-Length headers, and obfuscated transfer encodings).
type SmugglingRule struct{}

// NewSmugglingRule creates a new HTTP Request Smuggling detector rule.
func NewSmugglingRule() *SmugglingRule {
	return &SmugglingRule{}
}

func (r *SmugglingRule) Name() string {
	return "HTTP Request Smuggling (CL.TE / TE.CL) Detector"
}

// httpCanonicalHeaderKey normalizes a header key to its canonical form (e.g. content-length -> Content-Length).
func httpCanonicalHeaderKey(key string) string {
	parts := strings.Split(strings.TrimSpace(key), "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "-")
}

func (r *SmugglingRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event == nil || event.ReqHeaders == nil {
		return nil
	}

	var findings []*Finding
	now := time.Now()

	// 1. Check for both Content-Length and Transfer-Encoding headers present
	hasCL := false
	hasTE := false
	var clValues []string
	var teValues []string

	for k, vals := range event.ReqHeaders {
		// Check for whitespace obfuscation in header names
		if strings.Contains(k, " ") || strings.Contains(k, "\t") || strings.Contains(k, "\r") || strings.Contains(k, "\n") {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("smuggle-header-space-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityCritical,
				Category:    "REQUEST_SMUGGLING",
				RuleName:    r.Name(),
				Title:       "HTTP Header Name Whitespace Obfuscation",
				Description: fmt.Sprintf("Request header name contains whitespace (%q), which can be exploited for request smuggling via parser desynchronization.", k),
				Evidence:    fmt.Sprintf("Header key: %q", k),
				Location:    "HTTP Request Headers",
				Remediation: "Reject requests with whitespace in header names.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}

		canonicalKey := httpCanonicalHeaderKey(k)
		if canonicalKey == "Content-Length" {
			hasCL = true
			clValues = append(clValues, vals...)
		} else if canonicalKey == "Transfer-Encoding" {
			hasTE = true
			teValues = append(teValues, vals...)
		}
	}

	// A. Multiple Content-Length headers with conflicting values
	if len(clValues) > 1 {
		firstVal := clValues[0]
		conflicting := false
		for _, val := range clValues[1:] {
			if val != firstVal {
				conflicting = true
				break
			}
		}
		if conflicting {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("smuggle-multi-cl-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityCritical,
				Category:    "REQUEST_SMUGGLING",
				RuleName:    r.Name(),
				Title:       "Multiple Conflicting Content-Length Headers",
				Description: "Request contains multiple Content-Length headers with conflicting values. Front-end and back-end proxies may interpret message boundaries differently, causing request smuggling.",
				Evidence:    fmt.Sprintf("Content-Length values: %v", clValues),
				Location:    "HTTP Request Headers",
				Remediation: "Reject requests with multiple Content-Length headers at the gateway.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	// B. CL.TE / TE.CL Desynchronization (Both Content-Length and Transfer-Encoding present)
	if hasCL && hasTE {
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("smuggle-cl-te-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityCritical,
			Category:    "REQUEST_SMUGGLING",
			RuleName:    r.Name(),
			Title:       "HTTP Request Smuggling: Content-Length and Transfer-Encoding Both Present",
			Description: "The request specifies both Content-Length and Transfer-Encoding headers. According to RFC 7230, Transfer-Encoding overrides Content-Length, but discrepancies lead to CL.TE or TE.CL smuggling attacks.",
			Evidence:    fmt.Sprintf("Content-Length: %v | Transfer-Encoding: %v", clValues, teValues),
			Location:    "HTTP Request Headers",
			Remediation: "Reject any request containing both Content-Length and Transfer-Encoding headers.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// C. Transfer-Encoding Obfuscation & Malformed Transfer-Encoding
	if hasTE {
		for _, teVal := range teValues {
			if strings.HasSuffix(teVal, "\t") || strings.HasSuffix(teVal, " ") || strings.Contains(teVal, "\r") || strings.Contains(teVal, "\n") {
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("smuggle-te-obf-%s", event.ID),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    SeverityCritical,
					Category:    "REQUEST_SMUGGLING",
					RuleName:    r.Name(),
					Title:       "Transfer-Encoding Header Obfuscation",
					Description: fmt.Sprintf("Transfer-Encoding header contains whitespace or obfuscation characters (%q), facilitating request desynchronization.", teVal),
					Evidence:    fmt.Sprintf("Transfer-Encoding: %q", teVal),
					Location:    "HTTP Request Headers",
					Remediation: "Strictly validate and normalize Transfer-Encoding header values.",
					URL:         event.URL,
					Method:      event.Method,
				})
				break
			}
		}
	}

	return findings
}
