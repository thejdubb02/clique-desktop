//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// APPMODEL_ERROR_NO_PACKAGE. Anything else, including the insufficient-buffer
// error this deliberately provokes, means we are inside a package.
const appmodelNoPackage = 15700

// True when Windows launched us from the MSIX rather than from a loose exe.
// There the install directory is read only and the operating system keeps the
// app current on its own, so the in-app updater has nothing to do and could
// not do it if it tried.
func runningPackaged() bool {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentPackageFullName")
	if proc.Find() != nil {
		return false
	}
	length := uint32(0)
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&length)), 0)
	return r != appmodelNoPackage
}
