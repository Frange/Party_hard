package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// the party list breathes: no two texts that share a column are closer than 6 px of ink, the rows stay inside the
// panel, the title is clear of the RETO DE HOY button, and no speech bubble of the bedroom is drawn over the list.
func TestLevelSelectSpacing(t *testing.T) {
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu })
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang = lang, lang
		for _, sel := range []int{0, 9, 19, 24, 27} {
			g := auditGame(t)
			auditReset(g)
			t.Cleanup(func() { auditReset(g); g.mode = ModeMenu })
			g.mode, g.menuSel = ModeLevelSelect, sel
			g.time = 40 // any moment: the sleeper may be complaining right now
			textRec = &textRecorder{}
			img := ebiten.NewImage(screenW, screenH)
			g.Draw(img)
			img.Deallocate()
			var texts []textRect
			for _, r := range textRec.texts {
				if r.x >= 230 && r.s != "3:47" { // the panel; the logo and the clock of the bedroom are outside it
					texts = append(texts, r)
				}
			}
			textRec = nil
			for i, a := range texts {
				ax0, ay0, ax1, ay1 := a.ink()
				if ay0 < 14 || ay1 > 525 {
					t.Errorf("[%s] %q leaves the panel (ink y %.0f-%.0f)", lang, a.s, ay0, ay1)
				}
				if ay1 > levelFreeY { // the strip under the list stays free: no help / controls lines
					t.Errorf("[%s] %q is drawn under the list (ink to y %.0f, free from %.0f)", lang, a.s, ay1, levelFreeY)
				}
				for _, b := range texts[i+1:] {
					bx0, by0, bx1, by1 := b.ink()
					if bx1 <= ax0 || bx0 >= ax1 {
						continue // other column
					}
					gap := by0 - ay1
					if ay0 > by0 {
						gap = ay0 - by1
					}
					if gap < 6 {
						t.Errorf("[%s] %q and %q are %.0f px apart", lang, a.s, b.s, gap)
					}
				}
			}
			for _, l := range sleeperLines {
				for _, r := range texts {
					if r.s == tr(l) {
						t.Errorf("[%s] the sleeper's bubble %q is drawn over the party list", lang, l)
					}
				}
			}
		}
	}
}
