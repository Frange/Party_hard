package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The secret island and its park (2026-11-11, owner's decision): the names of the film's park, island,
// company and people are not ours, so the game says PARQUE FIESTÁSICO / ISLA NUBLADA (the same proper
// names in ES and EN), JUERGEN on the letterhead and John Jammond signs the invitation. The original
// names are written here in pieces so a search of the repository does not find them in this test either.
// Internal identifiers (specialJurassic, jurassic*.go, jp_* slugs, the -jurassic flag, the "jurassic"
// icon/theme/music keys) are not visible text and may keep the old word: only the bare literal
// "jurassic" is allowed.
var islandBanned = []string{
	"jur" + "assic",
	"jur" + "asic", // jurásico / jurásica once the accents are gone
	"nub" + "lar",
	"ham" + "mond",
	"mal" + "colm",
	"sat" + "tler",
	"ned" + "ry",
	"alan " + "grant",
}

// "in gen" / "ingen" as a word (not "ingenioso", "ingeniero"...).
var islandBannedRe = regexp.MustCompile(`\bin ?` + `gen\b`)

func islandFold(s string) string {
	s = strings.ToLower(s)
	return strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n").Replace(s)
}

func islandCheck(t *testing.T, where, s string) {
	t.Helper()
	low := islandFold(s)
	if low == "jur"+"assic" { // an internal key (flag, icon, theme, music folder alias)
		return
	}
	for _, b := range islandBanned {
		if strings.Contains(low, b) {
			t.Errorf("%s: %q names the film (%q)", where, s, b)
		}
	}
	if islandBannedRe.MatchString(low) {
		t.Errorf("%s: %q names the film's company", where, s)
	}
}

// TestIslandNoFilmNames: no string of the game (texts, translations, titles, trophies, scenes, the
// invitation, DEBUG, the gate sign...), no public json of assets and no LEEME/music README names the
// film's park, island, company or people.
func TestIslandNoFilmNames(t *testing.T) {
	fset := token.NewFileSet()
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			if bl, ok := n.(*ast.BasicLit); ok && bl.Kind == token.STRING {
				if s, err := strconv.Unquote(bl.Value); err == nil {
					islandCheck(t, fset.Position(bl.Pos()).String(), s)
				}
			}
			return true
		})
	}
	// the translations, as the game sees them
	initTables()
	for _, tb := range []*trTable{trEN, trES} {
		for k, v := range tb.exact {
			islandCheck(t, "translation key", k)
			islandCheck(t, "translation", v)
		}
	}
	// the island itself, as built
	d := jurassicLevel()
	islandCheck(t, "island title", d.Title)
	islandCheck(t, "island intro", d.Intro)
	if d.Title != "ISLA NUBLADA" {
		t.Errorf("island title %q, want ISLA NUBLADA", d.Title)
	}
	// public texts and the assets' json (never the user's data files)
	for _, f := range []string{"LEEME.txt", "music/README.md", "README.md"} {
		if b, err := os.ReadFile(f); err == nil {
			for i, line := range strings.Split(string(b), "\n") {
				islandCheck(t, f+":"+strconv.Itoa(i+1), line)
			}
		}
	}
	skip := map[string]bool{"progress.json": true, "progress.json.bak": true, "settings.json": true, "editor_prefs.json": true}
	filepath.Walk("assets", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json") || skip[filepath.Base(p)] {
			return nil
		}
		if b, err := os.ReadFile(p); err == nil {
			islandCheck(t, p, string(b))
		}
		return nil
	})
}

// TestIslandNewNamesShown: the new names are the same in ES and EN and reach the screens that show them.
func TestIslandNewNamesShown(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	uiLang = "en"
	for _, s := range []string{"ISLA NUBLADA", "PARQUE FIESTÁSICO"} {
		if en := tr(s); en != s {
			t.Errorf("%q is a proper name: it must not be translated (got %q)", s, en)
		}
	}
	want := map[string]string{
		"INVITACIÓN A ISLA NUBLADA":              "INVITATION TO ISLA NUBLADA",
		"LLEGADA A ISLA NUBLADA":                 "ARRIVAL AT ISLA NUBLADA",
		"¡Le esperamos en el Parque Fiestásico!": "See you at the Parque Fiestásico!",
		"JUERGEN — INVITACIÓN EXCLUSIVA":         "JUERGEN — EXCLUSIVE INVITATION",
		"— John Jammond":                         "— John Jammond",
		"¡Taxi! ¡A Isla Nublada!":                "Taxi! To Isla Nublada!",
		"Era cretácica":                          "Cretaceous era",
		"Última fiesta cretácica":                "Last Cretaceous party",
	}
	for es, en := range want {
		if got := tr(es); got != en {
			t.Errorf("tr(%q) in EN = %q, want %q", es, got, en)
		}
	}
	if partyTypeFor("ISLA NUBLADA") != "Última fiesta cretácica" || genreForTitle("ISLA NUBLADA").id != "isla" {
		t.Errorf("the island lost its subtitle or its music: %q %+v", partyTypeFor("ISLA NUBLADA"), genreForTitle("ISLA NUBLADA"))
	}
	inv := ""
	for _, p := range invParas() {
		inv += p.text + "\n"
	}
	for _, s := range []string{"JUERGEN", "Isla Nublada", "Parque Fiestásico", "John Jammond"} {
		if !strings.Contains(inv, s) {
			t.Errorf("the invitation lost %q:\n%s", s, inv)
		}
	}
}

// TestIslandGateSignSpelled: every letter of the gate sign has a glyph (a missing glyph is a gap) and
// the two lines fit inside the purple board with room to spare.
func TestIslandGateSignSpelled(t *testing.T) {
	for _, line := range []string{"PARQUE", "FIESTÁSICO"} {
		for _, r := range line {
			if base, ok := jpAccent[r]; ok {
				r = base
			}
			if _, ok := jpGlyphs[r]; !ok {
				t.Errorf("no glyph for %q in %q", r, line)
			}
		}
	}
	// board (inner purple) x 32..158; text at k=2: 12 px per letter, the last one 10 px wide; edge 1 px
	for _, c := range []struct {
		s string
		x int
	}{{"PARQUE", 60}, {"FIESTÁSICO", 36}} {
		w := 12*len([]rune(c.s)) - 2
		if c.x-1 < 32+2 || c.x+w+1 > 158-2 {
			t.Errorf("%q at x=%d (w %d) does not fit the sign 32..158 with margin", c.s, c.x, w)
		}
		if mid := c.x + w/2; mid < 94 || mid > 96 {
			t.Errorf("%q is not centred (middle %d, sign middle 95)", c.s, mid)
		}
	}
}
