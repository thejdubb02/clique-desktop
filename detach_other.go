//go:build !windows

package main

import "os/exec"

// Only Windows has the container this exists to escape.
func startDetached(argv []string) error {
	return exec.Command(argv[0], argv[1:]...).Start()
}
