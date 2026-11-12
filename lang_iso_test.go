package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestTestsRestoreLanguage is the guard of the language globals: a test once left uiLang = "en" and
// the tests after it ran in English (and failed or passed by luck under -shuffle). Every function of
// the test files that assigns uiLang/gameLang/editorLang, or restores only gameLang/editorLang (g.Draw
// copies gameLang into uiLang, so that still leaks uiLang), must call snapshotLang(t) or put uiLang
// back itself in a defer / t.Cleanup.
func TestTestsRestoreLanguage(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no test files found: %v", err)
	}
	langVars := map[string]bool{"uiLang": true, "gameLang": true, "editorLang": true}
	mentions := func(n ast.Node, name string) bool {
		found := false
		ast.Inspect(n, func(m ast.Node) bool {
			if id, ok := m.(*ast.Ident); ok && id.Name == name {
				found = true
			}
			return !found
		})
		return found
	}
	fs := token.NewFileSet()
	checked := 0
	for _, f := range files {
		af, err := parser.ParseFile(fs, f, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || fd.Name.Name == "snapshotLang" {
				continue
			}
			touches, restores := false, false
			restoreIn := func(n ast.Node) {
				if mentions(n, "gameLang") || mentions(n, "editorLang") {
					touches = true
				}
				if mentions(n, "uiLang") {
					restores = true
				}
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.AssignStmt:
					for _, l := range x.Lhs {
						if id, ok := l.(*ast.Ident); ok && langVars[id.Name] {
							touches = true
						}
					}
				case *ast.DeferStmt:
					restoreIn(x)
				case *ast.CallExpr:
					if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "snapshotLang" {
						restores = true
					}
					if se, ok := x.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "Cleanup" {
						restoreIn(x)
					}
				}
				return true
			})
			if touches {
				checked++
				if !restores {
					t.Errorf("%s: %s changes the language globals without snapshotLang(t) (testiso_test.go) or putting uiLang back",
						fs.Position(fd.Pos()), fd.Name.Name)
				}
			}
		}
	}
	if checked < 20 {
		t.Fatalf("only %d functions touch the language: the guard lost its coverage", checked)
	}
}
