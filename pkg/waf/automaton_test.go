package waf

import (
	"reflect"
	"sort"
	"testing"
)

func TestNewAutomaton_ValidPatterns(t *testing.T) {
	patterns := [][]byte{
		[]byte("he"),
		[]byte("she"),
		[]byte("his"),
		[]byte("hers"),
	}

	a := NewAutomaton(patterns)
	if a == nil {
		t.Fatal("expected non-nil Automaton")
	}
	if len(a.patterns) != 4 {
		t.Errorf("expected 4 patterns, got %d", len(a.patterns))
	}
	if len(a.nextStates) == 0 {
		t.Errorf("expected nextStates to be initialized")
	}
	if len(a.failStates) == 0 {
		t.Errorf("expected failStates to be initialized")
	}
	if len(a.outputs) == 0 {
		t.Errorf("expected outputs to be initialized")
	}
}

func TestNewAutomaton_ExceedsMaxPatternLength(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when pattern length exceeds maximum")
		}
	}()

	longPattern := make([]byte, MaxPatternLength+1)
	NewAutomaton([][]byte{longPattern})
}

func TestNewAutomaton_ExceedsMaxTotalPatternBytes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when total pattern bytes exceed maximum")
		}
	}()

	var patterns [][]byte
	bytesPerPattern := 1024
	numPatterns := (MaxTotalPatternBytes / bytesPerPattern) + 2

	for i := 0; i < numPatterns; i++ {
		patterns = append(patterns, make([]byte, bytesPerPattern))
	}

	NewAutomaton(patterns)
}

func TestNewAutomaton_EmptyData(t *testing.T) {
	// Empty patterns list
	a := NewAutomaton([][]byte{})
	if a == nil {
		t.Fatal("expected non-nil Automaton even with empty patterns")
	}

	// Scan with empty data
	called := false
	a.Scan([]byte{}, func(patternIdx int, pos int) {
		called = true
	})
	if called {
		t.Errorf("callback should not be called for empty data")
	}

	// Scan with nil automaton
	var nilA *Automaton
	nilA.Scan([]byte("test"), func(patternIdx int, pos int) {
		called = true
	})
	if called {
		t.Errorf("callback should not be called for nil automaton")
	}
}

func TestNewAutomaton_EmptyPatternString(t *testing.T) {
	patterns := [][]byte{
		[]byte("he"),
		[]byte(""),
		[]byte("she"),
	}
	a := NewAutomaton(patterns)
	if a == nil {
		t.Fatal("expected non-nil Automaton")
	}

	type match struct {
		patternIdx int
		pos        int
	}
	var got []match
	a.Scan([]byte("she"), func(patternIdx int, pos int) {
		got = append(got, match{patternIdx, pos})
	})

	if len(got) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(got), got)
	}
}

func TestAutomaton_Scan(t *testing.T) {
	patterns := [][]byte{
		[]byte("he"),   // 0
		[]byte("she"),  // 1
		[]byte("his"),  // 2
		[]byte("hers"), // 3
	}

	a := NewAutomaton(patterns)

	type match struct {
		patternIdx int
		pos        int
	}

	tests := []struct {
		name     string
		data     []byte
		expected []match
	}{
		{
			name: "ushers",
			data: []byte("ushers"),
			expected: []match{
				{1, 3}, // "she" ends at index 3
				{0, 3}, // "he" ends at index 3
				{3, 5}, // "hers" ends at index 5
			},
		},
		{
			name: "his and hers",
			data: []byte("his and hers"),
			expected: []match{
				{2, 2},  // "his" ends at index 2
				{0, 9},  // "he" ends at index 9
				{3, 11}, // "hers" ends at index 11
			},
		},
		{
			name: "hello",
			data: []byte("hello"),
			expected: []match{
				{0, 1}, // "he" ends at index 1
			},
		},
		{
			name: "completely disjoint",
			data: []byte("foo bar baz"),
			expected: []match{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []match
			a.Scan(tt.data, func(patternIdx int, pos int) {
				got = append(got, match{patternIdx, pos})
			})

			// sort both slices to ensure order doesn't matter
			sort.Slice(got, func(i, j int) bool {
				if got[i].pos == got[j].pos {
					return got[i].patternIdx < got[j].patternIdx
				}
				return got[i].pos < got[j].pos
			})

			sort.Slice(tt.expected, func(i, j int) bool {
				if tt.expected[i].pos == tt.expected[j].pos {
					return tt.expected[i].patternIdx < tt.expected[j].patternIdx
				}
				return tt.expected[i].pos < tt.expected[j].pos
			})

			if len(got) == 0 && len(tt.expected) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
