//go:build windows

package main

import (
	"unsafe"

	"github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	dwmwaUseImmersiveDarkMode = 20
	dwmwaCaptionColor         = 35
	dwmwaTextColor            = 36
)

// COLORREF is 0x00BBGGRR. CSS integers are 0xRRGGBB, so red and blue swap.
func colorref(rgb int) uint32 {
	r := uint32(rgb>>16) & 0xFF
	g := uint32(rgb>>8) & 0xFF
	b := uint32(rgb) & 0xFF
	return r | g<<8 | b<<16
}

// setCaption paints the Windows title bar to match the panel. Win11 takes
// the real colours. Win10 does not, so it gets dark or light chrome instead.
// Failures stay silent: a default caption is not worth an error box.
func setCaption(w webview2.WebView, panel int, fg int) {
	hwnd := hwndOf(w)
	if hwnd == 0 {
		return
	}
	proc := windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	if proc.Find() != nil {
		return
	}
	caption := colorref(panel)
	text := colorref(fg)
	_, _, _ = proc.Call(uintptr(hwnd), dwmwaCaptionColor, uintptr(unsafe.Pointer(&caption)), unsafe.Sizeof(caption))
	_, _, _ = proc.Call(uintptr(hwnd), dwmwaTextColor, uintptr(unsafe.Pointer(&text)), unsafe.Sizeof(text))
	var dark int32
	if captionIsDark(panel) {
		dark = 1
	}
	_, _, _ = proc.Call(uintptr(hwnd), dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&dark)), unsafe.Sizeof(dark))
}
