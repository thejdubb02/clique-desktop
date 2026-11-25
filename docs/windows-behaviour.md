# Window behaviour: raising, links and the tray

Moved out of CLAUDE.md (issue #1). CLAUDE.md keeps the one-line rules; this is the full reasoning.

## Bringing the window forward must not resize it

`SW_RESTORE` un-maximizes a maximized window. Calling it unconditionally when
raising the window meant anything that brought CLIque forward shrank a full
screen window back to 1280x860, which reads as the app resizing itself for no
reason. Check `IsIconic` first: only a minimized window needs restoring, and
`SW_SHOW` brings back anything else, including a window hidden to the tray while
maximized, at the size it had.

Two things reach that path, and both are easy to trigger without meaning to: the
tray's Show item, and a second launch finding the mutex already held. Clicking a
toast the watcher raised launches the app too, so that counts as a second launch
as well.

## Links go to the person's browser, and JavaScript is how

WebView2 raises `NewWindowRequested` for both ways the panel opens a link, a
`window.open` and an anchor with `target="_blank"`. go-webview2 does not surface
that event, so before `external.js` a click on a link did nothing at all: no
window, no error, nothing to notice. The fix is injected JavaScript that hands
the URL to a binding, not a change to the library.

**Only `http` and `https` reach `ShellExecute`.** It will launch a file path, a
UNC path or an application protocol just as happily as a web page, and the page
asking is one the panel rendered out of somebody's working directory. Note that
rejecting an empty host is not the same check: `file://attacker/share/x.exe`
has a host. `external_test.go` covers both separately for that reason.

## The tray, and the way it can trap someone

`getlantern/systray` is cgo-free on Windows, which is the only reason it is here
at all. Three things about it are not obvious:

- **Its message loop needs its own locked thread.** `systray.Run` creates a
  window and pumps messages, Windows message queues are per thread, and an
  unlocked goroutine can migrate threads mid loop. So: a goroutine that calls
  `runtime.LockOSThread()` first. It cannot share the main goroutine, which
  belongs to the webview.
- **Close-to-tray is installed from `onReady`, never before it.** The window
  procedure is subclassed so WM_CLOSE hides instead of destroying. Install that
  while the tray is still only hoped for and a tray that never appears leaves a
  hidden window with nothing to restore it and no way to quit short of Task
  Manager. The tray has to exist before hiding is a safe thing to do.
- **`GWLP_WNDPROC` is -4, and a negative constant cannot be converted to
  `uintptr`.** It goes through a variable, where the conversion sign extends.
  As a constant it is a compile error that `go vet` reports and `go build`
  does not.

Everything about the tray fails silently. No tray means an ordinary window that
quits when closed, which is what the app did before and is never worth an error
box.

