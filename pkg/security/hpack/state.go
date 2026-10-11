package hpack

// State represents the current validation state of the HPACK header block.
type State struct {
	SeenPseudoHeaders map[string]bool
	LastWasPseudo     bool
	Error             error
}

func NewState() *State {
	return &State{
		SeenPseudoHeaders: make(map[string]bool),
		LastWasPseudo:     true,
	}
}
