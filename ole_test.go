package main

import (
	"os"
	"strings"
	"testing"
)

// Dropping a file from Explorer needs OleInitialize on the thread that owns
// the window, and it has to happen before the window is made. Nothing on this
// machine can reproduce the Windows behaviour, and the failure is silent on
// the one machine that can: no error, no log, the page just never sees a
// dragenter. So the order is pinned here instead.
func TestFileDropsAreEnabledBeforeTheWindow(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("main.go: %v", err)
	}
	text := string(src)
	call := strings.Index(text, "enableFileDrops()")
	window := strings.Index(text, "webview2.NewWithOptions")
	if call < 0 {
		t.Fatal("main no longer calls enableFileDrops: every dropped file will " +
			"be silently ignored on Windows")
	}
	if window < 0 {
		t.Fatal("main no longer creates the window, so this check cannot mean anything")
	}
	if call > window {
		t.Fatal("enableFileDrops runs after the window is created; WebView2 " +
			"registers its drop target while the window is being made, so by " +
			"then it is too late")
	}
}

// OleInitialize, not CoInitializeEx. The edge package already did the second
// one and it is not what drag and drop needs.
func TestFileDropsUseOle(t *testing.T) {
	src, err := os.ReadFile("ole_windows.go")
	if err != nil {
		t.Fatalf("ole_windows.go: %v", err)
	}
	text := string(src)
	if !strings.Contains(text, `NewProc("OleInitialize")`) {
		t.Fatal("enableFileDrops does not call OleInitialize")
	}
	if !strings.Contains(text, "//go:build windows") {
		t.Fatal("ole_windows.go lost its build tag and will break the Linux test build")
	}
}
