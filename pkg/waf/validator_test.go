package waf

import "testing"

func TestValidateSQLi(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"SELECT * FROM users", false},
		{"admin' OR 1=1 --", true},
		{"normal_query", false},
		{"OR 1=1", true},
		{"-- comment", true},
		{"select * from table where id=1", false},
		{"admin' or 1=1", true},
		{"UNION SELECT", false},
		{"1=1", true},
		{"%20OR%201=1", true},
		{"SELECT name FROM users WHERE id=1; --", true},
		{"# comment line", true},
		{"%2d%2dcomment", true},
	}

	for _, tt := range tests {
		data := tt
		t.Run(data.input, func(t *testing.T) {
			if got := ValidateSQLi(data.input); got != data.want {
				t.Errorf("ValidateSQLi(%q) = %v, want %v", data.input, got, data.want)
			}
		})
	}
}
