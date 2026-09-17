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
		return false
	}
	instanceMutex = h
	return true
}
