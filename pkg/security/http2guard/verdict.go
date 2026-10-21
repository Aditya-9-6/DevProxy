package http2guard

// Action is the admission action returned by a Guard.
type Action uint8

const (
	// ActionAllow permits the protocol event.
	ActionAllow Action = iota
	// ActionReject rejects the protocol event.
	ActionReject
)

// Reason identifies why an event was admitted or rejected.
type Reason uint8

const (
	// ReasonNone indicates successful admission.
	ReasonNone Reason = iota
	// ReasonMalformedEvent indicates an invalid direct guard event.
	ReasonMalformedEvent
	// ReasonMalformedFrame indicates an invalid HTTP/2 frame header.
	ReasonMalformedFrame
	// ReasonRapidResetBurstExceeded indicates a per-connection rapid budget rejection.
	ReasonRapidResetBurstExceeded
	// ReasonSettingsBurstExceeded indicates a per-connection SETTINGS budget rejection.
	ReasonSettingsBurstExceeded
	// ReasonProcessRapidResetLimitExceeded indicates a process-wide rapid budget rejection.
	ReasonProcessRapidResetLimitExceeded
	// ReasonProcessSettingsLimitExceeded indicates a process-wide SETTINGS budget rejection.
	ReasonProcessSettingsLimitExceeded
	// ReasonInvalidState indicates that a guard or adapter was not initialized.
	ReasonInvalidState
)

// Verdict is a fixed-size admission result containing no dynamic data.
type Verdict struct {
	Action Action
	Reason Reason
}

// Allowed reports whether the event may proceed.
func (v Verdict) Allowed() bool {
	return v.Action == ActionAllow
}

// Rejected reports whether the event must be rejected.
func (v Verdict) Rejected() bool {
	return v.Action == ActionReject
}
