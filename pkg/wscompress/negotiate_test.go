package wscompress

import (
	"testing"
)

func TestNegotiateOffer(t *testing.T) {
	hdr := "permessage-deflate; server_no_context_takeover; client_max_window_bits=12"
	found, params, err := ParseOffer(hdr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected permessage-deflate to be found")
	}
	if !params.ServerNoContextTakeover {
		t.Fatal("expected server_no_context_takeover to be true")
	}
	if params.ClientMaxWindowBits != 12 {
		t.Fatalf("expected client_max_window_bits=12, got %d", params.ClientMaxWindowBits)
	}

	res := FormatResponse(params)
	if res == "" {
		t.Fatal("expected non-empty formatted response")
	}
}
