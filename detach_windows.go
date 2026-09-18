//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func startDetached(argv []string) error {
	var err error
	for _, flags := range detachFlagOrder() {
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags, HideWindow: true}
		if err = cmd.Start(); err == nil {
			return nil
		}
	}
	return err
}
