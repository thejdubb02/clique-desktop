package main

import "testing"

func TestNewAlerts(t *testing.T) {
	waiting := Session{ID: "a", Name: "alpha", Signal: "waiting", Saying: "need you"}

	t.Run("first poll seeds without alerting", func(t *testing.T) {
		alerts, next := newAlerts(nil, []Session{waiting, {ID: "b", Name: "beta", Signal: "error"}})
		if len(alerts) != 0 {
			t.Fatalf("alerts = %v, want none", alerts)
		}
		if next["a"] != "waiting" || next["b"] != "error" {
			t.Fatalf("next = %v", next)
		}
	})

	t.Run("empty to waiting alerts", func(t *testing.T) {
		prev := map[string]string{"a": ""}
		alerts, next := newAlerts(prev, []Session{waiting})
		if len(alerts) != 1 || alerts[0].ID != "a" || alerts[0].Signal != "waiting" {
			t.Fatalf("alerts = %v", alerts)
		}
		if next["a"] != "waiting" {
			t.Fatalf("next = %v", next)
		}
	})

	t.Run("unchanged waiting does not alert", func(t *testing.T) {
		prev := map[string]string{"a": "waiting"}
		alerts, next := newAlerts(prev, []Session{waiting})
		if len(alerts) != 0 {
			t.Fatalf("alerts = %v, want none", alerts)
		}
		if next["a"] != "waiting" {
			t.Fatalf("next = %v", next)
		}
	})

	t.Run("waiting to empty to waiting alerts again", func(t *testing.T) {
		prev := map[string]string{"a": "waiting"}
		alerts, next := newAlerts(prev, []Session{{ID: "a", Name: "alpha", Signal: ""}})
		if len(alerts) != 0 {
			t.Fatalf("cleared waiting: alerts = %v, want none", alerts)
		}
		if next["a"] != "" {
			t.Fatalf("next after clear = %v", next)
		}
		alerts, next = newAlerts(next, []Session{waiting})
		if len(alerts) != 1 || alerts[0].Signal != "waiting" {
			t.Fatalf("second waiting: alerts = %v", alerts)
		}
		if next["a"] != "waiting" {
			t.Fatalf("next = %v", next)
		}
	})

	t.Run("vanished session is dropped", func(t *testing.T) {
		prev := map[string]string{"a": "waiting", "b": "error"}
		alerts, next := newAlerts(prev, []Session{waiting})
		if len(alerts) != 0 {
			t.Fatalf("alerts = %v, want none", alerts)
		}
		if _, ok := next["b"]; ok {
			t.Fatalf("vanished id still in next: %v", next)
		}
		if next["a"] != "waiting" {
			t.Fatalf("next = %v", next)
		}
	})
}

func TestNeedingCountWatch(t *testing.T) {
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
	}
	if n := needingCount(mix); n != 2 {
		t.Fatalf("mix = %d, want 2", n)
	}
}
