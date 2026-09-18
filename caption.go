package main

import _ "embed"

//go:embed caption.js
var captionJS string

// captionIsDark reports whether Windows 10 should draw a dark caption for this
// panel colour. Win11 takes the colour itself; this is the fallback that at
// least agrees with the theme.
func captionIsDark(rgb int) bool {
	r := float64((rgb>>16)&0xFF) / 255
	g := float64((rgb>>8)&0xFF) / 255
	b := float64(rgb&0xFF) / 255
	return 0.2126*r+0.7152*g+0.0722*b < 0.5
}
