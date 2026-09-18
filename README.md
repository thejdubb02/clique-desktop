# CLIque Desktop

A dedicated desktop window for the [CLIque](https://github.com/thejdubb02/clique)
panel, for Windows.

CLIque runs on a Linux box, because it drives tmux. This is the client you open
on a PC: your own window, your own icon in the taskbar, no browser tab, no URL
bar, and a toast when a session needs an answer.

About 8 MB. No Electron, no bundled browser, no Node. It uses the Microsoft Edge
WebView2 runtime that Windows already ships.

## Install

Paste this into PowerShell. It trusts our signing certificate once, then installs
CLIque properly: Start menu entry, taskbar icon, and an entry in Add or remove
programs.

```powershell
irm https://github.com/thejdubb02/clique-desktop/releases/latest/download/install.ps1 | iex
```

The certificate step needs one administrator prompt, once per machine, because
the package is signed with our own key rather than a certificate from a public
authority. Every update after it is silent.

After that, Windows keeps CLIque current in the background whether or not the
app is running, and downloads only the parts that changed. Nothing in the app
polls for updates when it is installed this way, because the operating system
already does.

On first run, enter the URL of your panel, for example
`https://yourbox.tailnet.ts.net/clique/` or `http://192.168.1.10:3200/`. That is
the whole setup. The app checks the panel answers before it saves.

### The loose exe

`CLIque.exe` is still published with every release for anyone who would rather
not install anything. It updates itself instead: a card appears in the corner
when a new version exists and waits for you to press Restart. It is the same
app, just without the Start menu entry and without the operating system doing
the updating.

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

## The tray

CLIque sits in the notification area while it runs. The tooltip carries the
number of sessions waiting on you, from the same poll that raises the toasts, so
it needs the token to say anything other than the app's name.

Closing the window puts it back in the tray rather than quitting, so toasts keep
arriving. Click the tray icon for **Show CLIque** to bring it back and **Quit
CLIque** to actually stop it. Launching CLIque again while it is already running
brings the existing window forward instead of doing nothing.

If the tray cannot start for any reason, the app runs as an ordinary window and
closing it quits, exactly as it did before. Nothing about the tray is allowed to
stop you using the app.

## Updates

**Installed from the PowerShell line above**, Windows does it. It re-reads the
package's manifest on its own schedule, in the background, whether or not the app
is running, and pulls only the blocks that changed. There is nothing to press and
nothing to notice.

**Running the loose exe**, the app does it, and only when you say so. It checks
once after the window opens, then every half hour, and a small card appears in
the bottom corner when there is a new version. Press **Restart now** and it
downloads it, swaps itself out and reopens. Press **Later** and it goes away
until the next check. The check never blocks startup, never interrupts you if it
fails, and the download is verified against a checksum published with the
release.

Either way, restarting is safe whenever you feel like it, including in the middle
of something. Your sessions are not running in this app, they are running in tmux
on the panel's machine, so closing this window interrupts nothing.

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
