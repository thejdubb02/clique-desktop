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
func TestPackagedUpdateCommand(t *testing.T) {
	cmd := packagedUpdateCommand()
	if cmd[0] != "powershell" {
		t.Fatalf("not powershell: %q", cmd[0])
	}
	script := cmd[len(cmd)-1]

	// -NonInteractive matters: a prompt on a hidden window waits forever.
	for _, flag := range []string{"-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command"} {
		found := false
		for _, a := range cmd {
			if a == flag {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %q", flag)
		}
	}

	// Without this Windows refuses to replace a package that is running, and
	// the update silently does nothing at all.
	if !strings.Contains(script, "-ForceTargetApplicationShutdown") {
		t.Error("nothing would shut the running app down, so the install would be refused")
	}
	if !strings.Contains(script, appInstallerURL) {
		t.Error("the manifest url is not in the script")
	}
	// The family name carries a hash of the signing identity. Hardcoding one
	// would stop matching the day the key is replaced, and nothing would say so.
	if !strings.Contains(script, "(Get-AppxPackage -Name "+packageName+").PackageFamilyName") {
		t.Error("the package family name should be asked for, not written down")
	}
	// A relaunch that names no app id starts nothing.
	if !strings.Contains(script, "'!"+packageAppID+"'") {
		t.Errorf("no app id in the relaunch: %q", script)
	}
	// One backslash in the script itself. Two in the Go literal, and it is an
	// easy place to end up with a doubled one that PowerShell then cannot resolve.
	if !strings.Contains(script, `shell:appsFolder\`) || strings.Contains(script, `shell:appsFolder\\`) {
		t.Errorf("appsFolder path is not singly escaped: %q", script)
	}
	// A failed install must not stop the relaunch, or a bad release leaves
	// somebody with nothing running.
	if !strings.Contains(script, "catch { }") {
		t.Error("an install failure would take the relaunch with it")
	}
}
