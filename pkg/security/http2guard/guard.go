// Package http2guard provides a dependency-free, lock-free HTTP/2 abuse guard.
//
// The package deliberately depends only on the Go standard library. Higher-level
// HTTP/2 frame translation belongs in package proxy so that policy does not depend
// on transport implementations and cannot introduce an import cycle.
package http2guard

import (
	"fmt"
	"time"
)

// MaxStreamID is the largest valid HTTP/2 stream identifier.
const MaxStreamID uint32 = 1<<31 - 1

// EventKind identifies an HTTP/2 event charged by a Guard.
type EventKind uint8

const (
	// EventUnknown is an invalid or unsupported event.
	EventUnknown EventKind = iota
	// EventStreamOpen represents a newly opened client stream.
	EventStreamOpen
	// EventStreamReset represents an RST_STREAM cancellation.
	EventStreamReset
	// EventSettings represents a non-ACK SETTINGS frame.
	EventSettings
	// EventSettingsACK represents a SETTINGS acknowledgement.
	EventSettingsACK
)

// Event is a fixed-size description of an HTTP/2 protocol event.
type Event struct {
	Kind     EventKind
	StreamID uint32
}

// Config configures one per-connection Guard.
type Config struct {
	// ConnectionID selects the process-wide shard for this connection.
	ConnectionID uint64
	// RapidReset charges stream opens and RST_STREAM cancellations.
	RapidReset BudgetConfig
	// Settings charges SETTINGS frames and SETTINGS acknowledgements.
	Settings BudgetConfig
	// Process optionally points to a ProcessLimiter shared by all connections.
	Process *ProcessLimiter
	// Now optionally supplies a deterministic, concurrency-safe clock.
	Now func() time.Time
}

// DefaultConfig returns conservative per-connection admission defaults.
func DefaultConfig() Config {
	return Config{
		RapidReset: BudgetConfig{Burst: 32, Window: time.Second},
		Settings:   BudgetConfig{Burst: 16, Window: 10 * time.Second},
	}
}

// Accounting contains the current per-connection budget counters.
type Accounting struct {
	RapidEvents    uint32
	SettingsEvents uint32
}

// Guard applies independent rapid-reset and SETTINGS budgets to one connection.
// Guard is safe for concurrent use and must not be copied after first use.
type Guard struct {
	rapid        Budget
	settings     Budget
	process      *ProcessLimiter
	connectionID uint64
	clock        func() time.Time
}

// NewGuard creates a per-connection HTTP/2 admission guard.
func NewGuard(config Config) (*Guard, error) {
	rapid, err := newBudget(config.RapidReset)
	if err != nil {
		return nil, fmt.Errorf("rapid-reset budget: %w", err)
	}
	settings, err := newBudget(config.Settings)
	if err != nil {
		return nil, fmt.Errorf("settings budget: %w", err)
	}

	clock := config.Now
	if clock == nil {
		clock = time.Now
	}

	return &Guard{
		rapid:        rapid,
		settings:     settings,
		process:      config.Process,
		connectionID: config.ConnectionID,
		clock:        clock,
	}, nil
}

// Admit evaluates and charges one protocol event.
func (g *Guard) Admit(event Event) Verdict {
	if g == nil || g.clock == nil {
		return Verdict{Action: ActionReject, Reason: ReasonInvalidState}
	}
	if !event.valid() {
		return Verdict{Action: ActionReject, Reason: ReasonMalformedEvent}
	}

	now := g.clock()
	budget := &g.settings
	reason := ReasonSettingsBurstExceeded
	if event.isRapid() {
		budget = &g.rapid
		reason = ReasonRapidResetBurstExceeded
	}
	if !budget.Consume(now) {
		return Verdict{Action: ActionReject, Reason: reason}
	}

	if g.process != nil && !g.process.Consume(g.connectionID, event.Kind, now) {
		return Verdict{Action: ActionReject, Reason: processReason(event)}
	}
	return Verdict{Action: ActionAllow, Reason: ReasonNone}
}

// AdmitStreamOpen evaluates a newly opened stream.
func (g *Guard) AdmitStreamOpen(streamID uint32) Verdict {
	return g.Admit(Event{Kind: EventStreamOpen, StreamID: streamID})
}

// AdmitStreamReset evaluates an RST_STREAM cancellation.
func (g *Guard) AdmitStreamReset(streamID uint32) Verdict {
	return g.Admit(Event{Kind: EventStreamReset, StreamID: streamID})
}

// AdmitSettings evaluates a non-ACK SETTINGS frame.
func (g *Guard) AdmitSettings() Verdict {
	return g.Admit(Event{Kind: EventSettings})
}

// AdmitSettingsACK evaluates a SETTINGS acknowledgement.
func (g *Guard) AdmitSettingsACK() Verdict {
	return g.Admit(Event{Kind: EventSettingsACK})
}

// Snapshot returns the current per-connection counters at now.
func (g *Guard) Snapshot(now time.Time) Accounting {
	if g == nil {
		return Accounting{}
	}
	return Accounting{
		RapidEvents:    g.rapid.Count(now),
		SettingsEvents: g.settings.Count(now),
	}
}

func (e Event) valid() bool {
	switch e.Kind {
	case EventStreamOpen, EventStreamReset:
		return e.StreamID != 0 && e.StreamID <= MaxStreamID
	case EventSettings, EventSettingsACK:
		return e.StreamID == 0
	default:
		return false
	}
}

func (e Event) isRapid() bool {
	return e.Kind == EventStreamOpen || e.Kind == EventStreamReset
}

func processReason(event Event) Reason {
	if event.isRapid() {
		return ReasonProcessRapidResetLimitExceeded
	}
	return ReasonProcessSettingsLimitExceeded
}

// ProcessConfig configures sharded, process-wide admission limits.
// Burst values are aggregate limits distributed across the configured shards.
type ProcessConfig struct {
	RapidReset BudgetConfig
	Settings   BudgetConfig
	Shards     uint16
}

// DefaultProcessConfig returns conservative aggregate process-wide defaults.
func DefaultProcessConfig() ProcessConfig {
	return ProcessConfig{
		RapidReset: BudgetConfig{Burst: 4096, Window: time.Second},
		Settings:   BudgetConfig{Burst: 1024, Window: 10 * time.Second},
		Shards:     64,
	}
}

// ProcessAccounting contains aggregate counters across process-wide shards.
type ProcessAccounting struct {
	RapidEvents    uint64
	SettingsEvents uint64
}

type processShard struct {
	rapid    Budget
	settings Budget
}

// ProcessLimiter enforces aggregate limits using independently padded shards.
// The same ProcessLimiter should be shared by every connection Guard.
type ProcessLimiter struct {
	shards []processShard
}

// NewProcessLimiter creates a sharded process-wide limiter.
func NewProcessLimiter(config ProcessConfig) (*ProcessLimiter, error) {
	if config.Shards == 0 {
		return nil, fmt.Errorf("process limiter requires at least one shard")
	}
	if err := validateAggregateBudget(config.RapidReset, config.Shards); err != nil {
		return nil, fmt.Errorf("rapid-reset process budget: %w", err)
	}
	if err := validateAggregateBudget(config.Settings, config.Shards); err != nil {
		return nil, fmt.Errorf("settings process budget: %w", err)
	}

	shards := make([]processShard, int(config.Shards))
	rapidBase := config.RapidReset.Burst / uint32(config.Shards)
	rapidRemainder := config.RapidReset.Burst % uint32(config.Shards)
	settingsBase := config.Settings.Burst / uint32(config.Shards)
	settingsRemainder := config.Settings.Burst % uint32(config.Shards)

	for index := range shards {
		rapidLimit := rapidBase
		if index < int(rapidRemainder) {
			rapidLimit++
		}
		settingsLimit := settingsBase
		if index < int(settingsRemainder) {
			settingsLimit++
		}

		rapid, err := newBudget(BudgetConfig{Burst: rapidLimit, Window: config.RapidReset.Window})
		if err != nil {
			return nil, fmt.Errorf("rapid-reset shard %d: %w", index, err)
		}
		settings, err := newBudget(BudgetConfig{Burst: settingsLimit, Window: config.Settings.Window})
		if err != nil {
			return nil, fmt.Errorf("settings shard %d: %w", index, err)
		}
		shards[index].rapid = rapid
		shards[index].settings = settings
	}

	return &ProcessLimiter{shards: shards}, nil
}

// Consume charges an event to the shard selected by connectionID.
func (p *ProcessLimiter) Consume(connectionID uint64, kind EventKind, now time.Time) bool {
	if p == nil || len(p.shards) == 0 {
		return true
	}

	index := int(mix64(connectionID) % uint64(len(p.shards)))
	shard := &p.shards[index]
	switch kind {
	case EventStreamOpen, EventStreamReset:
		return shard.rapid.Consume(now)
	case EventSettings, EventSettingsACK:
		return shard.settings.Consume(now)
	default:
		return false
	}
}

// Snapshot returns aggregate counters across all shards at now.
func (p *ProcessLimiter) Snapshot(now time.Time) ProcessAccounting {
	if p == nil {
		return ProcessAccounting{}
	}

	var accounting ProcessAccounting
	for index := range p.shards {
		accounting.RapidEvents += uint64(p.shards[index].rapid.Count(now))
		accounting.SettingsEvents += uint64(p.shards[index].settings.Count(now))
	}
	return accounting
}

// Shards returns the number of process-wide shards.
func (p *ProcessLimiter) Shards() int {
	if p == nil {
		return 0
	}
	return len(p.shards)
}

func validateAggregateBudget(config BudgetConfig, shards uint16) error {
	if config.Burst > 0 && config.Window < MinimumWindow {
		return fmt.Errorf("budget window %s is below minimum %s", config.Window, MinimumWindow)
	}
	if config.Window < 0 {
		return fmt.Errorf("budget window must not be negative: %s", config.Window)
	}
	if uint64(config.Burst) > uint64(MaxBurst)*uint64(shards) {
		return fmt.Errorf("aggregate burst %d exceeds shard capacity", config.Burst)
	}
	return nil
}

func mix64(value uint64) uint64 {
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return value ^ (value >> 31)
}
