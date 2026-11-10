package main

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var semverRe = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// changelogHeaderRe: "## [v1.0.0] - 2026-11-10 - grande"
var changelogHeaderRe = regexp.MustCompile(`^## \[v([0-9.]+)\] - (\d{4}-\d{2}-\d{2}) - (menor|medio|grande|inicial)$`)

// the version is one semver constant
func TestVersionIsSemver(t *testing.T) {
	if !semverRe.MatchString(gameVersion) {
		t.Fatalf("gameVersion %q is not MAJOR.MINOR.PATCH", gameVersion)
	}
	if versionLabel() != "v."+gameVersion {
		t.Errorf("label %q", versionLabel())
	}
	for _, s := range []string{gameWindowTitle(), editorWindowTitle()} {
		if !strings.HasSuffix(s, versionLabel()) || !strings.HasPrefix(s, "Party Hardy - ") {
			t.Errorf("window title %q does not end with the version", s)
		}
	}
}

func semverParts(t *testing.T, v string) [3]int {
	t.Helper()
	m := semverRe.FindStringSubmatch(v)
	if m == nil {
		t.Fatalf("%q is not semver", v)
	}
	var p [3]int
	for i := range p {
		p[i], _ = strconv.Atoi(m[i+1])
	}
	return p
}

func semverLess(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// CHANGELOG.md starts with the version of the code; every entry is well formed, newest first, and the kind of
// change matches the number that went up (menor = x.y.Z, medio = x.Y.0, grande = X.0.0)
func TestVersionMatchesChangelog(t *testing.T) {
	b, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	type entry struct {
		v    [3]int
		raw  string
		date string
		kind string
	}
	var es []entry
	for _, l := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(l, "## [") {
			continue
		}
		m := changelogHeaderRe.FindStringSubmatch(l)
		if m == nil {
			t.Errorf("bad header %q (want \"## [vX.Y.Z] - AAAA-MM-DD - menor|medio|grande\")", l)
			continue
		}
		es = append(es, entry{semverParts(t, m[1]), m[1], m[2], m[3]})
	}
	if len(es) < 2 {
		t.Fatalf("only %d entries", len(es))
	}
	if es[0].raw != gameVersion {
		t.Fatalf("CHANGELOG starts with v%s but the game is v%s: raise both together", es[0].raw, gameVersion)
	}
	for i := 0; i+1 < len(es); i++ {
		cur, old := es[i], es[i+1]
		if !semverLess(old.v, cur.v) {
			t.Errorf("v%s is listed above v%s but is not newer", cur.raw, old.raw)
		}
		if cur.date < old.date {
			t.Errorf("v%s (%s) is older than v%s (%s)", cur.raw, cur.date, old.raw, old.date)
		}
		var want string
		switch {
		case cur.v[0] != old.v[0]:
			want = "grande"
		case cur.v[1] != old.v[1]:
			want = "medio"
		default:
			want = "menor"
		}
		if cur.kind != want {
			t.Errorf("v%s is marked %q but the number that went up says %q", cur.raw, cur.kind, want)
		}
		if (want == "medio" && cur.v[2] != 0) || (want == "grande" && (cur.v[1] != 0 || cur.v[2] != 0)) {
			t.Errorf("v%s: the lower numbers must go back to 0", cur.raw)
		}
	}
	// the files that carry the version
	if l, _ := os.ReadFile("LEEME.txt"); !strings.Contains(string(l), versionLabel()) {
		t.Errorf("LEEME.txt does not say %s", versionLabel())
	}
	sh, _ := os.ReadFile("build.sh")
	for _, w := range []string{"version.go", "CHANGELOG.md", "VERSION.txt", "PartyHardy-Mac.zip", "PartyHardy-Windows.zip"} {
		if !strings.Contains(string(sh), w) {
			t.Errorf("build.sh lost %q", w)
		}
	}
	if strings.Contains(string(sh), "<string>1.0</string>") {
		t.Error("build.sh still writes a fixed version in Info.plist")
	}
}

// versionScreenTexts draws a screen in a language and returns the texts drawn.
func versionScreenTexts(t *testing.T, lang string, credits bool) []textRect {
	t.Helper()
	g := auditGame(t)
	ps, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = ps, pe, pu; auditReset(g) })
	gameLang, editorLang, uiLang = lang, lang, lang
	auditReset(g)
	if credits {
		g.mode = ModeSettings
		g.showCredits = true
	} else {
		g.mode = ModeMenu
	}
	textRec = &textRecorder{}
	t.Cleanup(func() { textRec = nil })
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
	return append([]textRect(nil), textRec.texts...)
}

// the version is on screen (main menu corner and credits), in Spanish and English, whole scale, inside the
// screen with >= 16 px to the edge and nothing touching it
func TestVersionShownOnScreen(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		for _, credits := range []bool{false, true} {
			texts := versionScreenTexts(t, lang, credits)
			var ver *textRect
			for i := range texts {
				if strings.Contains(texts[i].s, versionLabel()) {
					ver = &texts[i]
				}
			}
			if ver == nil {
				t.Errorf("[%s credits=%v] the version %s is not drawn", lang, credits, versionLabel())
				continue
			}
			if ver.x < 16 || ver.y < 16 || ver.x+ver.w > float64(screenW)-16 || ver.y+ver.h > float64(screenH)-16 {
				t.Errorf("[%s credits=%v] %q at %.0f,%.0f %.0fx%.0f is closer than 16 px to the edge", lang, credits, ver.s, ver.x, ver.y, ver.w, ver.h)
			}
			if ver.x != float64(int(ver.x)) || ver.y != float64(int(ver.y)) {
				t.Errorf("[%s credits=%v] %q not on whole pixels (%.2f,%.2f): blurry", lang, credits, ver.s, ver.x, ver.y)
			}
			for i := range texts {
				o := &texts[i]
				if o == ver || o.s == "" {
					continue
				}
				// the lines of the same block (the homage under the version, the game name over it in the credits)
				// sit at their line pitch; any other text keeps >= 12 px away
				min := 12.0
				if o.s == tr(menuHomageLine) || o.s == tr("Party Hardy · Fiesta Nocturna") {
					min = -3 // line boxes of one block may share the 2 px of padding under the glyphs
				}
				if g := rectGap(ver, o); g < min {
					t.Errorf("[%s credits=%v] %q is %.1f px from %q (want >= %.0f)", lang, credits, o.s, g, ver.s, min)
				}
			}
		}
	}
}

// rectGap: the air between two texts (the larger of the horizontal and the vertical gap; < 0 = they overlap)
func rectGap(a, b *textRect) float64 {
	dx := math.Max(b.x-(a.x+a.w), a.x-(b.x+b.w))
	dy := math.Max(b.y-(a.y+a.h), a.y-(b.y+b.h))
	return math.Max(dx, dy)
}
