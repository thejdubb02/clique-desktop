package main

import (
	"strings"
	"testing"
)

func TestShellInitJSCarriesTheVersion(t *testing.T) {
	js := shellInitJS("0.3.12")
	for _, want := range []string{"window.cliqueShell", `"desktop"`, `"0.3.12"`} {
		if !strings.Contains(js, want) {
			t.Fatalf("shellInitJS() = %q, missing %s", js, want)
		}
	}
}

// A build from source has no version stamped in, and an empty string sitting
// next to the panel's version says less than "dev" does.
func TestShellInitJSNamesAnUnstampedBuild(t *testing.T) {
	for _, in := range []string{"", "   "} {
		if !strings.Contains(shellInitJS(in), `"dev"`) {
			t.Fatalf("shellInitJS(%q) = %q", in, shellInitJS(in))
		}
	}
}

// It is evaluated in the page, so it is quoted rather than pasted. The version
// comes from a linker flag rather than from anyone hostile; this is here so
// that stays true if it ever comes from somewhere else.
func TestJSStringQuotes(t *testing.T) {
	cases := map[string]string{
		`0.3.12`:    `"0.3.12"`,
		`a"b`:       `"a\"b"`,
		`a\b`:       `"a\\b"`,
		"a\nb":      `"a\nb"`,
		`</script>`: "\"\\u003c/script\\u003e\"",
	}
	for in, want := range cases {
		if got := jsString(in); got != want {
			t.Fatalf("jsString(%q) = %s, want %s", in, got, want)
		}
	}
}
