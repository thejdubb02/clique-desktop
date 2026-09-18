# clique-desktop — standing rules

A dedicated window around the CLIque web panel, for Windows. The panel runs on
a Linux box; this is a viewer of it. Go, no cgo, no framework, no build step
beyond `go build`.

## What it is not

It does not run tmux, talk to tmux, or know that tmux exists. It does not parse
panes, know which AI vendor is running, or interpret what a model is doing.
Those are the panel's job, or nobody's. Same three rules as `clique` itself:
filesystem/tmux/process state only, a driver not an IDE, clean room on Codeman.

**The panel's web UI is the UI.** The only HTML this repo owns is the first-run
setup page. If a feature would mean rendering a session list, a terminal, or a
settings screen here, it belongs in the panel instead, where every client gets
it at once.

## The build constraint that shapes everything

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w"
```

This is why the app exists in the shape it does: it cross-compiles from the
Linux devbox to a single Windows `.exe` in about a second, so there is no CI
round-trip between a change and something testable. **Never add a dependency
that needs cgo.** That would move every build to a Windows runner and turn a
one-second loop into a five-minute one. `scripts/build.sh` is the command.

`-H windowsgui` is what stops a console window appearing behind the app.

## Traps, each of which has already bitten

- **Bindings run on the UI thread.** `w.Bind` callbacks are invoked
  synchronously from the WebView2 message handler, so any I/O inside one
  freezes the window for its full duration. Validate cheaply, return, and
  finish in a goroutine, calling back with `w.Dispatch(...)` plus `w.Eval`.
  The first-run probe was written the blocking way and hung the window for
  five seconds on a typo.
- **Point at the tailnet URL, never the public host.**
  `https://wsg-devbox.tail7de7a3.ts.net/clique/` is served by Tailscale serve
  straight to the panel on loopback: real cert, tailnet only, and nothing in
  front of it. `clique.willhitestrategy.org` gates `/` behind Tinyauth
  (password + TOTP) and leaves only `/api/` and `/ws` open, which is correct
  for machines and impossible for a webview that has to render the UI.
- **A missing WebView2 Runtime returns a nil webview.** Say so in a message
  box. Exiting silently is indistinguishable from a corrupt download.
- **The token is not the login.** The webview does the panel's own password
  login and keeps the cookie in its own data directory. The token in
  `config.json` is only for the notification poller, which reads
  `GET /api/state` with `Authorization: Bearer`. The app works without one,
  minus toasts.

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

## Two ways it is distributed, and they update differently

Hydraulic Conveyor builds the Windows package from this Linux box, the same as
Rampart, and reads the same root signing key from
`/etc/hydraulic/conveyor/defaults.conf`. `scripts/package.sh <version>` does the
whole thing: it builds the exe and the package from one version argument, because
a manifest that disagrees with the binary inside it updates on a schedule nobody
can explain.

- **The MSIX** is what a person installs. Windows keeps it current in the
  background and the install directory is read only, so `runningPackaged()`
  turns the in-app updater off there. Forgetting that would put a card in front
  of someone offering an update that cannot be applied.
- **The loose `CLIque.exe`** stays published for anyone who does not want to
  install anything, and keeps the self-updater. It is the same binary.

The icon and the file properties come from `resource_windows.syso`, generated by
`goversioninfo` inside `scripts/build.sh` and not committed. Without it the
taskbar shows the generic application icon and the SmartScreen warning has no
name to put in the dialog. go-webview2 finds it because `goversioninfo` also
registers the icon group under 32512, which is the id its default path loads.

## Updating

Modelled on Rampart, deliberately, because it is the behaviour Justin already
knows: the app says a new version exists and then waits. Nothing installs until
someone presses the button.

- The check runs **after** the window is up, never before it. A cold start must
  not wait on the network. Then every 30 minutes.
- **Every failure is silent.** No network, a 404, a malformed release: the
  answer is no update, not an error in someone's face.
- A build whose `Version` is still `dev` never offers an update, so running from
  source is never interrupted.
- A dismissed card coming back at the next check is correct, not a bug. Rampart
  does the same.
- The restart is safe to offer casually, and the card says so, because **the
  sessions are not in this process.** They are tmux on the panel's machine.
  Closing this window costs nothing, which is the whole reason an updater like
  this is appropriate here and would not be in an app that held state.

Two traps in the implementation:

- **Release the single-instance mutex before launching the new binary**, or the
  new process sees the old one still holding it and exits immediately, leaving
  nothing running at all.
- Windows will not let you overwrite a running `.exe`, but it will let you
  **rename** one. So: rename the current binary to `.old`, move the new one into
  place, launch, exit, and delete the `.old` on the next start, which is the
  only moment it is not in use.

The downloaded binary is checked against a `CLIque.exe.sha256` published beside
it in the same release. Be honest about what that is worth: it catches a
truncated or corrupt download. It is **not** a defence against a compromised
release, because it comes from the same place. Code signing is what that would
take, and we do not have it yet. Asset URLs are required to be on `github.com`
for the same reason: the app downloads something and then runs it, so where it
downloads from is a trust boundary.

## State

`%AppData%\CLIque\config.json` holds the server URL and the optional token.
`%AppData%\CLIque\webview\` is the WebView2 data directory, which is what makes
the panel login survive a restart and a browser profile wipe.

Nothing here is a secret we own, so nothing here is committed. The token is
typed in at runtime by whoever installs it.

## Checks

`go test ./...` runs on Linux and must stay that way, which is why the
Windows-only pieces sit behind `//go:build windows` with a no-op sibling. The
watcher's transition logic is a pure function precisely so it can be tested
without a network or a Windows box.

The test has to bite. Break the guard in `newAlerts` and confirm the test goes
red before trusting a green run.
