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

func startTray(w webview2.WebView) {
	go func() {
		// A panic here would kill the process; the window should keep running without a tray.
		defer func() { recover() }()
		// systray creates a window and pumps its own message loop. Windows
		// message queues are per thread, and an unlocked goroutine can migrate
		// threads mid loop. The main goroutine belongs to the webview.
		runtime.LockOSThread()
		systray.Run(func() { onReady(w) }, nil)
	}()
}

func onReady(w webview2.WebView) {
	systray.SetIcon(trayIcon)
	systray.SetTitle("CLIque")
	systray.SetTooltip(trayTooltip(Version, 0))
	// Not clickable: it is a label. Nothing else shows which version is
	// running, because the window belongs to the panel and reports the
	// panel's version rather than this app's.
	verItem := systray.AddMenuItem(trayTooltip(Version, 0), "The version running now")
	verItem.Disable()
	showItem := systray.AddMenuItem("Show CLIque", "Show CLIque")
	quitItem := systray.AddMenuItem("Quit CLIque", "Quit CLIque")
	go func() {
		for range showItem.ClickedCh {
			w.Dispatch(func() { showWindow(w) })
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
	// Only now is closing the window safe to turn into hiding it. Installed
	// any earlier and a tray that failed to appear would leave a hidden
	// window with nothing left to bring it back or shut it down.
	w.Dispatch(func() { hideOnClose(w) })
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
