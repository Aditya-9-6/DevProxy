package waf

import (
	"sync"
	"testing"
)

func TestParameterPollution(t *testing.T) {
	validator := NewParameterValidator()

	t.Run("CleanRequest", func(t *testing.T) {
		if !validator.ValidateQuery("id=123&name=test") {
			t.Error("Expected clean request to pass")
		}
	})

	t.Run("PollutedRequest", func(t *testing.T) {
		if validator.ValidateQuery("id=123&id=456") {
			t.Error("Expected polluted request to fail")
		}
	})

	t.Run("CaseSensitivity", func(t *testing.T) {
		if validator.ValidateQuery("ID=123&id=456") {
			t.Error("Expected case-insensitive pollution to fail")
		}
	})
}

func TestRecordParamUsageConcurrency(t *testing.T) {
	validator := NewParameterValidator()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			validator.RecordParamUsage("test")
		}()
	}
	wg.Wait()
}

func BenchmarkParameterValidation(b *testing.B) {
	validator := NewParameterValidator()
	query := "id=123&name=test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateQuery(query)
	}
}
