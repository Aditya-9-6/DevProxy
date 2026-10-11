package waf

import (
	"sync"
)

const AlphabetSize = 256

type Automaton struct {
	nextStates []int32
	failStates []int32
	outputs    [][]int
	patterns   [][]byte
}

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65536)
	},
}

func NewAutomaton(patterns [][]byte) *Automaton {
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

	stateCount := 1
	for idx, p := range patterns {
		curr := 0
		for _, b := range p {
			if a.nextStates[curr*AlphabetSize+int(b)] == -1 {
				a.nextStates[curr*AlphabetSize+int(b)] = int32(stateCount)
				stateCount++
			}
			curr = int(a.nextStates[curr*AlphabetSize+int(b)])
		}
		a.outputs[curr] = append(a.outputs[curr], idx)
	}

	queue := make([]int, 0, stateCount)
	for i := 0; i < AlphabetSize; i++ {
		if next := a.nextStates[i]; next != -1 {
			queue = append(queue, int(next))
		} else {
			a.nextStates[i] = 0
		}
	}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for i := 0; i < AlphabetSize; i++ {
			if v := a.nextStates[u*AlphabetSize+i]; v != -1 {
				f := a.failStates[u]
				a.failStates[int(v)] = a.nextStates[int(f)*AlphabetSize+i]
				a.outputs[int(v)] = append(a.outputs[int(v)], a.outputs[int(a.failStates[int(v)])]...)
				queue = append(queue, int(v))
			} else {
				a.nextStates[u*AlphabetSize+i] = a.nextStates[int(a.failStates[u])*AlphabetSize+i]
			}
		}
	}
	return a
}

func (a *Automaton) Scan(data []byte, callback func(patternIdx int, pos int)) {
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
