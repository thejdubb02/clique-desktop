# clique-desktop

Windows viewer for the CLIque panel. WebView2 around the panel URL. It does
not run or talk to tmux.

Build from Linux:

```
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w"
```

Config lives in `%AppData%/CLIque/config.json`. Token is only for desktop
notifications via `/api/state`.
