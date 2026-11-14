package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The big title of every party (the campaign and the random venues), in Spanish and in English, keeps >= 12 px from
// the HUD panel and from the screen edges (the margins rule); short names keep the big size.
func TestLevelTitleClearOfHUD(t *testing.T) {
	snapshotLang(t)
	g := hudPlay(t, false)
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	prevLvl, prevT := g.lvl, g.titleT
	t.Cleanup(func() { g.lvl, g.titleT = prevLvl, prevT })
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		for _, d := range levelDefs {
			g.lvl = &Level{Def: d}
			g.titleT = 1
			textRec = &textRecorder{}
			g.drawLevelTitle(img)
			rec := textRec
			textRec = nil
			found := false
			titleBottom, lines := 0.0, 0
			for _, tx := range rec.texts {
				if tx.kind == "bigText" {
					titleBottom = math.Max(titleBottom, tx.y+tx.h+2*pxOff(tx.scale))
				}
			}
			for _, tx := range rec.texts {
				if tx.kind == "drawText" { // the small line(s) of the party card: margins too (v1.4.0)
					lines++
					if gap := tx.x - (hudX + hudW); gap < levelTitleGap {
						t.Errorf("%s %q: card line %q %.0f px from the HUD panel", lang, d.Title, tx.s, gap)
					}
					if gap := screenW - (hudX + hudW) - (tx.x + tx.w); gap < levelTitleGap {
						t.Errorf("%s %q: card line %q %.0f px from the mirror of the HUD panel", lang, d.Title, tx.s, gap)
					}
					if tx.y < 16 {
						t.Errorf("%s %q: card line %q at y %.0f, under 16 px from the top edge", lang, d.Title, tx.s, tx.y)
					}
					if titleBottom > 0 && tx.y < titleBottom+4 {
						t.Errorf("%s %q: card line %q at y %.0f touches the title (bottom %.0f)", lang, d.Title, tx.s, tx.y, titleBottom)
					}
					if tx.x != math.Round(tx.x) || tx.y != math.Round(tx.y) {
						t.Errorf("%s %q: card line not on whole pixels (%.2f, %.2f)", lang, d.Title, tx.x, tx.y)
					}
				}
				if tx.kind != "bigText" {
					continue
				}
				found = true
				off := pxOff(tx.scale)
				left, right := tx.x-off, tx.x+tx.w+2*off
				if gap := left - (hudX + hudW); gap < levelTitleGap {
					t.Errorf("%s %q: %.0f px from the HUD panel (scale %.0f)", lang, d.Title, gap, tx.scale)
				}
				if gap := screenW - (hudX + hudW) - right; gap < levelTitleGap {
					t.Errorf("%s %q: %.0f px from the mirror of the HUD panel on the right", lang, d.Title, gap)
				}
				if tx.x != math.Round(tx.x) || tx.scale != math.Round(tx.scale) {
					t.Errorf("%s %q: not on whole pixels (x %.2f, scale %.2f)", lang, d.Title, tx.x, tx.scale)
				}
			}
			if !found {
				t.Errorf("%s %q: no title drawn", lang, d.Title)
			}
			if want := len(g.levelCardLines()); lines != want {
				t.Errorf("%s %q: %d card lines drawn, want %d", lang, d.Title, lines, want)
			}
			if g.partyCardLine() != "" && g.levelTitleBottom() < titleBottom {
				t.Errorf("%s %q: levelTitleBottom %.0f above the title's bottom %.0f", lang, d.Title, g.levelTitleBottom(), titleBottom)
			}
		}
	}
	for _, s := range []string{guadalajaraTitle, "CASINO", "ITALIA"} {
		if g.levelTitleScale(s) != 4 {
			t.Errorf("%q should keep the big size", s)
		}
	}
	if g.levelTitleScale(alcalaTitle) >= 4 {
		t.Errorf("%q is too wide for the big size next to the HUD", alcalaTitle)
	}
}

// The DESCRIPTION box of every visible party (the first message, under the title and its card), in Spanish and in
// English, with and without lines of forces in the HUD: >= 12 px from the HUD panel (its real height) and from its
// mirror on the right, >= 16 px from the screen edges, the text inside its box (QA v1.4.0: in ES the box of the
// PARQUE DE ATRACCIONES went over the panel).
func TestLevelTitleClearOfHUDDescBox(t *testing.T) {
	snapshotLang(t)
	g := hudPlay(t, false)
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	prevLvl, prevT, prevMsgs := g.lvl, g.titleT, g.msgs
	t.Cleanup(func() { g.lvl, g.titleT, g.msgs = prevLvl, prevT, prevMsgs })
	far := func(ax, ay, aw, ah, bx, by, bw, bh, gap float64) bool {
		return ax >= bx+bw+gap || bx >= ax+aw+gap || ay >= by+bh+gap || by >= ay+ah+gap
	}
	checked, wrapped := 0, 0
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		for _, forces := range []bool{false, true} {
			g.cops = nil
			if forces {
				for _, k := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
					g.cops = append(g.cops, &Cop{kind: k, state: cChase})
				}
			}
			for i, d := range levelDefs {
				if campaignRetired(i) || d.Intro == "" {
					continue
				}
				g.lvl = &Level{Def: d}
				g.titleT = 1
				g.msgs = []HudMsg{{d.Intro, 4}, {"¡A bailar!", 4}}
				hudRecord(g) // sets g.hud.bottom with the real height of the panel
				panelH := g.hud.bottom - 3 - hudY
				textRec = &textRecorder{}
				g.drawMsgBoxes(img)
				rec := textRec
				textRec = nil
				if len(rec.panels) != len(g.msgs) {
					t.Errorf("%s %q: %d boxes drawn, want %d", lang, d.Title, len(rec.panels), len(g.msgs))
					continue
				}
				checked++
				for _, b := range g.msgBoxes() {
					if len(b.lines) > 1 {
						wrapped++
					}
					what := fmt.Sprintf("%s %q forces=%v box (%.0f,%.0f %.0fx%.0f)", lang, d.Title, forces, b.x, b.y, b.w, b.h)
					if !far(b.x, b.y, b.w, b.h, hudX, hudY, hudW, panelH, levelTitleGap) {
						t.Errorf("%s: within %.0f px of the HUD panel (bottom %.0f)", what, levelTitleGap, hudY+panelH)
					}
					if !far(b.x, b.y, b.w, b.h, screenW-hudX-hudW, hudY, hudW, panelH, levelTitleGap) {
						t.Errorf("%s: within %.0f px of the mirror of the HUD panel", what, levelTitleGap)
					}
					if b.x < 16 || b.y < 16 || b.x+b.w > screenW-16 || b.y+b.h > screenH-16 {
						t.Errorf("%s: under 16 px from the screen edge", what)
					}
					if b.y < g.levelTitleBottom()+levelTitleGap {
						t.Errorf("%s: touches the title card (bottom %.0f)", what, g.levelTitleBottom())
					}
				}
				for _, tx := range rec.texts {
					in := false
					for _, p := range rec.panels {
						in = in || (tx.x >= p.x+6 && tx.x+tx.w <= p.x+p.w-6 && tx.y >= p.y && tx.y < p.y+p.h)
					}
					if !in {
						t.Errorf("%s %q: text %q (%.0f..%.0f) not inside its box", lang, d.Title, tx.s, tx.x, tx.x+tx.w)
					}
				}
			}
		}
	}
	if checked < 4*20 {
		t.Errorf("only %d parties checked", checked)
	}
	t.Logf("%d parties checked, %d long descriptions split in lines", checked, wrapped)
}
