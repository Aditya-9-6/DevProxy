package waf

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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

	t.Run("CollisionStress", func(t *testing.T) {
		// Generate many unique keys to ensure bitset doesn't trigger false positives easily
		var sb strings.Builder
		for i := 0; i < 50; i++ {
			sb.WriteString(fmt.Sprintf("key%d=val&", i))
		}
		query := sb.String()
		if !validator.ValidateQuery(query) {
			t.Error("Expected high-cardinality unique query to pass")
		}
	})

	t.Run("CaseSensitivity", func(t *testing.T) {
		if validator.ValidateQuery("ID=123&id=456") {
			t.Error("Expected case-insensitive pollution to fail")
		}
	})

	t.Run("URLEncodedKeys", func(t *testing.T) {
		if validator.ValidateQuery("%69d=123&id=456") {
			t.Error("Expected URL-encoded key pollution to fail")
		}
	})

	t.Run("MalformedAndSemicolonDelimiters", func(t *testing.T) {
		if !validator.ValidateQuery("a=1;b=2") {
			t.Error("Expected semicolon delimiter query to pass")
		}
		if validator.ValidateQuery("a=1;a=2") {
			t.Error("Expected semicolon delimiter duplicate query to fail")
		}
	})

	t.Run("EmptyAndSingleValues", func(t *testing.T) {
		if !validator.ValidateQuery("") {
			t.Error("Expected empty query to pass")
		}
		if !validator.ValidateQuery("solo") {
			t.Error("Expected solo parameter key to pass")
		}
	})
}

func TestRecordParamUsageConcurrency(t *testing.T) {
	validator := NewParameterValidator()
	var wg sync.WaitGroup
	routines := 50
	iter := 100

	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iter; j++ {
				validator.RecordParamUsage("test")
			}
		}()
	}
	wg.Wait()

	idx := hashParam("test")
	count := atomic.LoadUint64(&validator.Counters[idx][0])
	expected := uint64(routines * iter)
	if count != expected {
		t.Errorf("Expected atomic counter to be %d, got %d", expected, count)
	}
}

func BenchmarkParameterValidation(b *testing.B) {
	validator := NewParameterValidator()
	query := "id=123&name=test&user=admin"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateQuery(query)
	}
}
