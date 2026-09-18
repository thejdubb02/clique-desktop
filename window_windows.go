//go:build windows

package main

import (
	"sync"
	"unsafe"

	"github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	swHide    = 0
	swShow    = 5
	swRestore = 9
	wmClose   = 0x0010
)

// GWLP_WNDPROC. A negative constant cannot be converted to uintptr, so it
// goes through a variable, where the conversion sign extends as intended.
var gwlpWndProc = int32(-4)

var (
	user32Once sync.Once
	user32OK   bool

	procSetWindowLongPtrW   *windows.LazyProc
	procCallWindowProcW     *windows.LazyProc
	procShowWindow          *windows.LazyProc
	procSetForegroundWindow *windows.LazyProc
	procFindWindowW         *windows.LazyProc

	prevWndProc uintptr
	wndProcCB   uintptr
)

func loadUser32() bool {
	user32Once.Do(func() {
		user32 := windows.NewLazySystemDLL("user32.dll")
		procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
		procCallWindowProcW = user32.NewProc("CallWindowProcW")
		procShowWindow = user32.NewProc("ShowWindow")
		procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
		procFindWindowW = user32.NewProc("FindWindowW")
		if procSetWindowLongPtrW.Find() != nil ||
			procCallWindowProcW.Find() != nil ||
			procShowWindow.Find() != nil ||
			procSetForegroundWindow.Find() != nil ||
			procFindWindowW.Find() != nil {
			return
		}
		user32OK = true
	})
	return user32OK
}

func hwndOf(w webview2.WebView) windows.HWND {
	return windows.HWND(uintptr(w.Window()))
}

func raiseHWND(hwnd windows.HWND) {
	if hwnd == 0 || !loadUser32() {
		return
	}
	_, _, _ = procShowWindow.Call(uintptr(hwnd), swShow)
	_, _, _ = procShowWindow.Call(uintptr(hwnd), swRestore)
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
}

func showWindow(w webview2.WebView) {
	raiseHWND(hwndOf(w))
}

func findWindowByTitle(title string) windows.HWND {
	if !loadUser32() {
		return 0
	}
	name, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(name)))
	return windows.HWND(hwnd)
}

func hideOnClose(w webview2.WebView) {
	// Closing would destroy the window and quit. Hide it so the tray can restore it.
	// If subclassing fails, the existing quit-on-close behaviour is left alone.
	defer func() { recover() }()
	if !loadUser32() {
		return
	}
	hwnd := hwndOf(w)
	if hwnd == 0 {
		return
	}
	wndProcCB = windows.NewCallback(ourProc)
	prev, _, _ := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlpWndProc), wndProcCB)
	if prev == 0 {
		wndProcCB = 0
		return
	}
	prevWndProc = prev
}

func ourProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	if msg == wmClose && !trayQuitting() {
		_, _, _ = procShowWindow.Call(hwnd, swHide)
		return 0
	}
	if prevWndProc == 0 {
		return 0
	}
	r, _, _ := procCallWindowProcW.Call(prevWndProc, hwnd, msg, wParam, lParam)
	return r
}
