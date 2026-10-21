package http2guard

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var allocationSink atomic.Uint64

type testClock struct {
	nanos atomic.Int64
}

func newTestClock() *testClock {
	clock := &testClock{}
	clock.nanos.Store(int64(100 * time.Second))
	return clock
}

func (c *testClock) Now() time.Time {
	return time.Unix(0, c.nanos.Load())
}

func (c *testClock) Advance(duration time.Duration) {
	c.nanos.Add(int64(duration))
}

func TestBudgetConsume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    BudgetConfig
		attempts  int
		wantAllow int
		wantCount uint32
	}{
		{name: "disabled", config: BudgetConfig{}, attempts: 3, wantAllow: 3, wantCount: 0},
		{name: "single event", config: BudgetConfig{Burst: 1, Window: time.Second}, attempts: 2, wantAllow: 1, wantCount: 1},
		{name: "burst boundary", config: BudgetConfig{Burst: 3, Window: time.Second}, attempts: 5, wantAllow: 3, wantCount: 3},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			budget, err := NewBudget(test.config)
			if err != nil {
				t.Fatalf("NewBudget() error = %v", err)
			}

			allowed := 0
			now := time.Unix(100, 0)
			for attempt := 0; attempt < test.attempts; attempt++ {
				if budget.Consume(now) {
					allowed++
				}
			}
			if allowed != test.wantAllow {
				t.Fatalf("allowed = %d, want %d", allowed, test.wantAllow)
			}
			if got := budget.Count(now); got != test.wantCount {
				t.Fatalf("Count() = %d, want %d", got, test.wantCount)
			}
		})
	}
}

func TestBudgetRecoveryAndReset(t *testing.T) {
	t.Parallel()

	budget, err := NewBudget(BudgetConfig{Burst: 2, Window: time.Second})
	if err != nil {
		t.Fatalf("NewBudget() error = %v", err)
	}

	now := time.Unix(100, 0)
	if !budget.Consume(now) || !budget.Consume(now) {
		t.Fatal("expected first two events to fit budget")
	}
	if got := budget.Count(now); got != 2 {
		t.Fatalf("Count() = %d, want 2", got)
	}

	budget.Reset()
	if got := budget.Count(now); got != 0 {
		t.Fatalf("Count() after Reset = %d, want 0", got)
	}
	if !budget.Consume(now) {
		t.Fatal("expected event after reset to fit budget")
	}
	if got := budget.Count(now.Add(time.Second)); got != 0 {
		t.Fatalf("Count() in new window = %d, want 0", got)
	}
}

func TestBudgetInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config BudgetConfig
	}{
		{name: "burst too large", config: BudgetConfig{Burst: MaxBurst + 1, Window: time.Second}},
		{name: "missing window", config: BudgetConfig{Burst: 1}},
		{name: "negative window", config: BudgetConfig{Burst: 1, Window: -time.Second}},
		{name: "sub-millisecond window", config: BudgetConfig{Burst: 1, Window: time.Microsecond}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewBudget(test.config); err == nil {
				t.Fatal("NewBudget() error = nil, want error")
			}
		})
	}
}

func TestGuardSeparateBurstBudgets(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	config := DefaultConfig()
	config.RapidReset.Burst = 2
	config.Settings.Burst = 2
	config.Now = clock.Now
	guard, err := NewGuard(config)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	if verdict := guard.AdmitStreamOpen(1); !verdict.Allowed() {
		t.Fatalf("first rapid event rejected: %+v", verdict)
	}
	if verdict := guard.AdmitStreamReset(1); !verdict.Allowed() {
		t.Fatalf("second rapid event rejected: %+v", verdict)
	}
	if verdict := guard.AdmitStreamOpen(3); verdict.Reason != ReasonRapidResetBurstExceeded {
		t.Fatalf("rapid verdict = %+v, want rapid burst rejection", verdict)
	}
	if verdict := guard.AdmitSettings(); !verdict.Allowed() {
		t.Fatalf("settings event consumed rapid budget: %+v", verdict)
	}
	if verdict := guard.AdmitSettingsACK(); !verdict.Allowed() {
		t.Fatalf("settings ACK rejected: %+v", verdict)
	}
	if verdict := guard.AdmitSettings(); verdict.Reason != ReasonSettingsBurstExceeded {
		t.Fatalf("settings verdict = %+v, want settings burst rejection", verdict)
	}
}

func TestGuardMalformedEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event Event
	}{
		{name: "unknown kind", event: Event{Kind: EventUnknown}},
		{name: "zero stream", event: Event{Kind: EventStreamOpen}},
		{name: "stream overflow", event: Event{Kind: EventStreamOpen, StreamID: MaxStreamID + 1}},
		{name: "settings with stream", event: Event{Kind: EventSettings, StreamID: 1}},
		{name: "settings ACK with stream", event: Event{Kind: EventSettingsACK, StreamID: 1}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			guard, err := NewGuard(DefaultConfig())
			if err != nil {
				t.Fatalf("NewGuard() error = %v", err)
			}
			verdict := guard.Admit(test.event)
			if !verdict.Rejected() || verdict.Reason != ReasonMalformedEvent {
				t.Fatalf("verdict = %+v, want malformed event rejection", verdict)
			}
		})
	}
}

func TestGuardRecovery(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	config := DefaultConfig()
	config.RapidReset.Burst = 1
	config.Now = clock.Now
	guard, err := NewGuard(config)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	if verdict := guard.AdmitStreamOpen(1); !verdict.Allowed() {
		t.Fatalf("first event rejected: %+v", verdict)
	}
	if verdict := guard.AdmitStreamReset(1); verdict.Reason != ReasonRapidResetBurstExceeded {
		t.Fatalf("second verdict = %+v, want burst rejection", verdict)
	}

	clock.Advance(time.Second)
	if verdict := guard.AdmitStreamReset(1); !verdict.Allowed() {
		t.Fatalf("event after recovery rejected: %+v", verdict)
	}
}

func buildSharedProcessGuards(t *testing.T) (*Guard, *Guard, *Guard, *ProcessLimiter, *testClock) {
	t.Helper()

	clock := newTestClock()
	process, err := NewProcessLimiter(ProcessConfig{
		RapidReset: BudgetConfig{Burst: 2, Window: time.Second},
		Settings:   BudgetConfig{Burst: 1, Window: time.Second},
		Shards:     1,
	})
	if err != nil {
		t.Fatalf("NewProcessLimiter() error = %v", err)
	}

	newGuard := func(connectionID uint64) *Guard {
		config := DefaultConfig()
		config.ConnectionID = connectionID
		config.Process = process
		config.Now = clock.Now
		guard, err := NewGuard(config)
		if err != nil {
			t.Fatalf("NewGuard() error = %v", err)
		}
		return guard
	}

	return newGuard(1), newGuard(2), newGuard(3), process, clock
}

func TestGuardProcessWideLimit(t *testing.T) {
	t.Parallel()

	first, second, third, process, clock := buildSharedProcessGuards(t)
	if verdict := first.AdmitStreamOpen(1); !verdict.Allowed() {
		t.Fatalf("first connection rejected: %+v", verdict)
	}
	if verdict := second.AdmitStreamReset(1); !verdict.Allowed() {
		t.Fatalf("second connection rejected: %+v", verdict)
	}
	if verdict := third.AdmitStreamOpen(3); verdict.Reason != ReasonProcessRapidResetLimitExceeded {
		t.Fatalf("third verdict = %+v, want process rapid rejection", verdict)
	}

	if verdict := first.AdmitSettings(); !verdict.Allowed() {
		t.Fatalf("first settings event rejected: %+v", verdict)
	}
	if verdict := second.AdmitSettingsACK(); verdict.Reason != ReasonProcessSettingsLimitExceeded {
		t.Fatalf("settings verdict = %+v, want process settings rejection", verdict)
	}

	clock.Advance(time.Second)
	if verdict := third.AdmitStreamOpen(3); !verdict.Allowed() {
		t.Fatalf("event after process recovery rejected: %+v", verdict)
	}
	if got := process.Snapshot(clock.Now()).RapidEvents; got != 1 {
		t.Fatalf("process rapid count = %d, want 1", got)
	}
}

func TestGuardConcurrentAccounting(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	config := DefaultConfig()
	config.RapidReset.Burst = MaxBurst
	config.Now = clock.Now
	guard, err := NewGuard(config)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	const workers = 32
	const eventsPerWorker = 100
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer waitGroup.Done()
			for event := 0; event < eventsPerWorker; event++ {
				guard.AdmitStreamOpen(uint32(worker*eventsPerWorker + event + 1))
			}
		}(worker)
	}
	waitGroup.Wait()

	if got := guard.Snapshot(clock.Now()).RapidEvents; got != workers*eventsPerWorker {
		t.Fatalf("rapid count = %d, want %d", got, workers*eventsPerWorker)
	}
}

func TestGuardHotPathAllocationFree(t *testing.T) {
	// Keep this test serial. testing.AllocsPerRun panics while parallel tests
	// are active, and concurrent accounting is covered by the test above.
	config := DefaultConfig()
	config.RapidReset.Burst = MaxBurst
	guard, err := NewGuard(config)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	allocations := testing.AllocsPerRun(1000, func() {
		verdict := guard.AdmitStreamOpen(1)
		allocationSink.Store(uint64(verdict.Action))
	})
	if allocations != 0 {
		t.Fatalf("Admit() allocations = %v, want 0", allocations)
	}
}

func TestConstructorErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{name: "rapid burst too large", config: Config{RapidReset: BudgetConfig{Burst: MaxBurst + 1, Window: time.Second}}},
		{name: "rapid missing window", config: Config{RapidReset: BudgetConfig{Burst: 1}}},
		{name: "settings negative window", config: Config{Settings: BudgetConfig{Burst: 1, Window: -time.Second}}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewGuard(test.config); err == nil {
				t.Fatal("NewGuard() error = nil, want error")
			}
		})
	}

	if _, err := NewProcessLimiter(ProcessConfig{}); err == nil {
		t.Fatal("NewProcessLimiter() with zero shards error = nil, want error")
	}
	invalid := ProcessConfig{Shards: 1, RapidReset: BudgetConfig{Burst: MaxBurst + 1, Window: time.Second}}
	if _, err := NewProcessLimiter(invalid); err == nil {
		t.Fatal("NewProcessLimiter() aggregate overflow error = nil, want error")
	}
}

func TestVerdict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		verdict   Verdict
		wantAllow bool
	}{
		{name: "allow", verdict: Verdict{Action: ActionAllow, Reason: ReasonNone}, wantAllow: true},
		{name: "reject", verdict: Verdict{Action: ActionReject, Reason: ReasonMalformedFrame}, wantAllow: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.verdict.Allowed(); got != test.wantAllow {
				t.Fatalf("Allowed() = %v, want %v", got, test.wantAllow)
			}
			if got := test.verdict.Rejected(); got == test.wantAllow {
				t.Fatalf("Rejected() = %v, want %v", got, !test.wantAllow)
			}
		})
	}
}

func FuzzGuardAdmit(f *testing.F) {
	clock := newTestClock()
	config := DefaultConfig()
	config.RapidReset.Burst = MaxBurst
	config.Now = clock.Now
	guard, err := NewGuard(config)
	if err != nil {
		f.Fatalf("NewGuard() error = %v", err)
	}

	f.Add([]byte{})
	f.Add([]byte{byte(EventStreamOpen), 0, 0, 0, 1})
	f.Add([]byte{byte(EventSettings), 0, 0, 0, 1})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			t.Skip()
		}
		kind := EventKind(data[0])
		streamID := uint32(data[1%len(data)])
		verdict := guard.Admit(Event{Kind: kind, StreamID: streamID})
		if verdict.Action != ActionAllow && verdict.Action != ActionReject {
			t.Fatalf("invalid action %d", verdict.Action)
		}
	})
}
