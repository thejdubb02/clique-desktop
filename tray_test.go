package main

import "testing"

func TestTrayTooltip(t *testing.T) {
	tests := []struct {
		waiting int
		want    string
	}{
		{0, "CLIque"},
		{1, "CLIque - 1 session needs you"},
		{2, "CLIque - 2 sessions need you"},
		{5, "CLIque - 5 sessions need you"},
	}
	for _, tt := range tests {
		got := trayTooltip(tt.waiting)
		if got != tt.want {
			t.Errorf("trayTooltip(%d) = %q, want %q", tt.waiting, got, tt.want)
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
