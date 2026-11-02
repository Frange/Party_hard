package main

import (
	"os"
	"path/filepath"
	"testing"
)

// what is saved as imported goes to <project>/imports, is found again by the loaders and never lands in assets
func TestImportsFolder(t *testing.T) {
	proj := t.TempDir()
	root := filepath.Join(proj, "assets")
	os.MkdirAll(spritesDir(root, "people"), 0o755)
	ca := newCharAsset("amigo", "people", 24, 44)
	saveRoot, name := importSaveRoot(root, true, "people", "amigo")
	ca.Meta.Name = name
	if err := saveCharAsset(saveRoot, ca); err != nil {
		t.Fatal(err)
	}
	if !dirExists(filepath.Join(proj, "imports", "sprites", "people", "amigo")) {
		t.Fatal("not saved under imports/")
	}
	if dirExists(filepath.Join(root, "sprites", "people", "amigo")) {
		t.Fatal("imported character leaked into assets/")
	}
	if got := spriteNames(root, "people"); len(got) != 1 || got[0] != "amigo" {
		t.Fatalf("names: %v", got)
	}
	if !inImports(root, spriteDirOf(root, "people", "amigo")) {
		t.Fatal("dir not resolved to imports")
	}
	// a name that already exists in assets gets a suffix instead of overwriting it
	os.MkdirAll(filepath.Join(root, "sprites", "people", "abogado"), 0o755)
	if _, n := importSaveRoot(root, true, "people", "abogado"); n != "abogado_importado" {
		t.Fatalf("collision name: %s", n)
	}
	if r, n := importSaveRoot(root, false, "people", "abogado"); r != root || n != "abogado" {
		t.Fatal("non imported must save to assets")
	}
}
