package main

import "testing"

func TestCaptionIsDark(t *testing.T) {
	tests := []struct {
		rgb  int
		dark bool
	}{
		{0x000000, true},
		{0xFFFFFF, false},
		{0x252526, true},
		{0xF5F5F5, false},
		{0x2D7D46, true},
	}
	for _, tt := range tests {
		if got := captionIsDark(tt.rgb); got != tt.dark {
			t.Errorf("captionIsDark(0x%06X) = %v, want %v", tt.rgb, got, tt.dark)
		}
	}
}
