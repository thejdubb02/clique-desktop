# CLIque Desktop

A dedicated desktop window for the [CLIque](https://github.com/thejdubb02/clique)
panel, for Windows.

CLIque runs on a Linux box, because it drives tmux. This is the client you open
on a PC: your own window, your own icon in the taskbar, no browser tab, no URL
bar, and a toast when a session needs an answer.

About 8 MB. No Electron, no bundled browser, no Node. It uses the Microsoft Edge
WebView2 runtime that Windows already ships.

## Install

1. Download `CLIque.exe` from the [releases](https://github.com/thejdubb02/clique-desktop/releases).
2. Run it. Windows may warn that the publisher is unrecognised, because the
   binary is not code-signed yet: choose More info, then Run anyway.
3. Enter the URL of your panel, for example
   `https://yourbox.tailnet.ts.net/clique/` or `http://192.168.1.10:3200/`.

That is the whole setup. The app checks the panel answers before it saves.

### Notifications (optional)

Leave the token box empty and the app works fine, you just will not get desktop
toasts. To turn them on, mint an API token in the panel under Settings, then
paste it into the second box. The app polls `/api/state` every five seconds and
raises a toast when a session starts waiting for you or hits an error, once per
change rather than once per poll.

You can add it later by editing `%AppData%\CLIque\config.json`.

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

Not in this version: a tray icon, auto-update, deep links, or starting the panel
for you. Mac and Linux builds are possible from the same code but are not built
yet.

## Build it yourself

Go 1.23 or newer. It cross-compiles from Linux or Mac, no cgo, no Windows
toolchain needed:

```bash
./scripts/build.sh          # writes dist/CLIque.exe
go test ./...               # runs anywhere
```

## Licence

MIT, same as the panel.
