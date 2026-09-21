# CLIque Desktop

A dedicated desktop window for the [CLIque](https://github.com/thejdubb02/clique)
panel, for Windows.

CLIque runs on a Linux box, because it drives tmux. This is the client you open
on a PC: your own window, your own icon in the taskbar, no browser tab, no URL
bar, and a toast when a session needs an answer.

About 8 MB. No Electron, no bundled browser, no Node. It uses the Microsoft Edge
WebView2 runtime that Windows already ships.

## Install

In PowerShell:

```powershell
irm https://raw.githubusercontent.com/thejdubb02/clique-desktop/main/scripts/install.ps1 | iex
```

Downloads the latest installer, checks it against the checksum published
alongside it, and runs it. No admin prompt: it installs to your own user
profile, adds a Start menu entry and an uninstaller, and opens CLIque when
it's done.

Or by hand: download `CLIque-Setup.exe` from the
[latest release](https://github.com/thejdubb02/clique-desktop/releases/latest)
and run it yourself, same installer, same result.

Either way, the installer and the app are signed with our own certificate
rather than a paid one, so Windows SmartScreen shows an "unknown publisher"
warning the first time. Click **More info → Run anyway**. That is the only
click this ever asks for: it does not come back on an update.

On first run, enter the URL of your panel, for example
`https://yourbox.tailnet.ts.net/clique/` or `http://192.168.1.10:3200/`. That is
the whole setup. The app checks the panel answers before it saves.

### The loose exe

`CLIque.exe` is also published with every release for anyone who would rather
not install anything. It is the exact same binary the installer puts down, just
without the Start menu entry and the uninstaller.

### Notifications (optional)

Leave the token box empty and the app works fine, you just will not get desktop
toasts. To turn them on, mint an API token in the panel under Settings, then
paste it into the second box. The app polls `/api/state` every five seconds and
raises a toast when a session starts waiting for you or hits an error, once per
change rather than once per poll.

You can add it later by editing `%AppData%\CLIque\config.json`.

## Links

A link you click in a session opens in your own browser, the one with your
bookmarks and your logins, not inside this window. Only `http` and `https`
links are handed over, so a page cannot use a link to start a program.

## It matches your theme

The Windows title bar takes the colour of the panel's own chrome, so the window
is one thing rather than two stacked. It follows along when you switch themes.
Windows 11 takes the exact colour; Windows 10 has no way to be told one, so it
gets dark or light chrome to match instead.

## The tray

CLIque sits in the notification area while it runs. The tooltip carries the
version you are running and the number of sessions waiting on you, from the same poll that raises the toasts, so
it needs the token to say anything other than the app's name.

The version is on the icon's menu too, as a label. It is the only place it
appears: the window belongs to the panel, and the number in the panel's own
corner is the panel's version, not this app's.

Closing the window puts it back in the tray rather than quitting, so toasts keep
arriving. Click the tray icon for **Show CLIque** to bring it back and **Quit
CLIque** to actually stop it. Launching CLIque again while it is already running
brings the existing window forward instead of doing nothing.

If the tray cannot start for any reason, the app runs as an ordinary window and
closing it quits, exactly as it did before. Nothing about the tray is allowed to
stop you using the app.

## Updates

**Nothing happens at startup.** Opening CLIque opens the version you have
installed and gets on with it: it never waits on the network and never does
anything about an update while you are trying to start.

CLIque then checks a minute later and every half hour after that. When a new
version exists it downloads and verifies it quietly in the background, and only
then does a small button appear next to the version number, so by the time you
see it a restart is an instant swap, not a wait. Click it and it installs and
reopens on the new version, no second confirmation.

**You don't have to click it.** The next time CLIque closes and reopens, by
any means, it applies whatever was already downloaded and verified on its
own. The button just makes it happen sooner.

The check never blocks startup and never interrupts you when it fails: no
network, a bad release, or no answer at all mean the same thing, which is no
card.

Either way, restarting is safe whenever you feel like it, including in the
middle of something. Your sessions are not running in this app, they are
running in tmux on the panel's machine, so closing this window interrupts
nothing.

## Requirements

- Windows 10 or 11 with the Edge WebView2 runtime, which is preinstalled on
  Windows 11 and on any Windows 10 that has had Edge updated. If it is missing,
  the app says so and points you at the download.
- A reachable CLIque panel. Over the internet, put it behind Tailscale or a
  reverse proxy with TLS, exactly as the panel's own README describes.

## What it does not do

It is a viewer, not a second copy of the panel. It does not run tmux, it cannot
host sessions, and it has no opinion about which CLI you are running. Everything
you see is the panel's own interface.

Not in this version: deep links, or starting the panel for you. Mac and Linux
builds are possible from the same code but are not built yet.

## Build it yourself

Go 1.23 or newer. It cross-compiles from Linux or Mac, no cgo, no Windows
toolchain needed:

```bash
./scripts/build.sh          # writes dist/CLIque.exe
go test ./...               # runs anywhere
```

`scripts/package.sh <version>` does the full release build: the exe above,
signed, plus `CLIque-Setup.exe` built with NSIS (`apt install nsis`) and signed
with [osslsigncode](https://github.com/mtrojnar/osslsigncode) (`apt install
osslsigncode`). Both free, both run on Linux. It needs a code-signing
cert/key; generate a self-signed one with `openssl req -x509 -newkey rsa:3072
-nodes -addext extendedKeyUsage=codeSigning ...` and point `CLIQUE_SIGNING_CERT`
/ `CLIQUE_SIGNING_KEY` at it, or leave them unset to use the estate's own key.

`scripts/release.sh <version>` does the whole thing: package.sh, tag, push,
and `gh release create` with every file package.sh built (by glob, not a
hand-typed list: v0.3.17 shipped without the checksum the in-app updater
needs because a hand-typed asset list left it off), then checks the
*published* release actually has all four expected assets before it calls
itself done. `RELEASE_NOTES="..." scripts/release.sh 0.3.18` to set real
release notes; without it, the release publishes with just the tag name.

## Licence

MIT, same as the panel.
