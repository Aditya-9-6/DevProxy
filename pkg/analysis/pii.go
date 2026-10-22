package analysis

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

var (
	// Matches potential credit card candidate strings (13-19 digits with optional spaces or dashes)
	cardCandidateRegex = regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`)
	// US Social Security Number candidate pattern (AAA-GG-SSSS)
	ssnRegex = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
)

// PIIRule passively scans for plaintext financial credentials and sensitive identity numbers.
type PIIRule struct{}

// NewPIIRule creates a new PII rule instance.
func NewPIIRule() *PIIRule {
	return &PIIRule{}
}

func (r *PIIRule) Name() string {
	return "PII & Financial Data Leak Auditor"
}

func (r *PIIRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	now := time.Now()

	// 1. Scan Response Body for Credit Card Leaks (PCI-DSS violation)
	if len(event.RespBody) > 0 {
		candidates := cardCandidateRegex.FindAllString(string(event.RespBody), 5)
		for _, cand := range candidates {
			clean := sanitizeCardDigits(cand)
			if len(clean) >= 13 && len(clean) <= 19 && IsValidLuhn(clean) {
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("pii-card-resp-%s", event.ID),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    SeverityCritical,
					Category:    "PCI_COMPLIANCE",
					RuleName:    r.Name(),
					Title:       "Valid Credit Card Number Leaked in Response",
					Description: "Detected an unmasked, mathematically valid credit card number (passing Luhn mod-10 verification) in the HTTP response body. Transmitting unmasked card numbers violates PCI-DSS requirements.",
					Evidence:    fmt.Sprintf("Card ending in ...%s (Masked: %s)", clean[len(clean)-4:], maskCard(clean)),
					Location:    "HTTP Response Body",
					Remediation: "Never return full primary account numbers (PAN) in API responses. Truncate to the last 4 digits (e.g. ************1234) or tokenize.",
					URL:         event.URL,
					Method:      event.Method,
				})
				break // One finding per response is sufficient
			}
		}

		// Check SSN
		if ssnMatches := ssnRegex.FindAllString(string(event.RespBody), 5); len(ssnMatches) > 0 {
			for _, cand := range ssnMatches {
				if isValidSSN(cand) {
					findings = append(findings, &Finding{
						ID:          fmt.Sprintf("pii-ssn-resp-%s", event.ID),
						RequestID:   event.ID,
						Timestamp:   now,
						Severity:    SeverityCritical,
						Category:    "PRIVACY_VIOLATION",
						RuleName:    r.Name(),
						Title:       "Social Security Number (SSN) Leaked in Response",
						Description: "Discovered an unmasked US Social Security Number format in the response payload. Transmitting SSNs in plaintext poses identity theft risks and violates GDPR/CCPA privacy standards.",
						Evidence:    fmt.Sprintf("SSN format: ***-**-%s", cand[len(cand)-4:]),
						Location:    "HTTP Response Body",
						Remediation: "Redact or mask Social Security Numbers in backend responses, or encrypt before transmission.",
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

func sanitizeCardDigits(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// IsValidLuhn implements the Luhn formula (mod 10 checksum) used by credit card issuers.
func IsValidLuhn(number string) bool {
	if len(number) < 13 || len(number) > 19 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}

		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

func maskCard(clean string) string {
	if len(clean) < 4 {
		return "****"
	}
	return strings.Repeat("*", len(clean)-4) + clean[len(clean)-4:]
}

func isValidSSN(s string) bool {
	// Format: AAA-GG-SSSS (length 11)
	if len(s) != 11 {
		return false
	}
	area := s[0:3]
	group := s[4:6]
	serial := s[7:11]

	// Area number cannot be 000, 666, or 900-999
	if area == "000" || area == "666" || area[0] == '9' {
		return false
	}
	// Group number cannot be 00
	if group == "00" {
		return false
	}
	// Serial number cannot be 0000
	if serial == "0000" {
		return false
	}
	return true
}
