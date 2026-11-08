//go:build darwin || linux

package main

import "syscall"

// lowerPriority: nice 19 so a test run never makes the PC sluggish.
func lowerPriority() { _ = syscall.Setpriority(syscall.PRIO_PROCESS, 0, 19) }
