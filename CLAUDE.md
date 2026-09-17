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
