package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipWithVersion(t *testing.T, path, entry, text string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create(entry)
	w.Write([]byte(text))
	zw.Close()
	f.Close()
}

// the version of version.go, the first CHANGELOG entry and the VERSION.txt of each zip must agree
func TestVersionCheck(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "version.go"), []byte("package main\n\nconst gameVersion = \"1.2.3\"\n"), 0o644)
	os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# x\n\n## [v1.2.3] - 2026-11-10 - menor\n\n## [v1.2.2] - 2026-11-09 - menor\n"), 0o644)
	mac, win := filepath.Join(root, "m.zip"), filepath.Join(root, "w.zip")
	zipWithVersion(t, mac, "mac/VERSION.txt", "Party Hardy v1.2.3\n")
	zipWithVersion(t, win, "VERSION.txt", "Party Hardy v1.2.3\n")
	reps := []*zipReport{verifyZip(mac, nil), verifyZip(win, nil)}
	if reps[0].version != "Party Hardy v1.2.3" {
		t.Fatalf("read %q", reps[0].version)
	}
	if line, ok := versionCheck(root, reps); !ok {
		t.Fatalf("matching versions rejected: %s", line)
	}
	reps[1].version = "v1.2.2"
	if line, ok := versionCheck(root, reps); ok || !strings.Contains(line, "MISMATCH") {
		t.Fatalf("a stale zip passed: %s", line)
	}
	reps[1].version = ""
	if _, ok := versionCheck(root, reps); ok {
		t.Fatal("a zip without VERSION.txt passed")
	}
	os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("## [v1.2.2] - 2026-11-09 - menor\n"), 0o644)
	reps[1].version = "v1.2.3"
	if _, ok := versionCheck(root, reps); ok {
		t.Fatal("a CHANGELOG behind the code passed")
	}
}
