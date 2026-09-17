//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jchv/go-webview2"
)

var (
	pendingMu sync.Mutex
	pending   struct {
		version string
		exeURL  string
		sumURL  string
	}
)

func main() {
	if !claimSingleInstance() {
		return
	}
	cleanupOld()

	cfg, _ := Load()

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath: WebViewDataPath(),
		WindowOptions: webview2.WindowOptions{
			Title:  "CLIque",
			Width:  1280,
			Height: 860,
		},
	})
	if w == nil {
		alert("CLIque", "This needs the Microsoft Edge WebView2 Runtime, which is missing.\n\n"+
			"Install it from https://developer.microsoft.com/microsoft-edge/webview2/ and start CLIque again.")
		return
	}
	defer w.Destroy()

	_ = w.Bind("cliqueUpdatePending", func() string {
		pendingMu.Lock()
		defer pendingMu.Unlock()
		return pending.version
	})
	// Bindings run on the UI thread: capture the URLs and return, then
	// download and swap in a goroutine.
	_ = w.Bind("cliqueRestart", func() string {
		pendingMu.Lock()
		exeURL, sumURL := pending.exeURL, pending.sumURL
		pendingMu.Unlock()
		if exeURL == "" || sumURL == "" {
			return "no update is ready"
		}
		go func() {
			if err := applyUpdate(exeURL, sumURL); err != nil {
				w.Dispatch(func() {
					w.Eval("window.__cliqueUpdateFailed(" + jsString(err.Error()) + ")")
				})
			}
		}()
		return ""
	})
	w.Init(updateJS)

	if cfg.ServerURL == "" {
		// Bindings are invoked on the UI thread, so the probe cannot happen
		// here: a five second timeout against an unreachable host would freeze
		// the window. Validate the URL, hand back, and finish in a goroutine.
		_ = w.Bind("saveServer", func(rawURL, token string) string {
			u, errMsg := parseServerURL(rawURL)
			if errMsg != "" {
				return errMsg
			}
			go func() {
				if err := probeHealthz(u); err != nil {
					w.Dispatch(func() { w.Eval("window.cliqueFailed(" + jsString(err.Error()) + ")") })
					return
				}
				next := Config{ServerURL: u, Token: strings.TrimSpace(token)}
				if err := Save(next); err != nil {
					w.Dispatch(func() { w.Eval("window.cliqueFailed(" + jsString(err.Error()) + ")") })
					return
				}
				if next.Token != "" {
					go Watch(next)
				}
				w.Dispatch(func() { w.Navigate(next.ServerURL) })
			}()
			return ""
		})
		w.SetHtml(setupHTML)
	} else {
		w.Navigate(cfg.ServerURL)
		if cfg.Token != "" {
			go Watch(cfg)
		}
	}
	go pollUpdates(w)
	w.Run()
}

func pollUpdates(w webview2.WebView) {
	check := func() {
		ver, exeURL, sumURL, ok := newerRelease(Version)
		if !ok {
			return
		}
		pendingMu.Lock()
		pending.version = ver
		pending.exeURL = exeURL
		pending.sumURL = sumURL
		pendingMu.Unlock()
		w.Dispatch(func() {
			w.Eval("window.__cliqueUpdate(" + jsString(ver) + ")")
		})
	}
	check()
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for range t.C {
		check()
	}
}

// jsString renders a Go string as a JavaScript literal safe to paste into Eval.
func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func parseServerURL(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", "enter an http or https URL"
	}
	return raw, ""
}

func probeHealthz(serverURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(strings.TrimRight(serverURL, "/") + "/healthz")
	if err != nil {
		return fmt.Errorf("cannot reach the panel: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("panel health check returned %s", res.Status)
	}
	return nil
}
