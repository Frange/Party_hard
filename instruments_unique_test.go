package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Two init() that assign the same instruments["x"] silently overwrite each other (the last file by name wins):
// the menu's pad once replaced the level songs' pad. Every key must be registered in one place only.
func TestInstrumentKeysRegisteredOnce(t *testing.T) {
	re := regexp.MustCompile(`instruments\["([^"]+)"\]\s*=`)
	files, _ := filepath.Glob("*.go")
	seen := map[string]string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			if prev, dup := seen[m[1]]; dup {
				t.Errorf("instruments[%q] defined in %s and %s", m[1], prev, f)
			}
			seen[m[1]] = f
		}
	}
	if len(seen) < 10 {
		t.Errorf("only %d instrument keys found: the scan is broken", len(seen))
	}
}
