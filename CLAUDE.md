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

## Distribution, and how signing works

One binary, two ways to get it, same updater either way — `runningPackaged()`
and the MSIX/Conveyor branch it used to feed are gone (2026-09-20). Conveyor is
a paid tool; NSIS (`makensis`) and `osslsigncode` are free, both run natively on
Linux, and between them they do everything Conveyor did that still mattered
once the app had its own updater (below).

- **`CLIque-Setup.exe`** — an NSIS installer (`installer.nsi`), per-user
  install (`$LOCALAPPDATA`), so no admin prompt. Start Menu entry, uninstaller,
  Add/Remove Programs entry.
- **The loose `CLIque.exe`** — the same binary the installer places, published
  separately for anyone who would rather not install anything.

Both are Authenticode-signed with a self-signed cert/key generated once with
openssl (no CA, so no renewal to track). It lives at
`/etc/clique-desktop/signing/{codesign.pem,codesign.key}` on the devbox, mode
600, not in this repo, backed up in Vaultwarden as "CLIque Desktop - code
signing root key" — same treatment as Rampart's signing key, and for the same
reason: **losing it means a new signer identity and every installed copy stops
trusting updates until reinstalled by hand.**

**A self-signed cert does not clear Windows SmartScreen.** That is a separate
cloud reputation check on the download itself, not a signature check, and only
real download volume or a paid EV certificate clears it. First install shows
"Windows protected your PC"; **More info → Run anyway** clears it, once. It
does not come back on an update, because updates never go through that
download-and-run path — see below.

`scripts/package.sh <version>` builds, signs, packages and signs again:
`build.sh` → sign `CLIque.exe` → `makensis` builds the installer around the
signed exe → sign the installer too → recompute `CLIque.exe.sha256` from the
final signed bytes, since the updater's checksum has to match what actually
shipped.

The icon and the file properties come from `resource_windows.syso`, generated by
`goversioninfo` inside `scripts/build.sh` and not committed. Without it the
taskbar shows the generic application icon and the SmartScreen warning has no
name to put in the dialog. go-webview2 finds it because `goversioninfo` also
registers the icon group under 32512, which is the id its default path loads.

## Updating

The app owns its own updater end to end now (`update.go`); there is no
Windows-managed background path to defer to or race with, which is what made
the MSIX-era failures (four in one evening, 2026-09-18: an unconditional
`SW_RESTORE`, a mutex with no window behind it, a family name read too late, a
helper killed with the app replacing it) possible in the first place. Worth
remembering if anyone is ever tempted to have the app replace an installed copy
of itself from the inside: it went wrong four different ways and every one of
them ended with nothing running.

- **Nothing at startup.** The first check is a minute after the window is up,
  then every 30 minutes. A cold start never waits on the network.
- **Staged silently, in the background.** A newer release is downloaded and
  checksum-verified before anything appears on screen — the card only shows
  once a restart would be an instant rename, not a wait.
- **A sidecar file carries the verified digest**, so a *different* process (the
  next cold start, not the one that downloaded it) can trust the staged file
  without re-fetching it. Never trust a staged file on presence alone; a crash
  mid-download or a pulled release both leave one sitting there unverified.
- **Restarting installs it, click or no click.** `applyPendingUpdateIfStaged`
  runs before the single-instance mutex is claimed, so closing CLIque and
  reopening it applies whatever was already staged, on its own. The button
  (`applyStagedOrDownload`) just makes it happen sooner, or downloads on the
  spot if nothing was staged yet.
- **Every failure is silent.** No network, a 404, a malformed release: the
  answer is no update, not an error in someone's face.
- A build whose `Version` is still `dev` never offers an update, so running from
  source is never interrupted.
- **Release the single-instance mutex before launching the new binary**, or the
  new process sees the old one still holding it and exits immediately.
- Windows will not let you overwrite a running `.exe`, but it will let you
  **rename** one: rename the current binary to `.old`, move the staged one into
  place, launch, exit. `.old` is removed on the next start, the only moment it
  is not in use.
- The download is checked against `CLIque.exe.sha256`, and now that it is
  signed too, that checksum is a genuine defence against a compromised release,
  not just a truncated one. Asset URLs are still required to be on
  `github.com`: the app downloads something and then runs it, so where it
  downloads from is a trust boundary regardless of signing.
- The restart is safe to offer casually, and the card says so, because **the
  sessions are not in this process.** They are tmux on the panel's machine.
  Closing this window costs nothing, which is the whole reason an updater like
  this is appropriate here and would not be in an app that held state.

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
