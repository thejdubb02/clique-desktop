package main

import "testing"

func TestTrayTooltip(t *testing.T) {
	tests := []struct {
		version string
		waiting int
		want    string
	}{
		{"0.3.8", 0, "CLIque 0.3.8"},
		{"0.3.8", 1, "CLIque 0.3.8 - 1 session needs you"},
		{"0.3.8", 2, "CLIque 0.3.8 - 2 sessions need you"},
		{"0.3.8", 5, "CLIque 0.3.8 - 5 sessions need you"},
		// A build from source has no version, and "CLIque dev" would be a
		// worse answer than the name on its own.
		{"", 0, "CLIque"},
		{"dev", 0, "CLIque"},
		{"dev", 2, "CLIque - 2 sessions need you"},
		{"", 3, "CLIque - 3 sessions need you"},
	}
	for _, tt := range tests {
		got := trayTooltip(tt.version, tt.waiting)
		if got != tt.want {
			t.Errorf("trayTooltip(%q, %d) = %q, want %q", tt.version, tt.waiting, got, tt.want)
		}
	}
}

func TestNeedingCount(t *testing.T) {
	if n := needingCount(nil); n != 0 {
		t.Fatalf("nil = %d, want 0", n)
	}
	if n := needingCount([]Session{}); n != 0 {
		t.Fatalf("empty slice = %d, want 0", n)
	}
	if n := needingCount([]Session{{Signal: "waiting"}}); n != 1 {
		t.Fatalf("waiting = %d, want 1", n)
	}
	if n := needingCount([]Session{{Signal: "error"}}); n != 1 {
		t.Fatalf("error = %d, want 1", n)
	}
	if n := needingCount([]Session{{Signal: ""}}); n != 0 {
		t.Fatalf("empty signal = %d, want 0", n)
	}
	mix := []Session{
		{Signal: "waiting"},
		{Signal: "error"},
		{Signal: ""},
		{Signal: "ok"},
		{Signal: "waiting"},
	}
	if n := needingCount(mix); n != 3 {
		t.Fatalf("mix = %d, want 3", n)
	}
}
