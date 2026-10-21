package proxy_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	devproxy "github.com/Aditya-9-6/DevProxy/pkg/proxy"
	"github.com/Aditya-9-6/DevProxy/pkg/security/http2guard"
	"golang.org/x/net/http2"
)

var adapterAllocationSink atomic.Uint64

func newGuardAndAdapterForTest(t testing.TB, rapidBurst, settingsBurst uint32) (*http2guard.Guard, *devproxy.HTTP2GuardAdapter) {
	t.Helper()

	config := http2guard.DefaultConfig()
	config.RapidReset.Burst = rapidBurst
	config.Settings.Burst = settingsBurst
	guard, err := http2guard.NewGuard(config)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}
	return guard, devproxy.NewHTTP2GuardAdapter(guard)
}

func TestHTTP2GuardAdapterFrameValidation(t *testing.T) {
	t.Parallel()

	_, adapter := newGuardAndAdapterForTest(t, 100, 100)
	tests := []struct {
		name       string
		header     http2.FrameHeader
		wantAction http2guard.Action
		wantReason http2guard.Reason
	}{
		{name: "valid headers", header: http2.FrameHeader{Type: http2.FrameHeaders, StreamID: 1}, wantAction: http2guard.ActionAllow, wantReason: http2guard.ReasonNone},
		{name: "valid reset", header: http2.FrameHeader{Type: http2.FrameRSTStream, Length: 4, StreamID: 1}, wantAction: http2guard.ActionAllow, wantReason: http2guard.ReasonNone},
		{name: "valid settings", header: http2.FrameHeader{Type: http2.FrameSettings, Length: 6}, wantAction: http2guard.ActionAllow, wantReason: http2guard.ReasonNone},
		{name: "valid settings ACK", header: http2.FrameHeader{Type: http2.FrameSettings, Flags: http2.FlagSettingsAck}, wantAction: http2guard.ActionAllow, wantReason: http2guard.ReasonNone},
		{name: "unprotected data", header: http2.FrameHeader{Type: http2.FrameData, Length: 10, StreamID: 1}, wantAction: http2guard.ActionAllow, wantReason: http2guard.ReasonNone},
		{name: "reset length", header: http2.FrameHeader{Type: http2.FrameRSTStream, Length: 3, StreamID: 1}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "reset stream", header: http2.FrameHeader{Type: http2.FrameRSTStream, Length: 4}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "settings length", header: http2.FrameHeader{Type: http2.FrameSettings, Length: 5}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "settings ACK length", header: http2.FrameHeader{Type: http2.FrameSettings, Flags: http2.FlagSettingsAck, Length: 6}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "settings stream", header: http2.FrameHeader{Type: http2.FrameSettings, Length: 6, StreamID: 1}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "headers stream", header: http2.FrameHeader{Type: http2.FrameHeaders}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "headers padding", header: http2.FrameHeader{Type: http2.FrameHeaders, Flags: http2.FlagHeadersPadded, StreamID: 1}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
		{name: "headers priority", header: http2.FrameHeader{Type: http2.FrameHeaders, Flags: http2.FlagHeadersPriority, Length: 4, StreamID: 1}, wantAction: http2guard.ActionReject, wantReason: http2guard.ReasonMalformedFrame},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			verdict := adapter.AdmitFrame(test.header)
			if verdict.Action != test.wantAction || verdict.Reason != test.wantReason {
				t.Fatalf("verdict = %+v, want action=%d reason=%d", verdict, test.wantAction, test.wantReason)
			}
		})
	}
}

func TestHTTP2GuardAdapterSeparateBudgets(t *testing.T) {
	t.Parallel()

	_, adapter := newGuardAndAdapterForTest(t, 2, 2)
	headers := http2.FrameHeader{Type: http2.FrameHeaders, StreamID: 1}
	reset := http2.FrameHeader{Type: http2.FrameRSTStream, Length: 4, StreamID: 1}
	settings := http2.FrameHeader{Type: http2.FrameSettings, Length: 6}
	settingsACK := http2.FrameHeader{Type: http2.FrameSettings, Flags: http2.FlagSettingsAck}

	if verdict := adapter.AdmitFrame(headers); !verdict.Allowed() {
		t.Fatalf("first headers rejected: %+v", verdict)
	}
	if verdict := adapter.AdmitFrame(reset); !verdict.Allowed() {
		t.Fatalf("reset rejected: %+v", verdict)
	}
	if verdict := adapter.AdmitFrame(headers); verdict.Reason != http2guard.ReasonRapidResetBurstExceeded {
		t.Fatalf("third rapid verdict = %+v, want rapid rejection", verdict)
	}
	if verdict := adapter.AdmitFrame(settings); !verdict.Allowed() {
		t.Fatalf("settings rejected by rapid budget: %+v", verdict)
	}
	if verdict := adapter.AdmitFrame(settingsACK); !verdict.Allowed() {
		t.Fatalf("settings ACK rejected: %+v", verdict)
	}
	if verdict := adapter.AdmitFrame(settings); verdict.Reason != http2guard.ReasonSettingsBurstExceeded {
		t.Fatalf("third settings verdict = %+v, want settings rejection", verdict)
	}
}

func TestHTTP2GuardAdapterConcurrentAccounting(t *testing.T) {
	t.Parallel()

	guard, adapter := newGuardAndAdapterForTest(t, http2guard.MaxBurst, http2guard.MaxBurst)
	const workers = 32
	const eventsPerWorker = 100
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)

	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer waitGroup.Done()
			for event := 0; event < eventsPerWorker; event++ {
				streamID := uint32(worker*eventsPerWorker + event + 1)
				adapter.AdmitFrame(http2.FrameHeader{Type: http2.FrameHeaders, StreamID: streamID})
			}
		}(worker)
	}
	waitGroup.Wait()

	if got := guard.Snapshot(time.Now()).RapidEvents; got != workers*eventsPerWorker {
		t.Fatalf("rapid count = %d, want %d", got, workers*eventsPerWorker)
	}
}

func TestHTTP2GuardAdapterInvalidState(t *testing.T) {
	t.Parallel()

	var nilAdapter *devproxy.HTTP2GuardAdapter
	verdict := nilAdapter.AdmitFrame(http2.FrameHeader{Type: http2.FrameHeaders, StreamID: 1})
	if !verdict.Rejected() || verdict.Reason != http2guard.ReasonInvalidState {
		t.Fatalf("nil adapter verdict = %+v, want invalid state", verdict)
	}

	emptyAdapter := devproxy.NewHTTP2GuardAdapter(nil)
	verdict = emptyAdapter.AdmitFrame(http2.FrameHeader{Type: http2.FrameHeaders, StreamID: 1})
	if !verdict.Rejected() || verdict.Reason != http2guard.ReasonInvalidState {
		t.Fatalf("empty adapter verdict = %+v, want invalid state", verdict)
	}
}

func TestHTTP2GuardAdapterHotPathAllocationFree(t *testing.T) {
	// Keep this test serial. testing.AllocsPerRun panics while parallel tests
	// are active, and concurrent accounting is covered by the test above.
	_, adapter := newGuardAndAdapterForTest(t, http2guard.MaxBurst, http2guard.MaxBurst)
	header := http2.FrameHeader{Type: http2.FrameHeaders, StreamID: 1}
	allocations := testing.AllocsPerRun(1000, func() {
		verdict := adapter.AdmitFrame(header)
		adapterAllocationSink.Store(uint64(verdict.Action))
	})
	if allocations != 0 {
		t.Fatalf("AdmitFrame() allocations = %v, want 0", allocations)
	}
}

func FuzzHTTP2GuardAdapterAdmitFrame(f *testing.F) {
	_, adapter := newGuardAndAdapterForTest(f, http2guard.MaxBurst, http2guard.MaxBurst)
	f.Add(uint8(http2.FrameHeaders), uint8(0), uint32(0), uint32(1))
	f.Add(uint8(http2.FrameRSTStream), uint8(0), uint32(4), uint32(1))
	f.Add(uint8(http2.FrameSettings), uint8(http2.FlagSettingsAck), uint32(0), uint32(0))

	f.Fuzz(func(t *testing.T, frameType, flags uint8, length, streamID uint32) {
		header := http2.FrameHeader{
			Type:     http2.FrameType(frameType),
			Flags:    http2.Flags(flags),
			Length:   length,
			StreamID: streamID,
		}
		verdict := adapter.AdmitFrame(header)
		if verdict.Action != http2guard.ActionAllow && verdict.Action != http2guard.ActionReject {
			t.Fatalf("invalid action %d", verdict.Action)
		}
	})
}
