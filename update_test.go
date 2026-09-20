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

// The packaged update is a PowerShell script built by string concatenation and
// run on a machine nobody is watching, so the shape of it is worth pinning.
// Getting it wrong does not fail loudly: it starts nothing, or starts the old
// version, and the app looks like it simply never updates.
func TestPackagedRestartCommand(t *testing.T) {
	cmd := packagedRestartCommand(4242)
	script := cmd[len(cmd)-1]

	// It must not install anything. Replacing the package from inside the app
	// being replaced is what failed four ways, always ending with nothing
	// running, and this is the guard against somebody putting it back.
	if strings.Contains(script, "Add-AppxPackage") {
		t.Error("the restart installs a package again, which is what kept stranding people")
	}
	if strings.Contains(script, "ForceTargetApplicationShutdown") {
		t.Error("the restart still shuts the app down itself")
	}
	// The family name carries a hash of the signing identity, so hardcoding one
	// would stop matching the day the key is replaced, and nothing would say so.
	if !strings.Contains(script, "Get-AppxPackage -Name $n") || !strings.Contains(script, "$n = '"+packageName+"'") {
		t.Error("the package family name should be asked for, not written down")
	}
	// An empty family name builds a path that starts nothing.
	if !strings.Contains(script, "if ($f) { Start-Process") {
		t.Error("the relaunch is not guarded on having a family name")
	}
	// This process has to be gone before the new one runs, or the single
	// instance check finds the old copy and the new one exits. A duration is
	// a guess; the process id is not. 2500ms was the guess, and it is why
	// Restart could leave nothing running.
	if !strings.Contains(script, "Get-Process -Id $p") || !strings.Contains(script, "$p = 4242") {
		t.Error("the relaunch does not wait for this exact process to exit")
	}
	if !strings.Contains(script, "Start-Sleep") {
		t.Error("nothing waits for this copy to exit, so the relaunch would exit instead")
	}
	// The wait has to end even if the process never does, or a failed quit
	// leaves a hidden PowerShell alive for as long as the machine is up.
	if !strings.Contains(script, "$i -lt 80") {
		t.Error("the wait for this process is unbounded")
	}
	// Windows commits a background-staged update when the app closes, and the
	// package is briefly not enumerable while it does. One attempt lands in
	// that window and starts nothing.
	if !strings.Contains(script, "for ($i = 0; $i -lt 20; $i++)") {
		t.Error("the relaunch is a single attempt, so it fails silently mid-update")
	}
	if !strings.Contains(script, "'!"+packageAppID+"'") {
		t.Errorf("no app id in the relaunch: %q", script)
	}
	// One backslash in the script itself. Two in the Go literal, and it is an
	// easy place to end up with a doubled one PowerShell cannot resolve.
	if !strings.Contains(script, `shell:appsFolder\`) || strings.Contains(script, `shell:appsFolder\\`) {
		t.Errorf("appsFolder path is not singly escaped: %q", script)
	}
}
