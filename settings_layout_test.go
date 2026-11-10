package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// setMinGap is the least air between two blocks of the settings screen (title, tabs, column titles, rows,
// notice, language box, logo).
const setMinGap = 12.0

type setBlock struct {
	name       string
	x, y, w, h float64
}

// gapBetween: how far apart two rectangles are (the larger of the horizontal and vertical gaps; negative
// when they overlap).
func gapBetween(a, b setBlock) float64 {
	gx := math.Max(b.x-(a.x+a.w), a.x-(b.x+b.w))
	gy := math.Max(b.y-(a.y+a.h), a.y-(b.y+b.h))
	return math.Max(gx, gy)
}

// settingsBlocks draws the settings screen (tab shown by sel) and returns its blocks as drawn.
func settingsBlocks(t *testing.T, g *Game, sel int) []setBlock {
	t.Helper()
	g.mode, g.setSel = ModeSettings, sel
	settingsJoin = 1 // the notice line is drawn too
	textRec = &textRecorder{}
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
	texts := textRec.texts
	textRec = nil
	settingsJoin = 0
	find := func(es string) []textRect {
		var out []textRect
		for _, r := range texts {
			if r.s == tr(es) {
				out = append(out, r)
			}
		}
		if len(out) == 0 {
			t.Fatalf("%q (%q) not drawn", es, tr(es))
		}
		return out
	}
	tb := func(name string, r textRect) setBlock { return setBlock{name, r.x, r.y, r.w, r.h} }
	var bl []setBlock
	bl = append(bl, tb("title", find("AJUSTES")[0]))
	for i, cx := range setTabX {
		bl = append(bl, setBlock{fmt.Sprintf("tab %d", i+1), cx - setTabW/2, setTabY, setTabW, setTabH})
	}
	bl = append(bl, tb("column Tecla", find("Tecla")[0]), tb("column Mando", find("Mando")[0]))
	l := settingsLayout()
	bl = append(bl, setBlock{"rows", 240, setTop - 4, 680, settingsContentH(l) + 4})
	note := fmt.Sprintf("Jugador %d: pulsa un botón para unirte (una tecla = teclado)", 1)
	bl = append(bl, tb("notice", find(note)[0]))
	bl = append(bl, setBlock{"language box", setLangX, setLangY, setLangW, setLangH})
	for _, w := range []string{"PARTY", "HARDY"} {
		for _, r := range texts {
			if r.s == w && r.kind == "gradText" {
				bl = append(bl, tb("logo "+w, r))
			}
		}
	}
	return bl
}

// the settings screen in Spanish and English, both tabs: every block keeps setMinGap px of air from every
// other one (the logo letters only from the language box and the panel), the panel blocks stay inside the
// panel and the rows of each tab fit above the notice line
func TestSettingsLayoutMargins(t *testing.T) {
	g := auditGame(t)
	pg, pu := gameLang, uiLang
	t.Cleanup(func() {
		gameLang, uiLang = pg, pu
		auditReset(g)
		g.mode, settingsTab = ModeMenu, 0
	})
	panel := setBlock{"panel", 230, 14, 700, 512}
	for _, lang := range []string{"es", "en"} {
		gameLang, uiLang = lang, lang
		for _, sel := range []int{0, setRowP2} {
			auditReset(g)
			bl := settingsBlocks(t, g, sel)
			for i, a := range bl {
				inPanel := a.name != "language box" && a.name[:4] != "logo"
				if inPanel && (a.x < panel.x+4 || a.y < panel.y+4 || a.x+a.w > panel.x+panel.w-4 || a.y+a.h > panel.y+panel.h-4) {
					t.Errorf("%s sel=%d: %s %+v leaves the panel", lang, sel, a.name, a)
				}
				if !inPanel {
					if gp := gapBetween(a, panel); gp < 4 {
						t.Errorf("%s: %s is %.0f px from the panel", lang, a.name, gp)
					}
				}
				for _, b := range bl[i+1:] {
					if a.name[:4] == "logo" && b.name[:4] == "logo" {
						continue
					}
					if gp := gapBetween(a, b); gp < setMinGap {
						t.Errorf("%s sel=%d: %s %+v and %s %+v are %.0f px apart (min %.0f)", lang, sel, a.name, a, b.name, b, gp, setMinGap)
					}
				}
			}
			for _, b := range bl {
				if b.y != math.Round(b.y) || b.x != math.Round(b.x) {
					t.Errorf("%s: %s at a fractional position %+v", lang, b.name, b)
				}
			}
		}
	}
}
