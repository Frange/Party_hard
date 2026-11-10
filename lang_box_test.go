package main

import (
	"math"
	"testing"
)

// The language box (left of AJUSTES) holds its title and the "◄ Idioma ►" selector with >= 6 px of
// margin, centred, on whole pixels, in every language.
func TestSettingsLangBoxFits(t *testing.T) {
	g := auditGame(t)
	prevG, prevU := gameLang, uiLang
	t.Cleanup(func() { gameLang, uiLang = prevG, prevU })
	whole := func(v float64) bool { return v == math.Round(v) }
	for _, lg := range []string{"es", "en", otherLang("es"), otherLang("en")} {
		gameLang, uiLang = lg, lg
		title, label, ty, ly := g.settingsLangLayout()
		lw := g.textW(label, 2)
		tw := g.textW(title, 1)
		_, h1 := textMeasure(g, tr(title))
		_, h2 := textMeasure(g, tr(label))
		h1, h2 = h1*pxScale(1), h2*pxScale(2)
		cx := setLangX + setLangW/2
		lx := math.Round(cx - lw/2)
		tx := math.Round(cx - tw/2)
		if lx-setLangX < 6 || setLangX+setLangW-(lx+lw) < 6 || tx-setLangX < 6 {
			t.Errorf("%s: selector x %.0f..%.0f too close to the box %.0f..%.0f", lg, lx, lx+lw, setLangX, setLangX+setLangW)
		}
		if math.Abs((lx-setLangX)-(setLangX+setLangW-(lx+lw))) > 1 {
			t.Errorf("%s: selector not centred horizontally", lg)
		}
		if ty-setLangY < 6 || setLangY+setLangH-(ly+h2) < 6 || ly < ty+h1 {
			t.Errorf("%s: text y %.0f..%.0f outside the box %.0f..%.0f", lg, ty, ly+h2, setLangY, setLangY+setLangH)
		}
		if math.Abs((ty-setLangY)-(setLangY+setLangH-(ly+h2))) > 1 {
			t.Errorf("%s: text not centred vertically", lg)
		}
		if !whole(lx) || !whole(tx) || !whole(ty) || !whole(ly) || pxScale(2) != 2 || pxScale(1) != 1 {
			t.Errorf("%s: fractional text coordinates or scale", lg)
		}
	}
}
