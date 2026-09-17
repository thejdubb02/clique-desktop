//go:build !windows

package main

func claimSingleInstance() bool { return true }

func releaseSingleInstance() {}
