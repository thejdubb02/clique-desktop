package main

import (
	_ "embed"
	"net/url"
)

//go:embed external.js
var externalJS string

// True for a link that belongs in the person's own browser rather than in this
// window. Only http and https: the platform opener will happily launch a file
// path or an application protocol, and the page asking is one the panel
// rendered out of a working directory, so the scheme is checked rather than
// trusted. Kept out of the windows-tagged file so it can be tested anywhere.
func externalLink(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	return u.String(), true
}
