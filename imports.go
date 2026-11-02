package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// The imports folder: everything made from a picture you brought in (photos of
// friends, drawings...) is saved in <project>/imports, next to the assets
// folder, with the same layout (imports/sprites/<category>/<name>,
// imports/backgrounds/<name>). It is never uploaded to the repository
// (see .gitignore) and the game and the editor read it together with assets.
// ---------------------------------------------------------------------------

func importsRoot(root string) string { return filepath.Join(filepath.Dir(root), "imports") }

// inImports reports whether dir lives inside the imports folder.
func inImports(root, dir string) bool {
	rel, err := filepath.Rel(importsRoot(root), dir)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// unionNames lists the sub-folders of both places, once each, sorted.
func unionNames(a, b string) []string {
	seen := map[string]bool{}
	var out []string
	for _, d := range []string{a, b} {
		for _, n := range listDirs(d) {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

func spriteNames(root, cat string) []string {
	return unionNames(spritesDir(root, cat), spritesDir(importsRoot(root), cat))
}

// spriteDirOf is where a character or object really is (imports win over assets).
func spriteDirOf(root, cat, name string) string {
	if d := filepath.Join(spritesDir(importsRoot(root), cat), name); dirExists(d) {
		return d
	}
	return filepath.Join(spritesDir(root, cat), name)
}

func backgroundNames(root string) []string {
	return unionNames(backgroundsDir(root), backgroundsDir(importsRoot(root)))
}

func backgroundDirOf(root, name string) string {
	if d := filepath.Join(backgroundsDir(importsRoot(root)), name); dirExists(d) {
		return d
	}
	return filepath.Join(backgroundsDir(root), name)
}

// importSaveRoot is the root to save into: imports for things that came from a picture.
// A name that already exists among the game's own assets gets a suffix so nothing is overwritten.
func importSaveRoot(root string, imported bool, cat, name string) (string, string) {
	if !imported {
		return root, name
	}
	if cat != "" && dirExists(filepath.Join(spritesDir(root, cat), name)) {
		name += "_importado"
	}
	return importsRoot(root), name
}
