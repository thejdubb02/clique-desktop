package main

import "fmt"

// The version is in here because there is nowhere else to see it. The window
// is the panel's, and the panel reports its own version, not this app's, so
// after an update there was no way to tell whether the restart took.
func trayTooltip(version string, waiting int) string {
	name := "CLIque"
	// "dev" is what a build from source carries, and "CLIque dev" says less
	// than the name on its own.
	if version != "" && version != "dev" {
		name += " " + version
	}
	switch waiting {
	case 0:
		return name
	case 1:
		return name + " - 1 session needs you"
	default:
		return fmt.Sprintf("%s - %d sessions need you", name, waiting)
	}
}
