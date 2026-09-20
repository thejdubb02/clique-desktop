package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A release server good enough for the staging tests: one handler for the
// binary, one for its checksum, both served from bytes chosen by the test.
func fakeRelease(t *testing.T, body []byte) (exeURL, sumURL string, cleanup func()) {
	t.Helper()
	sum := sha256.Sum256(body)
	mux := http.NewServeMux()
	mux.HandleFunc("/exe", func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	mux.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hex.EncodeToString(sum[:]) + "  CLIque.exe\n"))
	})
	srv := httptest.NewServer(mux)
	return srv.URL + "/exe", srv.URL + "/sum", srv.Close
}

// The full lifecycle a background poll and a restart actually go through:
// stage while the app keeps running, verify it from a "later process" that
// only trusts the sidecar, then swap it into place.
func TestStageVerifySwapLifecycle(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "CLIque.exe")
	if err := os.WriteFile(exe, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}
	exeURL, sumURL, cleanup := fakeRelease(t, []byte("new binary"))
	defer cleanup()

	if verifyStaged(exe) {
		t.Fatal("nothing staged yet, should not verify")
	}
	if err := stageUpdate(exe, exeURL, sumURL); err != nil {
		t.Fatalf("stageUpdate: %v", err)
	}
	if !verifyStaged(exe) {
		t.Fatal("staged download should verify against its own sidecar")
	}
	// The running binary must be untouched until the swap.
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("staging touched the running binary: %q", got)
	}

	if err := swapInStaged(exe); err != nil {
		t.Fatalf("swapInStaged: %v", err)
	}
	got, err := os.ReadFile(exe)
	if err != nil || string(got) != "new binary" {
		t.Fatalf("exe after swap = %q, %v; want new binary", got, err)
	}
	if old, err := os.ReadFile(exe + ".old"); err != nil || string(old) != "old binary" {
		t.Fatalf(".old after swap = %q, %v; want old binary", old, err)
	}
	newPath, sumPath := stagedPaths(exe)
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Error("staged .new should be gone once swapped in")
	}
	if _, err := os.Stat(sumPath); !os.IsNotExist(err) {
		t.Error("staged sidecar should be gone once swapped in")
	}
}

// The download that catches a compromised or truncated release: a body that
// does not match its own published checksum must leave nothing staged.
func TestStageUpdateChecksumMismatchLeavesNothing(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "CLIque.exe")
	if err := os.WriteFile(exe, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/exe", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("new binary")) })
	mux.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("0", 64))) // deliberately wrong digest
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := stageUpdate(exe, srv.URL+"/exe", srv.URL+"/sum"); err == nil {
		t.Fatal("checksum mismatch should fail stageUpdate")
	}
	newPath, sumPath := stagedPaths(exe)
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Error("a failed stage should not leave the download behind")
	}
	if _, err := os.Stat(sumPath); !os.IsNotExist(err) {
		t.Error("a failed stage should not leave a sidecar behind")
	}
}

// A sidecar with no download next to it, or one that no longer matches (a
// pulled release, a half-written file from a crash), must never be trusted.
func TestVerifyStagedRejectsTamperedOrOrphaned(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "CLIque.exe")
	newPath, sumPath := stagedPaths(exe)

	if err := os.WriteFile(sumPath, []byte(strings.Repeat("a", 64)), 0644); err != nil {
		t.Fatal(err)
	}
	if verifyStaged(exe) {
		t.Fatal("a sidecar with no staged binary must not verify")
	}

	if err := os.WriteFile(newPath, []byte("something else entirely"), 0644); err != nil {
		t.Fatal(err)
	}
	if verifyStaged(exe) {
		t.Fatal("content that does not match the sidecar must not verify")
	}

	dropStaged(exe)
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Error("dropStaged left the orphaned download behind")
	}
	if _, err := os.Stat(sumPath); !os.IsNotExist(err) {
		t.Error("dropStaged left the orphaned sidecar behind")
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		candidate, running string
		want               bool
	}{
		{"0.2.0", "0.1.0", true},
		{"0.10.0", "0.9.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"1.2.3-4", "1.2.3", true},
		{"1.2.3", "1.2.3-4", false},
		{"1.2.3-4", "1.2.3-4", false},
		{"1.x.3", "1.0.2", true},
		{"1.x.3", "1.0.3", false},
		{"nightly", "0.1.0", false},
	}
	for _, tt := range tests {
		got := isNewer(tt.candidate, tt.running)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.candidate, tt.running, got, tt.want)
		}
	}
}

func TestIsNewerGarbageDoesNotPanic(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("isNewer panicked on garbage: %v", rec)
		}
	}()
	_ = isNewer("1.x.3", "0.1.0")
	_ = isNewer("??", "not-a-version")
	_ = isNewer("", "")
}
