package analysis

import (
	"fmt"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// RequestSmugglingRule inspects HTTP request headers for Content-Length and Transfer-Encoding
// header conflicts, obfuscation, and duplication characteristic of CL.TE or TE.CL smuggling attacks.
type RequestSmugglingRule struct{}

// NewRequestSmugglingRule creates a new RequestSmugglingRule instance.
func NewRequestSmugglingRule() *RequestSmugglingRule {
	return &RequestSmugglingRule{}
}

// Name returns the descriptive name of the security rule.
func (r *RequestSmugglingRule) Name() string {
	return "HTTP Request Smuggling (CL.TE / TE.CL) Desynchronization Detector"
}

// Evaluate analyzes request headers for HTTP request smuggling indicators.
func (r *RequestSmugglingRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event == nil || event.ReqHeaders == nil {
		return nil
	}

	var findings []*Finding
	now := time.Now()

	hasCL := false
	hasTE := false
	clValues := []string{}
	teValues := []string{}
	var clHeaderName string
	var teHeaderName string

	// Check all request headers case-insensitively for Content-Length and Transfer-Encoding
	for name, values := range event.ReqHeaders {
		lowerName := strings.ToLower(strings.TrimSpace(name))

		// Check for obfuscated or spaced header names (e.g., "Transfer-Encoding ", "Content- Length")
		if strings.Contains(name, " ") || strings.Contains(name, "\t") || strings.Contains(name, "\r") || strings.Contains(name, "\n") {
			if lowerName == "content-length" || strings.HasPrefix(lowerName, "content-length") {
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("smuggle-obf-cl-%s", event.ID),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    SeverityCritical,
					Category:    "REQUEST_SMUGGLING",
					RuleName:    r.Name(),
					Title:       "Obfuscated Content-Length Header",
					Description: fmt.Sprintf("Request contains whitespace-obfuscated Content-Length header (%q). Frontend and backend parsers may interpret this differently, leading to HTTP request desynchronization.", name),
					Evidence:    fmt.Sprintf("Header key: %q", name),
					Location:    "HTTP Request Header",
					Remediation: "Reject requests with whitespace or invalid characters in core routing headers.",
					URL:         event.URL,
					Method:      event.Method,
				})
			}
			if lowerName == "transfer-encoding" || strings.HasPrefix(lowerName, "transfer-encoding") {
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("smuggle-obf-te-%s", event.ID),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    SeverityCritical,
					Category:    "REQUEST_SMUGGLING",
					RuleName:    r.Name(),
					Title:       "Obfuscated Transfer-Encoding Header",
					Description: fmt.Sprintf("Request contains whitespace-obfuscated Transfer-Encoding header (%q). This is a hallmark of TE.CL / CL.TE smuggling exploits.", name),
					Evidence:    fmt.Sprintf("Header key: %q", name),
					Location:    "HTTP Request Header",
					Remediation: "Normalize headers and strictly reject requests with malformed header keys.",
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}

		if lowerName == "content-length" {
			hasCL = true
			clHeaderName = name
			clValues = append(clValues, values...)
		} else if lowerName == "transfer-encoding" {
			hasTE = true
			teHeaderName = name
			teValues = append(teValues, values...)
		}
	}

	// Check for duplicate Content-Length headers or conflicting values
	if len(clValues) > 1 {
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("smuggle-dup-cl-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityHigh,
			Category:    "REQUEST_SMUGGLING",
			RuleName:    r.Name(),
			Title:       "Multiple Content-Length Headers",
			Description: fmt.Sprintf("Request contains %d Content-Length headers with values %v. Multiple content lengths introduce ambiguity between proxy and backend server length determination.", len(clValues), clValues),
			Evidence:    fmt.Sprintf("Content-Length values: %v", clValues),
			Location:    fmt.Sprintf("HTTP Request Header: %s", clHeaderName),
			Remediation: "Ensure exact single Content-Length header is enforced at the gateway.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// Check for duplicate Transfer-Encoding headers
	if len(teValues) > 1 {
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("smuggle-dup-te-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityHigh,
			Category:    "REQUEST_SMUGGLING",
			RuleName:    r.Name(),
			Title:       "Multiple Transfer-Encoding Headers",
			Description: fmt.Sprintf("Request contains %d Transfer-Encoding headers. Duplicate transfer encodings can cause desynchronization.", len(teValues)),
			Evidence:    fmt.Sprintf("Transfer-Encoding values: %v", teValues),
			Location:    fmt.Sprintf("HTTP Request Header: %s", teHeaderName),
			Remediation: "Reject requests with multiple Transfer-Encoding headers.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// Check for simultaneous Content-Length AND Transfer-Encoding (CL.TE or TE.CL desynchronization vector)
	if hasCL && hasTE {
		// Check if Transfer-Encoding actually indicates chunked encoding or similar active transfer
		isChunkedActive := false
		for _, teVal := range teValues {
			if strings.Contains(strings.ToLower(teVal), "chunked") {
				isChunkedActive = true
				break
			}
		}

		if isChunkedActive {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("smuggle-clte-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityCritical,
				Category:    "REQUEST_SMUGGLING",
				RuleName:    r.Name(),
				Title:       "HTTP Request Smuggling: Simultaneous Content-Length and Transfer-Encoding (CL.TE / TE.CL)",
				Description: "The request specifies both a Content-Length header and a chunked Transfer-Encoding header. Per RFC 7230 Section 3.3.3, if a message is received with both Transfer-Encoding and Content-Length, the Transfer-Encoding overrides the Content-Length. However, front-end and back-end proxies often disagree on which takes precedence, enabling request smuggling (CL.TE or TE.CL desynchronization).",
				Evidence:    fmt.Sprintf("Content-Length: %v | Transfer-Encoding: %v", clValues, teValues),
				Location:    "HTTP Request Headers",
				Remediation: "Ambiguous request routing. Reject requests containing both Content-Length and Transfer-Encoding headers at the gateway proxy level.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	return findings
}
