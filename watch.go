package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

type Session struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Signal string `json:"signal"`
	Saying string `json:"saying"`
}

// A nil prev means this is the first poll: seed next without alerting.
func newAlerts(prev map[string]string, sessions []Session) (alerts []Session, next map[string]string) {
	next = make(map[string]string, len(sessions))
	first := prev == nil
	for _, s := range sessions {
		next[s.ID] = s.Signal
		if first {
			continue
		}
		if s.Signal != "waiting" && s.Signal != "error" {
			continue
		}
		if prev[s.ID] == s.Signal {
			continue
		}
		alerts = append(alerts, s)
	}
	return alerts, next
}

func needingCount(sessions []Session) int {
	n := 0
	for _, s := range sessions {
		if s.Signal == "waiting" || s.Signal == "error" {
			n++
		}
	}
	return n
}

func Watch(cfg Config) {
	client := &http.Client{Timeout: 10 * time.Second}
	endpoint := strings.TrimRight(cfg.ServerURL, "/") + "/api/state"
	var prev map[string]string
	for {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
			res, err := client.Do(req)
			if err == nil {
				var payload struct {
					Sessions []Session `json:"sessions"`
				}
				ok := res.StatusCode == http.StatusOK && json.NewDecoder(res.Body).Decode(&payload) == nil
				res.Body.Close()
				if ok {
					var alerts []Session
					alerts, prev = newAlerts(prev, payload.Sessions)
					setTrayCount(needingCount(payload.Sessions))
					for _, s := range alerts {
						body := s.Saying
						if body == "" {
							if s.Signal == "error" {
								body = "hit an error"
							} else {
								body = "is waiting for you"
							}
						}
						_ = beeep.Notify("CLIque: "+s.Name, body, "")
					}
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}
