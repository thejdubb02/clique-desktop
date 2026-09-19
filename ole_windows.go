//go:build windows

package main

import "golang.org/x/sys/windows"

// A file dragged in from Explorer arrives through OLE, and OLE is not COM.
//
// go-webview2's edge package locks the OS thread and calls
// CoInitializeEx(nil, COINIT_APARTMENTTHREADED) in its package init, and
// nothing more. WebView2 accepts an external drop by registering an
// IDropTarget on the host window, and RegisterDragDrop answers
// CO_E_NOTINITIALIZED on a thread that has never called OleInitialize. It
// fails quietly: nothing is logged, no error reaches us, and the page simply
// never sees a dragenter, so the window looks like it is ignoring the file.
//
// OleInitialize initialises COM as an STA on the way past and treats finding
// one already there as S_FALSE rather than a failure, so running after that
// init is fine. It has to be this thread: the one the edge package locked,
// which is the one main runs on and the one that owns the message loop.
//
// The return is ignored on purpose. There is no console in a -H windowsgui
// binary, and a message box because drag and drop will not work is worse than
// the thing it is reporting.
func enableFileDrops() {
	windows.NewLazySystemDLL("ole32.dll").NewProc("OleInitialize").Call(0)
}
