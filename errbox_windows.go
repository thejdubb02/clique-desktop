//go:build windows

package main

import "golang.org/x/sys/windows"

// alert puts a message in front of the user. Without it a missing WebView2
// runtime looks like the app doing nothing at all when it is double-clicked,
// which is indistinguishable from a broken download.
func alert(caption, text string) {
	c, err := windows.UTF16PtrFromString(caption)
	if err != nil {
		return
	}
	t, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	const mbIconError = 0x00000010
	_, _ = windows.MessageBox(0, t, c, mbIconError)
}
