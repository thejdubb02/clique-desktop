package main

import "testing"

func TestExternalLink(t *testing.T) {
	ok := []string{
		"https://useclique.dev/docs",
		"http://192.168.1.10:3200/",
		"https://fdroid.useclique.dev/repo",
	}
	for _, raw := range ok {
		if got, is := externalLink(raw); !is || got == "" {
			t.Fatalf("%q should open in a browser, got %q %v", raw, got, is)
		}
	}
	// Each of these reaches the platform opener as something other than a page.
	bad := []string{
		"file:///C:/Windows/System32/cmd.exe",
		"ms-settings:windowsupdate",
		"javascript:alert(1)",
		"C:\\Windows\\System32\\cmd.exe",
		"\\\\attacker\\share\\payload.exe",
		"mailto:someone@example.com",
		// These have a host, so nothing but the scheme check stops them.
		// file://server/share is a UNC path the opener would run.
		"file://attacker/share/payload.exe",
		"ftp://example.com/x",
		"ms-windows-store://home",
		"",
		"https://",
		"not a url at all",
	}
	for _, raw := range bad {
		if _, is := externalLink(raw); is {
			t.Fatalf("%q should not be handed to the browser", raw)
		}
	}
}
