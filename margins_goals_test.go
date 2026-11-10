package main

import (
	"fmt"
	"testing"
)

// Margins of the OBJETIVOS screen, its detail window, the DEBUG menu, the HELP window and the in-game HUD
// (the user's rule: always room; >= 12 px between blocks, >= 16 px to the screen edge, >= 6 px of padding in a box).
type mrect struct {
	name       string
	x, y, w, h float64
}

func (a mrect) gap(b mrect) float64 { // the free distance between two rectangles (negative: they overlap)
	dx := maxF(b.x-(a.x+a.w), a.x-(b.x+b.w))
	dy := maxF(b.y-(a.y+a.h), a.y-(b.y+b.h))
	return maxF(dx, dy)
}

func (a mrect) inset(b mrect) float64 { // how far a is inside b (the smallest of the four distances)
	return minF(minF(a.x-b.x, a.y-b.y), minF(b.x+b.w-(a.x+a.w), b.y+b.h-(a.y+a.h)))
}

func mustGap(t *testing.T, label string, a, b mrect, min float64) {
	t.Helper()
	if g := a.gap(b); g < min {
		t.Errorf("%s: %s %v and %s %v are %.0f px apart (>= %.0f needed)", label, a.name, a, b.name, b, g, min)
	}
}

func mustInside(t *testing.T, label string, a, b mrect, min float64) {
	t.Helper()
	if d := a.inset(b); d < min {
		t.Errorf("%s: %s %v is %.0f px from the edge of %s (>= %.0f needed)", label, a.name, a, d, b.name, min)
	}
}

func mtext(g *Game, name, s string, x, y, scale float64) mrect {
	return mrect{name, x, y, g.textW(s, scale), 8 * scale}
}

func TestMarginsGoalsScreen(t *testing.T) {
	g := goalDetailGame(t)
	screen := mrect{"screen", 0, 0, screenW, screenH}
	panel := mrect{"panel", 230, 56, 700, 440}
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		tabs := g.goalTabRects()
		title := mtext(g, "title", "OBJETIVOS", 244, 66, 1.6)
		counter := mtext(g, "counter", "9999 / 9999", 770, 66, 1.6)
		cards := mrect{"cards", albumX0, albumY0, 5*albumCardW + 4*albumGapX, albumRows*albumCardH + (albumRows-1)*albumGapY}
		list := mrect{"list", 244, 123, 356, trophyRows*30 - 6}
		detail := mrect{"detail", 640, 123, 270, 322}
		hint := mtext(g, "hint", tr("Los premios se usan desde la TIENDA, en su lista (marcados con TROFEO)."), 244, 458, 1)
		selLine := mrect{"selected card line", 244, 459, 676, 10}
		for i, r := range tabs {
			tb := mrect{fmt.Sprintf("tab%d", i), r[0], r[1], r[2], r[3]}
			l := lang + " " + tb.name
			mustInside(t, l, tb, panel, 6)
			mustGap(t, l, tb, title, 12)
			mustGap(t, l, tb, cards, 12)
			mustGap(t, l, tb, list, 12)
			mustGap(t, l, tb, detail, 12)
			if i > 0 {
				p := tabs[i-1]
				mustGap(t, l, tb, mrect{"prev tab", p[0], p[1], p[2], p[3]}, 4)
			}
			mustGap(t, l, tb, counter, 12)
		}
		mustInside(t, lang, title, panel, 6)
		mustInside(t, lang, counter, panel, 6)
		mustGap(t, lang, title, counter, 12)
		mustGap(t, lang, title, cards, 12)
		mustInside(t, lang, cards, panel, 6)
		mustInside(t, lang, mrect{"scroll bar", 920, albumY0, 4, cards.h}, panel, 6)
		mustGap(t, lang, cards, selLine, 12)
		mustInside(t, lang, selLine, panel, 6)
		mustGap(t, lang, list, detail, 12)
		mustGap(t, lang, list, hint, 12)
		mustGap(t, lang, detail, hint, 12)
		mustInside(t, lang, hint, panel, 6)
		mustInside(t, lang, panel, screen, 16) // (the panel is the 700x440 frame of the screen, 16+ px from every edge)
		// the escapes line (POLICÍAS) stays right of the last tab and inside the panel
		line := tr(g.copEscapeLine())
		w := g.textW(line, 1)
		esc := mrect{"escapes line", 914 - w, goalTabY + 6, w, 8}
		mustInside(t, lang, esc, panel, 6)
		mustGap(t, lang, esc, mrect{"last tab", tabs[tabCount-1][0], tabs[tabCount-1][1], tabs[tabCount-1][2], tabs[tabCount-1][3]}, 12)
		// CARRERA: the table and its footer
		table := mrect{"career table", careerLX0 - 6, careerY0 - 5, careerRX1 + 6 - (careerLX0 - 6), 8*careerPitch + 32}
		for i, r := range tabs {
			mustGap(t, lang+" career", mrect{fmt.Sprintf("tab%d", i), r[0], r[1], r[2], r[3]}, table, 12)
		}
		since := mtext(g, "career footer", careerSinceLine(g.progress.Started), 244, 452, 1)
		mustGap(t, lang, table, since, 12)
		mustInside(t, lang, table, panel, 6)
		mustInside(t, lang, since, panel, 6)
	}
}

// The detail window: the X, the arrows, the picture and the text column keep room from the frame, the header line and
// each other.
func TestMarginsGoalDetailWindow(t *testing.T) {
	win := mrect{"window", gdX, gdY, gdW, gdH}
	head := mrect{"header", gdX, gdY, gdW, 33}
	x := mrect{"X", gdCloseX, gdCloseY, gdCloseS, gdCloseS}
	prev := mrect{"prev", gdPrevX, gdArrowY, gdArrowW, gdArrowH}
	next := mrect{"next", gdNextX, gdArrowY, gdArrowW, gdArrowH}
	pic := mrect{"picture", gdPicX, gdPicY, gdPicW, gdPicH}
	col := mrect{"text column", gdTextX, gdTextTop, gdTextW, gdTextBottom - gdTextTop}
	for _, a := range []mrect{x, prev, next, pic, col} {
		mustInside(t, "detail", a, win, 6)
	}
	mustInside(t, "detail", x, mrect{"header", gdX, gdY, gdW, 32}, 6)
	for _, a := range []mrect{prev, next, pic, col} {
		mustGap(t, "detail", a, head, 6)
	}
	mustGap(t, "detail", pic, mrect{"header line", gdX, gdY + 32, gdW, 1}, 12)
	mustGap(t, "detail", col, mrect{"header line", gdX, gdY + 32, gdW, 1}, 12)
	for i, a := range []mrect{prev, pic, col, next} {
		for _, b := range []mrect{prev, pic, col, next}[i+1:] {
			mustGap(t, "detail", a, b, 12)
		}
	}
	mustGap(t, "detail", win, mrect{"back button", backBtnX, backBtnY, backBtnW, backBtnH}, 0)
	mustInside(t, "detail", win, mrect{"screen", 0, 0, screenW, screenH}, 16)
	// the in-game window's own X (trophyhud.go) too
	tx := mrect{"in-game X", trophyCloseX, trophyCloseY - 4, 22, 22}
	mustInside(t, "trophy window", tx, win, 6)
	mustGap(t, "trophy window", tx, mrect{"counter", 770, 66, 100, 13}, 12)
}

// The DEBUG menu: title, subtitle, rows and status line never touch each other nor the frame.
func TestMarginsDebugMenu(t *testing.T) {
	g := textGame()
	frame := mrect{"frame", 132, 28, 696, 504}
	rows := g.debugRows()
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		title := mtext(g, "title", tr("MODO DEBUG"), 480, 36, 2.2)
		title.x -= title.w / 2
		sub := mtext(g, "subtitle", tr("Todo es temporal: no se guarda nada y tu partida real no cambia."), 0, 70, 1.1)
		sub.x = 480 - sub.w/2
		list := mrect{"rows", 140, debugRow0 - 3, 680, float64(len(rows)) * debugRowH}
		status := mtext(g, "status", tr("SESIÓN DEBUG ACTIVA · nada se guarda hasta que salgas"), 0, 498, 1.1)
		status.x = 480 - status.w/2
		mustGap(t, lang, title, sub, 12)
		mustGap(t, lang, sub, list, 12)
		mustGap(t, lang, list, status, 12)
		for _, a := range []mrect{title, sub, status} {
			mustInside(t, lang, a, frame, 6)
		}
		mustInside(t, lang, list, frame, 6)
		mustGap(t, lang, frame, mrect{"back button", backBtnX, backBtnY, backBtnW, backBtnH}, 12)
	}
}

// The HELP window: the three columns, the modes block and the footer are 12 px apart; the footer is inside the frame.
func TestMarginsHelpWindow(t *testing.T) {
	frame := mrect{"frame", helpPanX, helpPanY, helpPanW, helpPanH}
	cols := mrect{"columns", helpPanX + 20, helpColsY, helpPanW - 40, 14 + 10 + 6*helpRowPitch + helpCapH}
	modes := mrect{"modes", helpPanX + 20, helpModesY0, helpPanW - 40, 28 + float64(len(helpModes))*helpModeRowH + helpCapH}
	foot := mrect{"footer", helpPanX + 20, helpFooterY, helpPanW - 40, 8}
	sub := mrect{"subtitle", helpPanX + 20, helpPanY + 38, 200, 8}
	mustGap(t, "help", sub, cols, 12)
	mustGap(t, "help", cols, modes, 11) // the rule line is the top of the modes block
	mustGap(t, "help", modes, foot, 12)
	mustInside(t, "help", foot, frame, 6)
	mustGap(t, "help", mrect{"close", helpCloseX, helpCloseY, helpCloseW, helpCloseH}, mrect{"header", helpPanX + 20, helpPanY + 10, 330, 16}, 12)
	if helpRowPitch-helpCapH < 4 || helpModeRowH-helpCapH < 4 {
		t.Errorf("the key rows touch: pitch %.0f/%.0f, key %.0f high", helpRowPitch, helpModeRowH, helpCapH)
	}
}

// The HUD: the panel stands 16 px from the screen edges, and the notes stacked under it (OBJETO, stamina, ambulance,
// jail) never touch the panel nor each other, in 1 and 2 players.
func TestMarginsHUDAndSideNotes(t *testing.T) {
	if hudX < 16 || hudY < 16 {
		t.Errorf("the HUD panel is at %.0f,%.0f: less than 16 px from the edge", hudX, hudY)
	}
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		for _, pl := range g.players() {
			pl.item, pl.stamina = "poison", 2
		}
		hudRecord(g)
		panelBottom := g.hud.bottom - 3
		o, s, r, j := g.hudSide()
		lines := 1.0
		if two {
			lines = 2
		}
		blocks := []mrect{{"OBJETO", hudX, o, 120, 8 + 12*(lines-1)}, {"stamina", hudX, s, 70, 4 + 6*(lines-1)}}
		if two {
			blocks = append(blocks, mrect{"ambulance", hudX, r, 120, 8}, mrect{"jail", hudX, j, 120, 8})
		}
		mustGap(t, fmt.Sprint("hud 2P=", two), blocks[0], mrect{"panel", hudX, hudY, hudW, panelBottom - hudY}, 12)
		for i, a := range blocks {
			for _, b := range blocks[i+1:] {
				mustGap(t, fmt.Sprint("hud 2P=", two), a, b, 6)
			}
		}
	}
}
