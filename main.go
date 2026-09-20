//go:build windows

package main

import (
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

	// Before the window exists, on the thread the edge package already locked.
	enableFileDrops()

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath: WebViewDataPath(),
		WindowOptions: webview2.WindowOptions{
			Title:  "CLIque",
			Width:  1280,
			Height: 860,
			// The icon the window, the taskbar button and alt-tab all use,
			// by resource id. Leaving this at zero does not fall back to the
			// embedded icon: go-webview2's default branch calls LoadImageW
			// with the system icon width where the image type belongs, which
			// fails, and the window ends up with no icon at all.
			IconId: 1,
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
		version, exeURL, sumURL := pending.version, pending.exeURL, pending.sumURL
		pendingMu.Unlock()
		if version == "" {
			return "no update is ready"
		}
		// A packaged install replaces itself through the manifest Windows
		// already knows about. A loose exe downloads the published binary,
		// checks it against its own checksum and swaps itself out. Same
		// button, and from here the only difference is which one can run.
		packaged := runningPackaged()
		if !packaged && (exeURL == "" || sumURL == "") {
			return "this release has no download for it"
		}
		go func() {
			var err error
			if packaged {
				// Only a restart. Windows installs the package itself.
				if err = restartPackaged(); err == nil {
					w.Dispatch(func() { w.Terminate() })
					return
				}
			} else {
				err = applyUpdate(exeURL, sumURL)
			}
			if err != nil {
				w.Dispatch(func() {
					w.Eval("window.__cliqueUpdateFailed(" + jsString(err.Error()) + ")")
				})
			}
		}()
		return ""
	})
	// Bindings run on the UI thread and launching a browser is not instant,
	// so this hands off and returns.
	_ = w.Bind("cliqueOpenExternal", func(raw string) string {
		go openExternal(raw)
		return ""
	})
	// Caption colour is a cheap DWM call, so it stays on the UI thread.
	_ = w.Bind("cliqueCaption", func(panel, fg int) {
		setCaption(w, panel, fg)
	})
	w.Init(shellInitJS(Version))
	w.Init(updateJS)
	w.Init(externalJS)
	w.Init(captionJS)

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
	// Both kinds of install check, because Windows' own background task runs
	// on a schedule nobody can predict and will happily leave a running app a
	// version behind for a day. The card is what makes an update something you
	// can take now; the background task remains the answer for anyone who
	// ignores it.
	go pollUpdates(w)
	installWindowHooks(w)
	restoreBox(w, cfg.Window)
	startTray(w, cfg.ServerURL)
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
		packaged := runningPackaged()
		w.Dispatch(func() {
			w.Eval("window.__cliqueUpdate(" + jsString(ver) + ", " +
				map[bool]string{true: "true", false: "false"}[packaged] + ")")
		})
	}
	// Nothing at startup. Opening the app opens the version that is installed
	// and does not wait on, or act on, anything to do with an update. The first
	// check is a minute in, by which point the window has been up long enough
	// that a card reads as news rather than as part of starting.
	time.Sleep(time.Minute)
	check()
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for range t.C {
		check()
	}
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
