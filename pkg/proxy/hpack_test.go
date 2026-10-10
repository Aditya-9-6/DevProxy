package proxy

import (
	"bytes"
	"testing"
)

func TestHPACKRoundTrip(t *testing.T) {
	encoder := NewEncoder(4096)
	decoder := NewDecoder(4096)

	origHeaders := []HeaderField{
		{Name: ":method", Value: "GET"},
		{Name: ":path", Value: "/health"},
		{Name: "user-agent", Value: "DevProxy-Test"},
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, origHeaders); err != nil {
		t.Fatalf("Failed to encode headers: %v", err)
	}

	decodedHeaders, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("Failed to decode headers: %v", err)
	}

	if len(decodedHeaders) != len(origHeaders) {
		t.Fatalf("Expected %d headers, got %d", len(origHeaders), len(decodedHeaders))
	}

	for i, hf := range origHeaders {
		if decodedHeaders[i].Name != hf.Name || decodedHeaders[i].Value != hf.Value {
			t.Errorf("Header mismatch at index %d: expected %v, got %v", i, hf, decodedHeaders[i])
		}
	}
}

func TestDynamicTableEviction(t *testing.T) {
	dt := NewDynamicTable(100)                            // Small table
	dt.Add(HeaderField{Name: "header1", Value: "value1"}) // Size: 7 + 6 + 32 = 45
	dt.Add(HeaderField{Name: "header2", Value: "value2"}) // Size: 7 + 6 + 32 = 45 (Total: 90)

	if len(dt.entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(dt.entries))
	}

	// Adding another should trigger eviction
	dt.Add(HeaderField{Name: "header3", Value: "value3"})
	if len(dt.entries) >= 3 {
		t.Fatalf("Expected eviction of oldest entry, got %d entries", len(dt.entries))
	}
}
