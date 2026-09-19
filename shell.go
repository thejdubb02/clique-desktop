package main

import (
	"encoding/json"
	"strings"
)

// The page has no way to know what is holding it. It shows the panel's
// version, which is the server's, and the server's version is not this app's:
// on 2026-09-19 the desktop had been through eleven releases and there was
// nowhere at all to read which one was running.
//
// So the shell announces itself, as one line of JavaScript evaluated before
// any page script. The panel prints it beside its own version when it is
// there and prints nothing when it is not, which is what a browser does.
func shellInitJS(version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		v = "dev"
	}
	return "window.cliqueShell = { kind: \"desktop\", version: " + jsString(v) + " };"
}

// jsString renders a Go string as a JavaScript literal safe to paste into Eval.
// Lives here rather than beside the window code so it is reachable from a test
// on any machine, not only a Windows build.
func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}
