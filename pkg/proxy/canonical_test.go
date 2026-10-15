package proxy

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewCanonicalEngine(t *testing.T) {
	engine := NewCanonicalEngine()
	if engine == nil {
		data := "expected non-nil engine"
		t.Fatalf("%s", data)
	}

	val, ok := engine.Lookup("server")
	if !ok || val != "DevProxy/1.0.0" {
		data := "expected server header to be DevProxy/1.0.0"
		t.Errorf("%s", data)
	}
}

func TestCanonicalEngineLookup(t *testing.T) {
	engine := NewCanonicalEngine()

	tests := []struct {
		name      string
		inputKey  string
		wantValue string
		wantFound bool
	}{
		{
			name:      "existing lowercase server",
			inputKey:  "server",
			wantValue: "DevProxy/1.0.0",
			wantFound: true,
		},
		{
			name:      "existing powered-by",
			inputKey:  "x-powered-by",
			wantValue: "DevProxy-Secure",
			wantFound: true,
		},
		{
			name:      "non-existent header",
			inputKey:  "x-non-existent",
			wantValue: "",
			wantFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, found := engine.Lookup(tc.inputKey)
			if found != tc.wantFound {
				t.Errorf("Lookup(%q) found = %v, want %v", tc.inputKey, found, tc.wantFound)
			}
			if found && val != tc.wantValue {
				t.Errorf("Lookup(%q) value = %q, want %q", tc.inputKey, val, tc.wantValue)
			}
		})
	}
}

func TestCanonicalEngineSetRules(t *testing.T) {
	engine := NewCanonicalEngine()

	newRules := map[string]string{
		"x-custom-header": "CustomValue",
	}

	engine.SetRules(newRules)

	val, ok := engine.Lookup("x-custom-header")
	if !ok || val != "CustomValue" {
		t.Errorf("expected updated rule to be present, got %q, found %v", val, ok)
	}

	_, okOld := engine.Lookup("server")
	if okOld {
		t.Errorf("expected old rules to be replaced")
	}
}

func TestCanonicalEngineCanonicalizeBytes(t *testing.T) {
	engine := NewCanonicalEngine()

	res := engine.CanonicalizeBytes("server", []byte("OldServer/1.0"))
	if !bytes.Equal(res, []byte("DevProxy/1.0.0")) {
		t.Errorf("CanonicalizeBytes mismatch, got %q", string(res))
	}

	// Non-canonical header should return input bytes unmodified
	resUnknown := engine.CanonicalizeBytes("x-unknown", []byte("OriginalVal"))
	if !bytes.Equal(resUnknown, []byte("OriginalVal")) {
		t.Errorf("CanonicalizeBytes for unknown header modified input: %q", string(resUnknown))
	}
}

func TestCanonicalEngineConcurrencyAndRace(t *testing.T) {
	engine := NewCanonicalEngine()
	var wg sync.WaitGroup
	workers := 50
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					engine.SetRules(map[string]string{
						"server": "DevProxy/Concurrent",
					})
				} else {
					engine.Lookup("server")
					engine.CanonicalizeBytes("server", []byte("SomeValue"))
				}
			}
		}(i)
	}

	wg.Wait()
}
