package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// the RETO DE HOY button: the title, the party and the modifiers fit inside the box, with margin, for every party
// and every combination of up to 3 modifiers, in Spanish and English (the text used to come out of the bottom).
func TestDailyButtonTextFitsBox(t *testing.T) {
	g := auditGame(t)
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu; textRec = nil })
	var combos [][]dailyMod
	for a := dailyMod(0); a < dmN; a++ {
		combos = append(combos, []dailyMod{a})
		for b := a + 1; b < dmN; b++ {
			combos = append(combos, []dailyMod{a, b})
			for c := b + 1; c < dmN; c++ {
				combos = append(combos, []dailyMod{a, b, c})
			}
		}
	}
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	const margin = 6.0
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang = lang, lang
		for lvl := 0; lvl < len(levelDefs); lvl++ {
			for _, mods := range combos {
				d := dailyChallenge{level: lvl, mods: mods}
				lines := g.dailyLinesFor(d)
				if len(lines) > 2 {
					t.Fatalf("[%s] level %d mods %v: %d lines (modifiers must fit in one)", lang, lvl, mods, len(lines))
				}
				textRec = &textRecorder{}
				g.drawDailyBox(img, true, tr("RETO DE HOY"), lines, true)
				if len(textRec.texts) != len(lines)+1 {
					t.Fatalf("drew %d texts for %d lines", len(textRec.texts), len(lines))
				}
				for _, r := range textRec.texts {
					x0, y0, x1, y1 := r.ink()
					if x0 < dailyBtnX+margin || x1 > dailyBtnX+dailyBtnW-margin || y0 < dailyBtnY+margin-3 || y1 > dailyBtnY+dailyBtnH-margin {
						t.Fatalf("[%s] level %d mods %v: %q ink (%.0f,%.0f)-(%.0f,%.0f) outside the box (%.0f,%.0f)-(%.0f,%.0f)", lang, lvl, mods, r.s, x0, y0, x1, y1, dailyBtnX, dailyBtnY, dailyBtnX+dailyBtnW, dailyBtnY+dailyBtnH)
					}
				}
			}
		}
	}
	textRec = nil
}
