package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// mainFlagNames parses main.go and returns the name of every flag.Xxx("name", ...) defined there.
func mainFlagNames(t *testing.T) []string {
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "flag" {
			return true
		}
		switch sel.Sel.Name {
		case "Int", "Int64", "Bool", "String", "Float64", "Duration", "Uint", "Uint64":
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if nm, err := strconv.Unquote(lit.Value); err == nil {
					names = append(names, nm)
				}
			}
		}
		return true
	})
	return names
}

// Every flag defined in main.go freezes the progress, except the normal-use ones.
func TestFrozenDebugFlagsActive(t *testing.T) {
	normal := map[string]bool{"mute": true, "lang": true, "noassets": true, "title": true, "notitle": true, "editor": true, "visible": true}
	names := mainFlagNames(t)
	if len(names) < 50 {
		t.Fatalf("only %d flags found in main.go: the parser lost them", len(names))
	}
	seen := map[string]bool{}
	for _, name := range names {
		seen[name] = true
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.String(name, "", "")
		if err := fs.Parse([]string{"-" + name, "1"}); err != nil {
			t.Fatal(err)
		}
		if got := debugFlagsActive(fs); got == normal[name] {
			t.Errorf("-%s: debugFlagsActive=%v, but normal-use=%v", name, got, normal[name])
		}
	}
	for name := range normal {
		if !seen[name] {
			t.Errorf("normal-use flag -%s is not defined in main.go (stale whitelist)", name)
		}
	}
	for _, must := range []string{"level", "shot", "ending", "players", "jurassic", "twop", "firstrun", "stealthbox", "camkill", "endstage"} {
		if !seen[must] {
			t.Errorf("flag -%s expected in main.go", must)
		}
	}
	if debugFlagsActive(flag.NewFlagSet("t", flag.ContinueOnError)) {
		t.Error("no flags: not a debug run")
	}
}

// With the progress frozen, saveProgress neither creates nor modifies the file.
func TestFrozenSaveProgressWritesNothing(t *testing.T) {
	oldFrozen, oldNo := progressFrozen, noAssets
	wd, _ := os.Getwd()
	t.Cleanup(func() { progressFrozen, noAssets = oldFrozen, oldNo; os.Chdir(wd) })
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0o755)
	os.Chdir(dir)
	noAssets = false
	path := progressPath()
	if path == "" {
		t.Fatal("no progress path")
	}
	if rp, _ := filepath.EvalSymlinks(filepath.Dir(path)); rp != func() string { r, _ := filepath.EvalSymlinks(filepath.Join(dir, "assets")); return r }() {
		t.Fatalf("progressPath outside the temp dir: %q", path)
	}
	progressFrozen = true
	saveProgress(Progress{Unlocked: 7})
	if _, err := os.Stat(path); err == nil {
		t.Fatal("frozen saveProgress created the file")
	}
	os.WriteFile(path, []byte("orig"), 0o644)
	saveProgress(Progress{Unlocked: 9})
	if b, _ := os.ReadFile(path); string(b) != "orig" {
		t.Fatalf("frozen saveProgress modified the file: %q", b)
	}
	progressFrozen = false
	saveProgress(Progress{Unlocked: 9})
	if b, _ := os.ReadFile(path); string(b) == "orig" {
		t.Fatal("unfrozen saveProgress should write")
	}
}

// Without a progress.json (a fresh checkout: the file is not in the repo) the
// game starts a new one: one level unlocked, nothing else.
func TestMissingProgressFileGivesDefault(t *testing.T) {
	oldNo := noAssets
	wd, _ := os.Getwd()
	t.Cleanup(func() { noAssets = oldNo; os.Chdir(wd) })
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0o755)
	os.Chdir(dir)
	noAssets = false
	p := loadProgress()
	if p.Unlocked != 1 {
		t.Fatalf("default Unlocked = %d, want 1", p.Unlocked)
	}
}
