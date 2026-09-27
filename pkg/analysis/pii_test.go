package analysis

import (
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestIsValidLuhn(t *testing.T) {
	// Standard test cards (valid Luhn, 16 digits)
	validCard := "4532015112830366"
	invalidCard := "4532015112830367"

	if !IsValidLuhn(validCard) {
		t.Errorf("expected %s to be valid Luhn", validCard)
	}
	if IsValidLuhn(invalidCard) {
		t.Errorf("expected %s to be invalid Luhn", invalidCard)
	}

	// Boundary lengths
	if IsValidLuhn("12345") {
		t.Errorf("expected too short number to be invalid")
	}
	if IsValidLuhn("123456789012345678901") {
		t.Errorf("expected too long number to be invalid")
	}
}

func TestPIIRule_CreditCardLeak(t *testing.T) {
	rule := NewPIIRule()

	// Response leaking a valid Luhn card: 4532-0151-1283-0366
	respBody := `{"status": "ok", "user": {"payment_method": "4532-0151-1283-0366"}}`

	event := &ringbuffer.TrafficEvent{
		ID:          "test-pii-1",
		Timestamp:   time.Now(),
		Method:      "GET",
		URL:         "https://api.example.com/account",
		StatusCode:  200,
		RespHeaders: http.Header{"Content-Type": []string{"application/json"}},
		RespBody:    []byte(respBody),
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected finding for leaked credit card, got none")
	}

	foundCard := false
	for _, f := range findings {
		if f.Category == "PCI_COMPLIANCE" && f.Severity == SeverityCritical {
			foundCard = true
			if f.RuleName != rule.Name() {
				t.Errorf("unexpected rule name: %s", f.RuleName)
			}
		}
	}
	if !foundCard {
		t.Errorf("expected PCI_COMPLIANCE finding for credit card leak")
	}
}

func TestPIIRule_InvalidCardNotReported(t *testing.T) {
	rule := NewPIIRule()

	// Numbers that look like cards but fail Luhn
	respBody := `{"order_id": "1234-5678-1234-5678", "transaction_ref": "9876543210987654"}`

	event := &ringbuffer.TrafficEvent{
		ID:          "test-pii-2",
		Timestamp:   time.Now(),
		Method:      "GET",
		URL:         "https://api.example.com/order",
		StatusCode:  200,
		RespHeaders: http.Header{"Content-Type": []string{"application/json"}},
		RespBody:    []byte(respBody),
	}

	findings := rule.Evaluate(event)
	for _, f := range findings {
		if f.Category == "PCI_COMPLIANCE" {
			t.Errorf("expected no PCI_COMPLIANCE findings for invalid Luhn numbers, got %+v", f)
		}
	}
}

func TestPIIRule_SSNLeak(t *testing.T) {
	rule := NewPIIRule()

	// SSN leak
	respBody := `{"user": {"ssn": "123-45-6789", "name": "John Doe"}}`

	event := &ringbuffer.TrafficEvent{
		ID:          "test-pii-3",
		Timestamp:   time.Now(),
		Method:      "GET",
		URL:         "https://api.example.com/profile",
		StatusCode:  200,
		RespHeaders: http.Header{"Content-Type": []string{"application/json"}},
		RespBody:    []byte(respBody),
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected SSN leak finding, got none")
	}

	foundSSN := false
	for _, f := range findings {
		if f.Category == "PRIVACY_VIOLATION" && f.Severity == SeverityCritical {
			foundSSN = true
		}
	}
	if !foundSSN {
		t.Errorf("expected PRIVACY_VIOLATION finding for SSN leak")
	}
}

func TestPIIRule_NoFindingsOnCleanResponse(t *testing.T) {
	rule := NewPIIRule()

	respBody := `{"items": [{"id": 1, "name": "Widget"}, {"id": 2, "name": "Gadget"}]}`

	event := &ringbuffer.TrafficEvent{
		ID:          "test-pii-4",
		Timestamp:   time.Now(),
		Method:      "GET",
		URL:         "https://api.example.com/items",
		StatusCode:  200,
		RespHeaders: http.Header{"Content-Type": []string{"application/json"}},
		RespBody:    []byte(respBody),
	}

	findings := rule.Evaluate(event)
	if len(findings) != 0 {
		t.Errorf("expected no findings on clean payload, got %d", len(findings))
	}
}
