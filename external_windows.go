//go:build windows

package main

import "golang.org/x/sys/windows"

// Hands the link to whatever Windows has as the default browser.
func openExternal(raw string) {
	target, ok := externalLink(raw)
	if !ok {
		return
	}
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return
	}
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return
	}
	_ = windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}
