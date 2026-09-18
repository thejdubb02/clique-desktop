package main

// WindowBox is where the window was last left. Persisted in config.json.
type WindowBox struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	W         int  `json:"w"`
	H         int  `json:"h"`
	Maximized bool `json:"maximized"`
}

// Empty reports a box that was never saved, so there is nothing to restore.
func (b WindowBox) Empty() bool { return b.W == 0 && b.H == 0 && !b.Maximized }

/* Whether a saved box is worth putting a window back into.
 *
 * A screen that is no longer plugged in leaves coordinates nobody can reach,
 * and a window restored onto one looks exactly like an app that did not start.
 * This app has had enough of those. A box counts as reachable when a decent
 * piece of it, not a sliver, overlaps a screen that exists now.
 *
 * Pure, and the screens are passed in, so it can be tested without a display. */
func boxOnScreen(b WindowBox, screens []WindowBox) bool {
	if b.W < 200 || b.H < 150 {
		return false
	}
	for _, s := range screens {
		over := overlap(b.X, b.X+b.W, s.X, s.X+s.W)
		down := overlap(b.Y, b.Y+b.H, s.Y, s.Y+s.H)
		// Enough to grab the title bar with, rather than one visible pixel.
		if over >= 120 && down >= 40 {
			return true
		}
	}
	return false
}

func overlap(a0, a1, b0, b1 int) int {
	lo, hi := a0, a1
	if b0 > lo {
		lo = b0
	}
	if b1 < hi {
		hi = b1
	}
	return hi - lo
}
