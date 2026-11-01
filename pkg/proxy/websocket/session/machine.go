package session

type State int

const (
	StateIdle State = iota
	StateConnected
	StateClosed
)

type Machine struct {
	state State
}

func NewMachine() *Machine {
	return &Machine{state: StateIdle}
}