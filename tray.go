package main

import "fmt"

func trayTooltip(waiting int) string {
	switch waiting {
	case 0:
		return "CLIque"
	case 1:
		return "CLIque - 1 session needs you"
	default:
		return fmt.Sprintf("CLIque - %d sessions need you", waiting)
	}
}
