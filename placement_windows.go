//go:build windows

package main

import (
	"unsafe"

	"github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	swShowMaximized = 3
	swShowNormal    = 1

	wmSize         = 0x0005
	wmExitSizeMove = 0x0232
	sizeMaximized  = 2
)

type winPoint struct{ X, Y int32 }

type winPlacement struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    winPoint
	PtMaxPosition    winPoint
	RcNormalPosition windows.Rect
}

var (
	procGetWindowPlacement  *windows.LazyProc
	procSetWindowPlacement  *windows.LazyProc
	procEnumDisplayMonitors *windows.LazyProc
	procGetMonitorInfoW     *windows.LazyProc
)

func loadPlacementProcs() bool {
	if !loadUser32() {
		return false
	}
	if procGetWindowPlacement == nil {
		user32 := windows.NewLazySystemDLL("user32.dll")
		procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
		procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
		procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
		procGetMonitorInfoW = user32.NewProc("GetMonitorInfoW")
	}
	return procGetWindowPlacement.Find() == nil && procSetWindowPlacement.Find() == nil &&
		procEnumDisplayMonitors.Find() == nil && procGetMonitorInfoW.Find() == nil
}

// The work areas of the screens attached right now, so a window is never put
// back onto one that has been unplugged.
func screenBoxes() []WindowBox {
	if !loadPlacementProcs() {
		return nil
	}
	var out []WindowBox
	cb := windows.NewCallback(func(hMonitor, hdc, lprc, data uintptr) uintptr {
		var info struct {
			Size    uint32
			Monitor windows.Rect
			Work    windows.Rect
			Flags   uint32
		}
		info.Size = uint32(unsafe.Sizeof(info))
		r, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
		if r != 0 {
			out = append(out, WindowBox{
				X: int(info.Work.Left), Y: int(info.Work.Top),
				W: int(info.Work.Right - info.Work.Left),
				H: int(info.Work.Bottom - info.Work.Top),
			})
		}
		return 1
	})
	_, _, _ = procEnumDisplayMonitors.Call(0, 0, cb, 0)
	return out
}

func currentBox(hwnd windows.HWND) (WindowBox, bool) {
	if hwnd == 0 || !loadPlacementProcs() {
		return WindowBox{}, false
	}
	var p winPlacement
	p.Length = uint32(unsafe.Sizeof(p))
	r, _, _ := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return WindowBox{}, false
	}
	return WindowBox{
		X: int(p.RcNormalPosition.Left), Y: int(p.RcNormalPosition.Top),
		W: int(p.RcNormalPosition.Right - p.RcNormalPosition.Left),
		H: int(p.RcNormalPosition.Bottom - p.RcNormalPosition.Top),
		Maximized: p.ShowCmd == swShowMaximized,
	}, true
}

// restoreBox puts the window back where it was left. The saved size is used
// only when it still lands on a screen; the maximized state is safe either way,
// because Windows decides which screen that means.
func restoreBox(w webview2.WebView, box WindowBox) {
	if box.Empty() {
		return
	}
	hwnd := hwndOf(w)
	if hwnd == 0 || !loadPlacementProcs() {
		return
	}
	var p winPlacement
	p.Length = uint32(unsafe.Sizeof(p))
	if r, _, _ := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p))); r == 0 {
		return
	}
	if boxOnScreen(box, screenBoxes()) {
		p.RcNormalPosition = windows.Rect{
			Left: int32(box.X), Top: int32(box.Y),
			Right: int32(box.X + box.W), Bottom: int32(box.Y + box.H),
		}
	}
	if box.Maximized {
		p.ShowCmd = swShowMaximized
	} else {
		p.ShowCmd = swShowNormal
	}
	_, _, _ = procSetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
}

// rememberBox writes the window's shape back to config.json. Called when a
// drag ends and when the window is maximized, not on every WM_SIZE, because a
// drag emits one of those per frame.
func rememberBox(hwnd windows.HWND) {
	box, ok := currentBox(hwnd)
	if !ok {
		return
	}
	cfg, err := Load()
	if err != nil || cfg.Window == box {
		return
	}
	cfg.Window = box
	_ = Save(cfg)
}
