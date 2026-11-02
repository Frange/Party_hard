package main

import (
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// after the export every built-in character must exist as an editable file
func TestAllBuiltinsExported(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("cp", "-R", "assets/.", root).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	os.Remove(filepath.Join(root, ".held_objects_fixed"))
	t.Log("repaired:", refreshBakedAssets(root))
	t.Log("added:", exportMissingCharacters(root))
	missing := 0
	slugs := map[string]bool{}
	for _, f := range famousPresets {
		sp := f(newRand(1))
		if sp.Name != "" {
			slugs[safeName(sp.Name)] = true
		}
	}
	for s := range slugs {
		found := false
		for _, cat := range charCategories {
			if _, err := os.Stat(filepath.Join(spritesDir(root, cat), s, "meta.json")); err == nil {
				found = true
			}
		}
		if !found {
			missing++
			t.Errorf("not in the editor: %s", s)
		}
	}
	t.Log("distinct famous:", len(slugs), "missing:", missing)
	for _, n := range []string{"sam_sam", "cesar", "diego", "lagarto_juancho", "bombero", "tecnico", "swat", "mulder", "ranger_red", "military_a"} {
		ok := false
		for _, cat := range charCategories {
			if _, err := os.Stat(filepath.Join(spritesDir(root, cat), n, "meta.json")); err == nil {
				ok = true
			}
		}
		if !ok {
			t.Errorf("%s is not there", n)
		}
	}
}

func newRand(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }
