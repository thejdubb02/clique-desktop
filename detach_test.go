package main

import "testing"

func TestDetachFlagOrder(t *testing.T) {
	order := detachFlagOrder()
	if len(order) != 2 {
		t.Fatalf("want a first choice and a fallback, got %d", len(order))
	}
	// Losing this is how the helper gets killed with the app it is replacing.
	if order[0]&createBreakawayFromJob == 0 {
		t.Error("the first attempt does not try to leave the app's job")
	}
	// And losing this is how a job that forbids breakaway ends with no helper
	// started at all.
	if order[1]&createBreakawayFromJob != 0 {
		t.Error("the fallback still asks to break away, so there is no fallback")
	}
	for i, flags := range order {
		if flags&detachedProcess == 0 {
			t.Errorf("attempt %d would keep the app's console", i)
		}
	}
}
