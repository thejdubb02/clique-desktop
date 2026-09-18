//go:build windows

package main

import "golang.org/x/sys/windows"

// Held for the process lifetime so the named mutex stays claimed.
var instanceMutex windows.Handle

func claimSingleInstance() bool {
	name, err := windows.UTF16PtrFromString(`Local\clique-desktop`)
	if err != nil {
		return true
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
		// Only step aside for a copy that is actually on screen. A held mutex
		// with no window behind it is a process that died badly, or one still
		// dying after an update restart, and standing down for it leaves
		// nothing running at all. That is how launching the app came to do
		// nothing: the update shut the old copy down, the new one found the
		// mutex not yet released, and exited.
		if hwnd := findWindowByTitle("CLIque"); hwnd != 0 {
			raiseHWND(hwnd)
			return false
		}
		return true
	}
	instanceMutex = h
	return true
}

func releaseSingleInstance() {
	if instanceMutex != 0 {
		_ = windows.CloseHandle(instanceMutex)
		instanceMutex = 0
	}
}
