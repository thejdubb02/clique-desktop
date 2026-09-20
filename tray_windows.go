//go:build windows

package main

import (
	_ "embed"
	"runtime"
	"sync/atomic"

	"github.com/getlantern/systray"
	"github.com/jchv/go-webview2"
)

//go:embed tray.ico
var trayIcon []byte

var (
	trayReady atomic.Bool
	quitting  atomic.Bool
)

func startTray(w webview2.WebView, serverURL string) {
	go func() {
		// A panic here would kill the process; the window should keep running without a tray.
		defer func() { recover() }()
		// systray creates a window and pumps its own message loop. Windows
		// message queues are per thread, and an unlocked goroutine can migrate
		// threads mid loop. The main goroutine belongs to the webview.
		runtime.LockOSThread()
		systray.Run(func() { onReady(w, serverURL) }, nil)
	}()
}

func onReady(w webview2.WebView, serverURL string) {
	systray.SetIcon(trayIcon)
	systray.SetTitle("CLIque")
	systray.SetTooltip(trayTooltip(Version, 0))
	// Not clickable: it is a label. Nothing else shows which version is
	// running, because the window belongs to the panel and reports the
	// panel's version rather than this app's.
	verItem := systray.AddMenuItem(trayTooltip(Version, 0), "The version running now")
	verItem.Disable()
	showItem := systray.AddMenuItem("Show CLIque", "Show CLIque")
	// The webview has no address bar and no F5, so a page stuck in a bad
	// state (a dead socket, a pane that stopped switching) had no way back
	// short of quitting the whole app. Sessions live in tmux on the panel's
	// machine, not this process, so a reload costs nothing.
	reloadItem := systray.AddMenuItem("Reload CLIque", "Reload the panel")
	quitItem := systray.AddMenuItem("Quit CLIque", "Quit CLIque")
	go func() {
		for range showItem.ClickedCh {
			w.Dispatch(func() { showWindow(w) })
		}
	}()
	go func() {
		for range reloadItem.ClickedCh {
			w.Dispatch(func() {
				showWindow(w)
				if serverURL != "" {
					w.Navigate(serverURL)
				}
			})
		}
	}()
	go func() {
		select {
		case <-quitItem.ClickedCh:
			quitting.Store(true)
			systray.Quit()
			w.Dispatch(func() { w.Terminate() })
		}
	}()
	trayReady.Store(true)
}

func setTrayCount(waiting int) {
	if !trayReady.Load() {
		return
	}
	systray.SetTooltip(trayTooltip(Version, waiting))
}

func trayQuitting() bool {
	return quitting.Load()
}
