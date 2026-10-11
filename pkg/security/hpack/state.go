package hpack

// State represents the current validation state of the HPACK header block.
// It uses a bitmask for pseudo-headers to guarantee zero heap allocations.
type State struct {
	SeenPseudoHeaders uint8
	LastWasPseudo     bool
	Error             error
}

// NewState creates and returns a new zero-allocation validation state.
func NewState() *State {
	return &State{
		SeenPseudoHeaders: 0,
		LastWasPseudo:     true,
	}
}
