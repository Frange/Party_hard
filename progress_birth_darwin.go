//go:build darwin

package main

import (
	"os"
	"syscall"
)

// fileBirthUnix: when the file was created (macOS keeps it); 0 if unknown.
func fileBirthUnix(st os.FileInfo) int64 {
	if s, ok := st.Sys().(*syscall.Stat_t); ok {
		return s.Birthtimespec.Sec
	}
	return 0
}
