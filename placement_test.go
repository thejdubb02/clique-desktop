package main

import "testing"

var oneScreen = []WindowBox{{X: 0, Y: 0, W: 1920, H: 1080}}

func TestBoxOnScreen(t *testing.T) {
	cases := []struct {
		name    string
		box     WindowBox
		screens []WindowBox
		want    bool
	}{
		{"where it was left", WindowBox{X: 100, Y: 100, W: 1280, H: 860}, oneScreen, true},
		{"hanging off the right edge but still grabbable",
			WindowBox{X: 1700, Y: 100, W: 1280, H: 860}, oneScreen, true},
		// The one that matters: a second monitor that is no longer there.
		{"on a screen that is gone", WindowBox{X: 2600, Y: 200, W: 1280, H: 860}, oneScreen, false},
		{"above the top of everything", WindowBox{X: 100, Y: -2000, W: 1280, H: 860}, oneScreen, false},
		{"a sliver on screen is not enough",
			WindowBox{X: 1900, Y: 100, W: 1280, H: 860}, oneScreen, false},
		{"too small to be a window", WindowBox{X: 10, Y: 10, W: 80, H: 40}, oneScreen, false},
		{"nothing saved", WindowBox{}, oneScreen, false},
		{"no screens at all", WindowBox{X: 0, Y: 0, W: 1280, H: 860}, nil, false},
		{"the second monitor, still plugged in",
			WindowBox{X: 2600, Y: 200, W: 1280, H: 860},
			[]WindowBox{{X: 0, Y: 0, W: 1920, H: 1080}, {X: 1920, Y: 0, W: 1920, H: 1080}}, true},
	}
	for _, c := range cases {
		if got := boxOnScreen(c.box, c.screens); got != c.want {
			t.Errorf("%s: boxOnScreen = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWindowBoxEmpty(t *testing.T) {
	if !(WindowBox{}).Empty() {
		t.Error("a zero box should be empty")
	}
	if (WindowBox{Maximized: true}).Empty() {
		t.Error("maximized on its own is still something to restore")
	}
	if (WindowBox{W: 1280, H: 860}).Empty() {
		t.Error("a size is something to restore")
	}
}
