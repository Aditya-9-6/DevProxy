package waf

const (
	// AlphabetSize represents the 256 possible byte values.
	AlphabetSize = 256

	// MaxPatternLength defines the maximum byte length of an individual pattern.
	MaxPatternLength = 4096

	// MaxTotalPatternBytes limits the cumulative size of all patterns to prevent OOM.
	MaxTotalPatternBytes = 1048576 // 1MB
)

// Automaton represents a compiled Aho-Corasick string matching automaton.
type Automaton struct {
	nextStates []int32
	failStates []int32
	outputs    [][]int
	patterns   [][]byte
}

func validatePatterns(patterns [][]byte) {
	totalBytes := 0
	for _, p := range patterns {
		if len(p) > MaxPatternLength {
			panic("pattern exceeds maximum allowed length")
		}
		totalBytes += len(p)
	}
	if totalBytes > MaxTotalPatternBytes {
		panic("total pattern bytes exceed maximum allowed limit")
	}
}

// NewAutomaton constructs an Aho-Corasick automaton from a list of byte patterns.
func NewAutomaton(patterns [][]byte) *Automaton {
	validatePatterns(patterns)

	numStates := 1
	for _, p := range patterns {
		numStates += len(p)
	}

	a := &Automaton{
		nextStates: make([]int32, numStates*AlphabetSize),
		failStates: make([]int32, numStates),
		outputs:    make([][]int, numStates),
		patterns:   patterns,
	}

	for i := range a.nextStates {
		a.nextStates[i] = -1
	}

	stateCount := a.insertPatterns(patterns)
	a.buildFailureTransitions(stateCount)
	return a
}

func (a *Automaton) insertPatterns(patterns [][]byte) int {
	stateCount := 1
	for idx, p := range patterns {
		if len(p) == 0 {
			continue
		}
		curr := 0
		for _, b := range p {
			target := curr*AlphabetSize + int(b)
			if a.nextStates[target] == -1 {
				a.nextStates[target] = int32(stateCount)
				stateCount++
			}
			curr = int(a.nextStates[target])
		}
		a.outputs[curr] = append(a.outputs[curr], idx)
	}
	return stateCount
}

func (a *Automaton) buildFailureTransitions(stateCount int) {
	queue := make([]int, 0, stateCount)
	for i := 0; i < AlphabetSize; i++ {
		if next := a.nextStates[i]; next != -1 {
			queue = append(queue, int(next))
		} else {
			a.nextStates[i] = 0
		}
	}

	head := 0
	for head < len(queue) {
		u := queue[head]
		head++

		for i := 0; i < AlphabetSize; i++ {
			v := a.nextStates[u*AlphabetSize+i]
			if v != -1 {
				f := a.failStates[u]
				failTarget := a.nextStates[int(f)*AlphabetSize+i]
				a.failStates[int(v)] = failTarget
				if len(a.outputs[int(failTarget)]) > 0 {
					a.outputs[int(v)] = append(a.outputs[int(v)], a.outputs[int(failTarget)]...)
				}
				queue = append(queue, int(v))
			} else {
				a.nextStates[u*AlphabetSize+i] = a.nextStates[int(a.failStates[u])*AlphabetSize+i]
			}
		}
	}
}

// Scan processes data and invokes callback for each matched pattern with pattern index and end position.
func (a *Automaton) Scan(data []byte, callback func(patternIdx int, pos int)) {
	if len(data) == 0 || a == nil {
		return
	}

	curr := 0
	for i, b := range data {
		curr = int(a.nextStates[curr*AlphabetSize+int(b)])
		if len(a.outputs[curr]) > 0 {
			for _, pIdx := range a.outputs[curr] {
				callback(pIdx, i)
			}
		}
	}
}
