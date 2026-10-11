package waf

import (
	"sync"
	"testing"
)

func TestAutomaton(t *testing.T) {
	patterns := [][]byte{[]byte("admin"), []byte("root")}
	a := NewAutomaton(patterns)
	data := []byte("login as admin or root")
	matches := make(map[int]int)
	a.Scan(data, func(pIdx, pos int) {
		matches[pIdx] = pos
	})
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
}

func TestAutomaton_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		patterns        [][]byte
		data            []byte
		wantMatchCount  int
		wantPatternSeen map[int]bool
	}{
		{
			name:            "Empty data",
			patterns:        [][]byte{[]byte("test")},
			data:            []byte(""),
			wantMatchCount:  0,
			wantPatternSeen: map[int]bool{},
		},
		{
			name:            "No matches",
			patterns:        [][]byte{[]byte("alpha"), []byte("beta")},
			data:            []byte("gamma delta epsilon"),
			wantMatchCount:  0,
			wantPatternSeen: map[int]bool{},
		},
		{
			name:            "Overlapping patterns (he, she, his, hers)",
			patterns:        [][]byte{[]byte("he"), []byte("she"), []byte("his"), []byte("hers")},
			data:            []byte("ushers"),
			wantMatchCount:  3, // "she" at 4, "he" at 4, "hers" at 5
			wantPatternSeen: map[int]bool{0: true, 1: true, 3: true},
		},
		{
			name:            "Pattern longer than input",
			patterns:        [][]byte{[]byte("verylongpatterndoesnotfit")},
			data:            []byte("short"),
			wantMatchCount:  0,
			wantPatternSeen: map[int]bool{},
		},
		{
			name:            "Non-ASCII UTF-8 match",
			patterns:        [][]byte{[]byte("世界"), []byte("こんにちは")},
			data:            []byte("hello 世界!"),
			wantMatchCount:  1,
			wantPatternSeen: map[int]bool{0: true},
		},
		{
			name:            "Empty pattern in list is ignored",
			patterns:        [][]byte{[]byte(""), []byte("target")},
			data:            []byte("find target now"),
			wantMatchCount:  1,
			wantPatternSeen: map[int]bool{1: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAutomaton(tc.patterns)
			matched := 0
			seen := make(map[int]bool)
			a.Scan(tc.data, func(pIdx, pos int) {
				matched++
				seen[pIdx] = true
			})

			if matched != tc.wantMatchCount {
				t.Errorf("got %d matches, want %d", matched, tc.wantMatchCount)
			}
			for pIdx := range tc.wantPatternSeen {
				if !seen[pIdx] {
					t.Errorf("expected pattern idx %d to match, but was not seen", pIdx)
				}
			}
		})
	}
}

func TestAutomaton_ValidationLimits(t *testing.T) {
	t.Run("pattern exceeds max length panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for pattern exceeding MaxPatternLength")
			}
		}()
		oversized := make([]byte, MaxPatternLength+1)
		NewAutomaton([][]byte{oversized})
	})

	t.Run("total pattern bytes exceeds max limit panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for total pattern bytes exceeding limit")
			}
		}()
		patterns := make([][]byte, 300)
		for i := range patterns {
			patterns[i] = make([]byte, 4000)
		}
		NewAutomaton(patterns)
	})
}

func TestAutomatonConcurrent(t *testing.T) {
	patterns := [][]byte{[]byte("test"), []byte("concurrency")}
	a := NewAutomaton(patterns)
	data := []byte("this is a test for concurrency safety")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count := 0
			a.Scan(data, func(pIdx, pos int) {
				count++
			})
			if count != 2 {
				t.Errorf("expected 2 matches concurrently, got %d", count)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkAutomatonScan(b *testing.B) {
	patterns := [][]byte{[]byte("admin"), []byte("root"), []byte("user"), []byte("password")}
	a := NewAutomaton(patterns)
	data := []byte("the quick brown fox jumps over the lazy admin and root user with password")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a.Scan(data, func(pIdx, pos int) {
			// Zero allocation match handler
		})
	}
}
