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
tray's Show item, and a second launch finding the mutex already held. A packaged
install can be launched by the Windows notification platform, so a toast the
watcher raised is enough to count as a second launch.

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
  background, and since 2026-09-18 the app also offers it: `runningPackaged()`
  now picks which mechanism the button uses rather than switching the check off.
  The install directory is read only, so a packaged copy cannot swap its own
  exe; it hands the manifest to `Add-AppxPackage` instead and Windows does the
  replacing. Same card, same button, different machinery underneath.
- **The loose `CLIque.exe`** stays published for anyone who does not want to
  install anything, and downloads its replacement directly. It is the same
  binary.

The icon and the file properties come from `resource_windows.syso`, generated by
`goversioninfo` inside `scripts/build.sh` and not committed. Without it the
taskbar shows the generic application icon and the SmartScreen warning has no
name to put in the dialog. go-webview2 finds it because `goversioninfo` also
registers the icon group under 32512, which is the id its default path loads.

## The packaged build does not install its own updates

Decided 2026-09-18, after four separate failures in one evening, all ending with
the app shut down and nothing running: an unconditional `SW_RESTORE`, a mutex
with no window behind it, a family name read too late, a helper killed with the
app it was replacing, and then the packaging tool's own entry point. Windows
already keeps an MSIX current through the manifest's background task. The button
now does only the part Windows does not: get you onto the version that is
installed.

If Windows has not staged the new one yet, restarting lands on the same version
and the card comes back. A wasted click is a better failure than no application.

**Do not put `Add-AppxPackage` back in this app.** `TestPackagedRestartCommand`
fails if anyone does. The loose exe keeps its own updater, which has never been
the problem.

## Starting must never depend on an update, or on another copy

**The fourth one was not our code at all.** Conveyor makes `updatecheck.exe` the
package's entry point by default, as an escape hatch for an update that needs a
full reinstall. For a JVM app it knows what to launch afterwards. For a plain
native binary it does not, so opening the installed app ran the update check,
showed a "checking for updates" window, found nothing to do and exited without
ever starting CLIque. `windows.manifests.msix.use-update-escape-hatch = false`
in `conveyor.conf` makes our own exe the entry point. Check it after any Conveyor
upgrade: `unzip -p output/*.msix AppxManifest.xml | grep Executable=` must say
`CLIque.exe`.


Three ways this app came to not open at all. All three end identically, with
the app shut down and nothing running, and all three are easy to reintroduce.

- **A held mutex with no window behind it used to make a second launch exit.**
  That is a process that died badly, or one still dying after an update
  restart, and standing down for it leaves nothing running. Step aside only for
  a copy that is actually on screen; otherwise open a window. Two windows beat
  none.
- **The packaged restart read the package family name after replacing the
  package.** `Get-AppxPackage` comes back empty in that moment, the launch path
  built from it goes nowhere, and `ForceTargetApplicationShutdown` has already
  closed the app. Read it first, and guard the relaunch on having one.
- **The helper doing the replacing was a child of the app being replaced.**
  `ForceTargetApplicationShutdown` terminates the whole app container, and a
  process this app started is inside it, so the helper was killed part way
  through its own work. `startDetached` breaks it out of the job first and falls
  back to a plain detached start where the job forbids that, because worse odds
  beat no helper at all. `detachFlagOrder` writes the flag values out so the
  ordering can be tested anywhere; the behaviour it protects can only be seen on
  Windows.

And nothing about updates happens at startup. The first check is a minute after
the window is up. Justin's instruction, 2026-09-18: open the installed version,
check in the background, and let a small card offer the restart.

If this path strands somebody a fourth time, take the button out of the packaged
build and let Windows do the updating alone. Slower, and it cannot end with
nothing running.

## Updating

Modelled on Rampart, deliberately, because it is the behaviour Justin already
knows: the app says a new version exists and then waits. Nothing installs until
someone presses the button.

**Both kinds of install check.** Windows' own background task is the backstop,
not the mechanism: it runs on a schedule nobody can predict and will leave a
running app a version behind for a day. Leaving the card off for packaged
installs meant the copy most people run was the one that never mentioned an
update, which was the wrong way round.

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

Three traps in the implementation:

- **Release the single-instance mutex before launching the new binary**, or the
  new process sees the old one still holding it and exits immediately, leaving
  nothing running at all.
- **The packaged path must not go through the package's own launcher.**
  `updatecheck.exe` is the MSIX entry point and it only checks for an update in
  Conveyor's `aggressive` mode, which we deliberately are not, because
  aggressive makes every cold start wait on the network before the window
  appears. Run any other way it prints that it is not in aggressive mode and
  launches the app, so the button would have restarted without updating.
  `Add-AppxPackage -AppInstallerFile` is what actually pulls the package in.
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
