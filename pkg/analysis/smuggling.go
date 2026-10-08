package analysis

import (
	"fmt"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// SmugglingRule detects HTTP Request Smuggling indicators (CL.TE, TE.CL, duplicate headers, and transfer-encoding obfuscation).
type SmugglingRule struct{}

// NewSmugglingRule creates a new HTTP Request Smuggling detection rule instance.
func NewSmugglingRule() *SmugglingRule {
	return &SmugglingRule{}
}

func (r *SmugglingRule) Name() string {
	return "HTTP Request Smuggling (CL.TE / TE.CL) Detector"
}

func (r *SmugglingRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event.ReqHeaders == nil {
		return nil
	}

	var findings []*Finding
	now := time.Now()

	hasContentLength := false
	var clValues []string
	hasTransferEncoding := false
	var teValues []string

	// Check all request headers case-insensitively for Content-Length and Transfer-Encoding
	for k, vv := range event.ReqHeaders {
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

		lowerKey := strings.ToLower(strings.TrimSpace(k))
		if lowerKey == "content-length" {
			hasContentLength = true
			clValues = append(clValues, vv...)
		} else if lowerKey == "transfer-encoding" {
			hasTransferEncoding = true
			teValues = append(teValues, vv...)
		}
	}

	// 1. Dual Content-Length and Transfer-Encoding Header Presence (CL.TE or TE.CL vector)
	if hasContentLength && hasTransferEncoding {
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("smuggle-cl-te-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityCritical,
			Category:    "REQUEST_SMUGGLING",
			RuleName:    r.Name(),
			Title:       "HTTP Request Smuggling: Conflicting Content-Length and Transfer-Encoding",
			Description: "The request contains both Content-Length and Transfer-Encoding headers. Front-end and back-end servers interpret these competing headers differently (CL.TE or TE.CL), allowing attackers to smuggle arbitrary HTTP requests.",
			Evidence:    fmt.Sprintf("Content-Length: %v | Transfer-Encoding: %v", clValues, teValues),
			Location:    "HTTP Request Headers",
			Remediation: "Reject requests containing both Content-Length and Transfer-Encoding headers at the gateway/proxy boundary, or normalize HTTP/1.1 requests to HTTP/2.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// 2. Multiple Content-Length headers or duplicate conflicting values
	if hasContentLength && len(clValues) > 1 {
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("smuggle-multi-cl-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityHigh,
			Category:    "REQUEST_SMUGGLING",
			RuleName:    r.Name(),
			Title:       "HTTP Request Smuggling: Multiple Content-Length Headers",
			Description: fmt.Sprintf("The request includes %d Content-Length headers. Front-end and back-end servers may disagree on which length header takes precedence.", len(clValues)),
			Evidence:    fmt.Sprintf("Content-Length values: %v", clValues),
			Location:    "HTTP Request Headers",
			Remediation: "Ensure API gateways reject requests with multiple Content-Length headers.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// 3. Multiple Transfer-Encoding headers or Obfuscation
	if hasTransferEncoding {
		if len(teValues) > 1 {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("smuggle-multi-te-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityHigh,
				Category:    "REQUEST_SMUGGLING",
				RuleName:    r.Name(),
				Title:       "HTTP Request Smuggling: Multiple Transfer-Encoding Headers",
				Description: fmt.Sprintf("The request includes %d Transfer-Encoding headers, which is often used in TE.TE desynchronization attacks.", len(teValues)),
				Evidence:    fmt.Sprintf("Transfer-Encoding values: %v", teValues),
				Location:    "HTTP Request Headers",
				Remediation: "Reject requests with multiple Transfer-Encoding headers.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}

		for _, teVal := range teValues {
			for _, token := range strings.Split(teVal, ",") {
				lowerTE := strings.ToLower(strings.TrimSpace(token))
				if lowerTE != "chunked" && lowerTE != "compress" && lowerTE != "deflate" && lowerTE != "gzip" && lowerTE != "identity" {
					findings = append(findings, &Finding{
						ID:          fmt.Sprintf("smuggle-obfuscated-te-%s", event.ID),
						RequestID:   event.ID,
						Timestamp:   now,
						Severity:    SeverityHigh,
						Category:    "REQUEST_SMUGGLING",
						RuleName:    r.Name(),
						Title:       "HTTP Request Smuggling: Obfuscated Transfer-Encoding Header",
						Description: fmt.Sprintf("The Transfer-Encoding header value '%s' appears obfuscated or non-standard, designed to bypass front-end proxy validation while being parsed by back-end servers.", teVal),
						Evidence:    fmt.Sprintf("Transfer-Encoding: %s", teVal),
						Location:    "HTTP Request Headers",
						Remediation: "Strictly validate Transfer-Encoding against RFC 9110 (only allow 'chunked', etc.).",
						URL:         event.URL,
						Method:      event.Method,
					})
					break
				}
			}
		}
	}

	return findings
}
