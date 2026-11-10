//go:build !darwin

package main

import "os"

// fileBirthUnix: not portable outside macOS; the modification time is used instead.
func fileBirthUnix(st os.FileInfo) int64 { return 0 }
