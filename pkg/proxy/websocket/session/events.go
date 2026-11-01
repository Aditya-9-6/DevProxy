package session

type EventType int

const (
	EventOpen EventType = iota
	EventClose
)