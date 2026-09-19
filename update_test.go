package main

import (
	"strings"
	"testing"
)

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
