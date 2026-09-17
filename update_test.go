package main

import "testing"

func TestIsNewer(t *testing.T) {
	tests := []struct {
		candidate, running string
		want               bool
	}{
		{"0.2.0", "0.1.0", true},
		{"0.10.0", "0.9.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"1.2.3-4", "1.2.3", true},
		{"1.2.3", "1.2.3-4", false},
		{"1.2.3-4", "1.2.3-4", false},
		{"1.x.3", "1.0.2", true},
		{"1.x.3", "1.0.3", false},
		{"nightly", "0.1.0", false},
	}
	for _, tt := range tests {
		got := isNewer(tt.candidate, tt.running)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.candidate, tt.running, got, tt.want)
		}
	}
}

func TestIsNewerGarbageDoesNotPanic(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("isNewer panicked on garbage: %v", rec)
		}
	}()
	_ = isNewer("1.x.3", "0.1.0")
	_ = isNewer("??", "not-a-version")
	_ = isNewer("", "")
}
