package analysis

import (
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func makeJWT(headerJSON, payloadJSON, signature string) string {
	h := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	p := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	if signature != "" {
		return h + "." + p + "." + signature
	}
	return h + "." + p + "."
}

func TestJWTAlgNone(t *testing.T) {
	header := `{"alg": "none", "typ": "JWT"}`
	payload := `{"sub": "user123", "name": "Aditya"}`
	token := makeJWT(header, payload, "")

	details, findings := ParseAndInspectJWT(token, "req-1", "https://api.test/profile", "GET", "Authorization")
	if details == nil {
		t.Fatal("expected details to be parsed")
	}

	foundCrit := false
	for _, f := range findings {
		if f.Severity == SeverityCritical && f.Title == "Insecure JWT Algorithm ('none' or empty)" {
			foundCrit = true
		}
	}
	if !foundCrit {
		t.Fatalf("expected critical finding for alg none, got %+v", findings)
	}
}

func TestJWTExpiredAndSensitiveClaim(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour).Unix()
	header := `{"alg": "HS256", "typ": "JWT"}`
	payload := `{"sub": "user456", "exp": ` + string(rune(past)) + `, "password": "supersecretpassword"}`
	payload = `{"sub": "user456", "exp": 1000000000, "password": "supersecretpassword"}`
	token := makeJWT(header, payload, "fakesig1234567890")

	details, findings := ParseAndInspectJWT(token, "req-2", "https://api.test/auth", "POST", "Authorization")
	if details == nil {
		t.Fatal("expected details")
	}
	if !details.IsExpired {
		t.Fatal("expected token to be marked expired")
	}

	foundSensitive := false
	foundExpired := false
	for _, f := range findings {
		if f.Title == "Sensitive Field 'password' in JWT Payload" {
			foundSensitive = true
		}
		if f.Title == "Expired JWT Token Transmitted" {
			foundExpired = true
		}
	}

	if !foundSensitive {
		t.Error("expected finding for sensitive field 'password'")
	}
	if !foundExpired {
		t.Error("expected finding for expired JWT token")
	}
}

func TestJWTRuleEvaluation(t *testing.T) {
	rule := NewJWTSecurityRule()

	badToken := makeJWT(`{"alg": "none"}`, `{"sub": "admin", "admin": true}`, "")

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+badToken)

	event := &ringbuffer.TrafficEvent{
		ID:         "evt-jwt-test",
		ReqHeaders: headers,
		URL:        "https://api.service.internal/admin/dashboard",
		Method:     "GET",
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatal("expected findings from JWTSecurityRule")
	}

	hasAlgNone := false
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			hasAlgNone = true
		}
	}
	if !hasAlgNone {
		t.Fatal("expected critical severity finding for alg none")
	}
}
