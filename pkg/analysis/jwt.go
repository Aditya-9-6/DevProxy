package analysis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

var (
	jwtRegex      = regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]{10,}\.eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]*\b`)
	sensitiveKeys = []string{
		"password", "passwd", "pwd", "secret", "private_key",
		"ssn", "social_security", "credit_card", "card_number", "cvv", "api_key",
	}
)

// JWTDetails contains decoded structure and security evaluation of a JSON Web Token.
type JWTDetails struct {
	Raw       string                 `json:"raw"`
	Header    map[string]interface{} `json:"header"`
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
	Algorithm string                 `json:"algorithm"`
	Type      string                 `json:"type"`
	Subject   string                 `json:"subject,omitempty"`
	Issuer    string                 `json:"issuer,omitempty"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	IssuedAt  *time.Time             `json:"issued_at,omitempty"`
	IsExpired bool                   `json:"is_expired"`
	Findings  []*Finding             `json:"findings"`
}

// DecodeBase64URL decodes base64url data with or without padding.
func DecodeBase64URL(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(seg)
}

// ParseAndInspectJWT parses, extracts, and runs security lint checks on a JWT token.
func ParseAndInspectJWT(tokenString, requestID, url, method, location string) (*JWTDetails, []*Finding) {
	tokenString = strings.TrimSpace(tokenString)
	if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		tokenString = strings.TrimSpace(tokenString[7:])
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, nil
	}

	headerBytes, err := DecodeBase64URL(parts[0])
	if err != nil {
		return nil, nil
	}

	payloadBytes, err := DecodeBase64URL(parts[1])
	if err != nil {
		return nil, nil
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, nil
	}

	sig := ""
	if len(parts) == 3 {
		sig = parts[2]
	}

	alg, _ := header["alg"].(string)
	typ, _ := header["typ"].(string)
	sub, _ := payload["sub"].(string)
	iss, _ := payload["iss"].(string)

	details := &JWTDetails{
		Raw:       tokenString,
		Header:    header,
		Payload:   payload,
		Signature: sig,
		Algorithm: alg,
		Type:      typ,
		Subject:   sub,
		Issuer:    iss,
		Findings:  make([]*Finding, 0),
	}

	now := time.Now()
	var findings []*Finding

	// Check 1: Algorithm 'none' or missing algorithm
	normalizedAlg := strings.ToLower(strings.TrimSpace(alg))
	if normalizedAlg == "none" || normalizedAlg == "" {
		f := &Finding{
			ID:          fmt.Sprintf("jwt-alg-none-%s", requestID),
			RequestID:   requestID,
			Timestamp:   now,
			Severity:    SeverityCritical,
			Category:    "INSECURE_AUTH",
			RuleName:    "JWT Security Linter",
			Title:       "Insecure JWT Algorithm ('none' or empty)",
			Description: fmt.Sprintf("The JWT specified algorithm '%s'. Algorithm 'none' indicates that signature verification is disabled, allowing arbitrary token tampering and privilege escalation.", alg),
			Evidence:    fmt.Sprintf("Header alg: %q", alg),
			Location:    location,
			Remediation: "Enforce strict server-side signature verification using asymmetric algorithms (RS256, ES256) or strong HMAC (HS256). Reject tokens with alg: none.",
			URL:         url,
			Method:      method,
		}
		findings = append(findings, f)
	}

	// Check 2: Missing or Invalid Signature
	if len(parts) == 2 || (len(parts) == 3 && parts[2] == "") {
		if normalizedAlg != "none" {
			f := &Finding{
				ID:          fmt.Sprintf("jwt-unsigned-%s", requestID),
				RequestID:   requestID,
				Timestamp:   now,
				Severity:    SeverityCritical,
				Category:    "INSECURE_AUTH",
				RuleName:    "JWT Security Linter",
				Title:       "Unsigned JWT Token",
				Description: "The JWT token has no cryptographic signature component.",
				Evidence:    "Token contains only header and payload segments without a signature.",
				Location:    location,
				Remediation: "Ensure tokens are signed with a trusted private key or secret before issuing.",
				URL:         url,
				Method:      method,
			}
			findings = append(findings, f)
		}
	}

	// Check 3: Expiration (exp)
	if expVal, ok := payload["exp"]; ok {
		var expUnix int64
		switch v := expVal.(type) {
		case float64:
			expUnix = int64(v)
		case int64:
			expUnix = v
		case json.Number:
			expUnix, _ = v.Int64()
		}

		if expUnix > 0 {
			expTime := time.Unix(expUnix, 0)
			details.ExpiresAt = &expTime
			if now.After(expTime) {
				details.IsExpired = true
				f := &Finding{
					ID:          fmt.Sprintf("jwt-expired-%s", requestID),
					RequestID:   requestID,
					Timestamp:   now,
					Severity:    SeverityMedium,
					Category:    "EXPIRED_CREDENTIAL",
					RuleName:    "JWT Security Linter",
					Title:       "Expired JWT Token Transmitted",
					Description: fmt.Sprintf("The JWT expired on %s, but is still being transmitted.", expTime.UTC().Format(time.RFC3339)),
					Evidence:    fmt.Sprintf("Current time: %s, Token exp: %s", now.UTC().Format(time.RFC3339), expTime.UTC().Format(time.RFC3339)),
					Location:    location,
					Remediation: "Refresh tokens promptly. Ensure expired tokens are rejected by the API backend.",
					URL:         url,
					Method:      method,
				}
				findings = append(findings, f)
			}
		}
	} else {
		// Missing expiration claim
		f := &Finding{
			ID:          fmt.Sprintf("jwt-no-exp-%s", requestID),
			RequestID:   requestID,
			Timestamp:   now,
			Severity:    SeverityLow,
			Category:    "INSECURE_AUTH",
			RuleName:    "JWT Security Linter",
			Title:       "JWT Missing Expiration ('exp') Claim",
			Description: "The JWT token does not contain an 'exp' claim. Unexpiring tokens pose high persistent risk if intercepted.",
			Evidence:    "Claim 'exp' not found in JWT payload.",
			Location:    location,
			Remediation: "Include a short-lived 'exp' claim (e.g. 15 minutes to 24 hours) along with refresh token rotation.",
			URL:         url,
			Method:      method,
		}
		findings = append(findings, f)
	}

	// Check 4: Excessive Lifetime (> 1 Year)
	if iatVal, ok := payload["iat"]; ok && details.ExpiresAt != nil {
		var iatUnix int64
		switch v := iatVal.(type) {
		case float64:
			iatUnix = int64(v)
		case int64:
			iatUnix = v
		}
		if iatUnix > 0 {
			iatTime := time.Unix(iatUnix, 0)
			details.IssuedAt = &iatTime
			lifespan := details.ExpiresAt.Sub(iatTime)
			if lifespan > 365*24*time.Hour {
				f := &Finding{
					ID:          fmt.Sprintf("jwt-lifespan-%s", requestID),
					RequestID:   requestID,
					Timestamp:   now,
					Severity:    SeverityLow,
					Category:    "INSECURE_AUTH",
					RuleName:    "JWT Security Linter",
					Title:       "Excessive JWT Lifespan (> 1 Year)",
					Description: fmt.Sprintf("The JWT token has a valid duration of %.1f days. Long-lived bearer tokens increase replay risk.", lifespan.Hours()/24),
					Evidence:    fmt.Sprintf("iat: %s, exp: %s", iatTime.UTC().Format(time.RFC3339), details.ExpiresAt.UTC().Format(time.RFC3339)),
					Location:    location,
					Remediation: "Shorten access token validity to minutes or hours, and rely on refresh tokens.",
					URL:         url,
					Method:      method,
				}
				findings = append(findings, f)
			}
		}
	}

	// Check 5: Sensitive plain-text credentials in claims
	for _, key := range sensitiveKeys {
		if val, exists := payload[key]; exists {
			f := &Finding{
				ID:          fmt.Sprintf("jwt-sensitive-%s-%s", requestID, key),
				RequestID:   requestID,
				Timestamp:   now,
				Severity:    SeverityHigh,
				Category:    "INFORMATION_DISCLOSURE",
				RuleName:    "JWT Security Linter",
				Title:       fmt.Sprintf("Sensitive Field '%s' in JWT Payload", key),
				Description: fmt.Sprintf("The JWT payload contains a sensitive key %q (%v). Standard JWT tokens are Base64-encoded, NOT encrypted, and visible to any client or proxy.", key, val),
				Evidence:    fmt.Sprintf("Payload contains sensitive key: %s", key),
				Location:    location,
				Remediation: "Never store credentials, passwords, or PII inside unencrypted JWT claims. Use JWE (encrypted JWT) or opaque reference tokens.",
				URL:         url,
				Method:      method,
			}
			findings = append(findings, f)
		}
	}

	details.Findings = findings
	return details, findings
}

// JWTSecurityRule passively inspects all HTTP requests and responses for JWT tokens.
type JWTSecurityRule struct{}

// NewJWTSecurityRule creates a new rule instance.
func NewJWTSecurityRule() *JWTSecurityRule {
	return &JWTSecurityRule{}
}

func (r *JWTSecurityRule) Name() string {
	return "JWT Security Linter & Passive Inspector"
}

func (r *JWTSecurityRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var allFindings []*Finding

	// 1. Scan Request Headers (Authorization, Cookie, custom headers)
	for name, values := range event.ReqHeaders {
		for _, val := range values {
			if strings.EqualFold(name, "Authorization") && strings.HasPrefix(strings.ToLower(val), "bearer ") {
				token := strings.TrimSpace(val[7:])
				if _, findings := ParseAndInspectJWT(token, event.ID, event.URL, event.Method, "Request Header: Authorization"); len(findings) > 0 {
					allFindings = append(allFindings, findings...)
				}
			} else {
				// Search for JWT pattern
				matches := jwtRegex.FindAllString(val, 2)
				for _, match := range matches {
					if _, findings := ParseAndInspectJWT(match, event.ID, event.URL, event.Method, fmt.Sprintf("Request Header: %s", name)); len(findings) > 0 {
						allFindings = append(allFindings, findings...)
					}
				}
			}
		}
	}

	// 2. Scan Response Headers (Set-Cookie, etc.)
	for name, values := range event.RespHeaders {
		for _, val := range values {
			matches := jwtRegex.FindAllString(val, 2)
			for _, match := range matches {
				if _, findings := ParseAndInspectJWT(match, event.ID, event.URL, event.Method, fmt.Sprintf("Response Header: %s", name)); len(findings) > 0 {
					allFindings = append(allFindings, findings...)
				}
			}
		}
	}

	// 3. Scan Request & Response Body if JSON or containing eyJ
	if len(event.ReqBody) > 0 && strings.Contains(string(event.ReqBody), "eyJ") {
		matches := jwtRegex.FindAllString(string(event.ReqBody), 2)
		for _, match := range matches {
			if _, findings := ParseAndInspectJWT(match, event.ID, event.URL, event.Method, "HTTP Request Body"); len(findings) > 0 {
				allFindings = append(allFindings, findings...)
			}
		}
	}

	if len(event.RespBody) > 0 && strings.Contains(string(event.RespBody), "eyJ") {
		matches := jwtRegex.FindAllString(string(event.RespBody), 2)
		for _, match := range matches {
			if _, findings := ParseAndInspectJWT(match, event.ID, event.URL, event.Method, "HTTP Response Body"); len(findings) > 0 {
				allFindings = append(allFindings, findings...)
			}
		}
	}

	return allFindings
}
