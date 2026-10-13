package waf

import "testing"

func TestJSONInspector_Depth(t *testing.T) {
	inspector := NewJSONInspector(3)
	valid := []byte(`{"a":{"b":{"c":1}}}`)
	invalid := []byte(`{"a":{"b":{"c":{"d":1}}}}`)

	if !inspector.InspectStream(valid) {
		t.Error("Expected valid depth")
	}
	if inspector.InspectStream(invalid) {
		t.Error("Expected invalid depth")
	}
}
