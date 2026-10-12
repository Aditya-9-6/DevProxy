package router

import "testing"

func TestRadixRouter(t *testing.T) {
	r := NewRadixRouter()

	// Test basic registration and lookup
	r.Add("/api/v1/users", "users_handler")
	r.Add("/api/v1/posts", "posts_handler")
	r.Add("/", "root_handler")

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{"Exact match users", "/api/v1/users", "users_handler"},
		{"Exact match posts", "/api/v1/posts", "posts_handler"},
		{"Root match", "/", "root_handler"},
		{"Empty path match", "", "root_handler"},
		{"Trailing slash", "/api/v1/users/", "users_handler"},
		{"Multiple slashes", "///api///v1///users///", "users_handler"},
		{"No leading slash", "api/v1/users", "users_handler"},
		{"Non-existent path", "/api/v2", nil},
		{"Partial match", "/api/v1", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Lookup(tt.path)
			if got != tt.expected {
				t.Errorf("Lookup(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestRadixRouter_Concurrency(t *testing.T) {
	r := NewRadixRouter()
	const workers = 10
	const iterations = 100

	for i := 0; i < workers; i++ {
		go func(id int) {
			r.Add("/path/"+string(rune(id)), id)
		}(i)
	}

	for i := 0; i < workers; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				_ = r.Lookup("/path/" + string(rune(id)))
			}
		}(i)
	}
}
