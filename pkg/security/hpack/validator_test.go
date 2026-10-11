package hpack

import (
	"errors"
	"testing"
)

func TestValidator(t *testing.T) {
	tests := []struct {
		name    string
		fields  [][2]string
		wantErr error
	}{
		{
			name:    "ValidRequest",
			fields:  [][2]string{{":method", "GET"}, {":path", "/"}, {"host", "example.com"}},
			wantErr: nil,
		},
		{
			name:    "InvalidOrder",
			fields:  [][2]string{{"host", "example.com"}, {":method", "GET"}},
			wantErr: ErrPseudoHeaderAfterRegular,
		},
		{
			name:    "DuplicatePseudo",
			fields:  [][2]string{{":method", "GET"}, {":method", "POST"}},
			wantErr: ErrDuplicatePseudoHeader,
		},
		{
			name:    "EmptyHeaderName",
			fields:  [][2]string{{"", "value"}},
			wantErr: ErrEmptyHeaderName,
		},
		{
			name:    "UppercaseHeaderName",
			fields:  [][2]string{{"Content-Type", "application/json"}},
			wantErr: ErrUppercaseHeaderName,
		},
		{
			name:    "UnknownPseudoHeader",
			fields:  [][2]string{{":unknown", "value"}},
			wantErr: ErrUnknownPseudoHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewState()
			val := NewValidator()
			var err error
			for _, f := range tt.fields {
				err = val.ValidateField(state, f[0], f[1])
				if err != nil {
					break
				}
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateField() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("ValidateField() unexpected error = %v", err)
			}
		})
	}
}
