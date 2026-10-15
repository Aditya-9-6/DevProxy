package waf

import (
	"net/url"
	"testing"
)

func TestParameterPollution(t *testing.T) {
	validator := NewParameterValidator()

	t.Run("CleanRequest", func(t *testing.T) {
		vals := url.Values{}
		vals.Add("id", "123")
		vals.Add("name", "test")
		if !validator.ValidateQuery(vals) {
			t.Error("Expected clean request to pass")
		}
	})

	t.Run("PollutedRequest", func(t *testing.T) {
		vals := url.Values{}
		vals.Add("id", "123")
		vals.Add("id", "456")
		if validator.ValidateQuery(vals) {
			t.Error("Expected polluted request to fail")
		}
	})
}

func BenchmarkParameterValidation(b *testing.B) {
	validator := NewParameterValidator()
	vals := url.Values{}
	vals.Add("id", "123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateQuery(vals)
	}
}
