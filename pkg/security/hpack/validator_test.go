package hpack

import "testing"

func TestValidator(t *testing.T) {
	tests := []struct {
		name    string
		fields  [][2]string
		wantErr bool
	}{
		{"Valid", [][2]string{{":method", "GET"}, {"host", "example.com"}}, false},
		{"InvalidOrder", [][2]string{{"host", "example.com"}, {":method", "GET"}}, true},
		{"DuplicatePseudo", [][2]string{{":method", "GET"}, {":method", "POST"}}, true},
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
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
