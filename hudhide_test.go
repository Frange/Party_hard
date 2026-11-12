package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// hudShown: does the HUD panel write anything now (alpha > 0)?
func hudShown(g *Game) bool { return len(hudRecord(g).texts) > 0 }

// hudSettle runs the HUD animation for sec seconds.
func hudSettle(g *Game, sec float64) {
	for t := 0.0; t < sec; t += 1.0 / 60 {
		g.updateHUD(1.0 / 60)
	}
}

// hudWalkOver draws the player with the feet at screen (sx, sy) the way the world does (drawPlayerOne, armed).
func hudWalkOver(g *Game, sx, sy float64) {
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.camI = Vec{}
	p, prev := g.p, g.p.pos
	defer func() { p.pos = prev }()
	p.pos = Vec{sx / wk, sy / wk}
	g.hudPlayerDraw(func() { g.drawPlayerOne(img, p, g.lvl) })
}

func TestHUDHidesUnderACharacter(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		hudSettle(g, 1)
		if !hudShown(g) {
			t.Fatalf("two=%v: the HUD is not shown with nobody over it", two)
		}
		// a character far from the panel does not hide it
		hudWalkOver(g, 600, 400)
		hudSettle(g, 1)
		if !hudShown(g) {
			t.Errorf("two=%v: a character far away hid the HUD", two)
		}
		// feet 40 px under the top: the sprite covers the panel
		hudWalkOver(g, 100, 40)
		g.updateHUD(1.0 / 60)
		if g.hudAlpha() >= 1 {
			t.Errorf("two=%v: the fade did not start at once", two)
		}
		hudSettle(g, hudFadeT+0.05)
		if g.hudAlpha() != 0 || hudShown(g) {
			t.Errorf("two=%v: alpha %.2f: the HUD is still drawn over a character", two, g.hudAlpha())
		}
		// it leaves: the HUD returns only after the hold time, with a fade
		hudSettle(g, hudHoldT*0.1)
		if hudShown(g) {
			t.Errorf("two=%v: the HUD came back before the minimum time", two)
		}
		hudSettle(g, hudHoldT+hudFadeT+0.1)
		if g.hudAlpha() != 1 || !hudShown(g) {
			t.Errorf("two=%v: the HUD did not come back (alpha %.2f)", two, g.hudAlpha())
		}
	}
}

// A character walking along the edge in and out does not make the HUD flicker: once hidden it stays hidden.
func TestHUDHysteresis(t *testing.T) {
	g := hudPlay(t, false)
	hudSettle(g, 1)
	flips, prev := 0, g.hudAlpha()
	for i := 0; i < 120; i++ {
		if i%6 < 3 { // 0.05 s in, 0.05 s out
			hudWalkOver(g, 100, 40)
		}
		g.updateHUD(1.0 / 60)
		a := g.hudAlpha()
		if i > 20 && a > prev {
			flips++
		}
		prev = a
	}
	if flips != 0 || g.hudAlpha() != 0 {
		t.Errorf("the HUD flickered: %d rises, alpha %.2f", flips, g.hudAlpha())
	}
}

// The hide box is the real panel plus 6 px, in both languages, with long figures and many forces.
func TestHUDHideRectMatchesPanel(t *testing.T) {
	snapshotLang(t)
	for _, lang := range []string{"es", "en"} {
		for _, forces := range []int{0, 5} {
			g := hudPlay(t, true)
			gameLang, editorLang = lang, lang
			g.challenge = true
			g.time = 6249
			g.points, g.hud.pts = 9999999, 9999999
			for _, k := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers}[:forces] {
				g.cops = append(g.cops, &Cop{kind: k, state: cChase})
			}
			rec := hudRecord(g)
			x0, y0, x1, y1 := g.hudHideRect()
			if x0 != hudX-6 || y0 != hudY-6 || x1 != hudX+hudW+6 {
				t.Errorf("%s: rect %v,%v,%v", lang, x0, y0, x1)
			}
			if want := g.hud.bottom - 3 + 6; y1 != want {
				t.Errorf("%s/%d: bottom %.0f, want %.0f", lang, forces, y1, want)
			}
			minX, minY, maxX, maxY := 1e9, 1e9, -1e9, -1e9
			for _, tx := range rec.texts {
				minX, minY = min(minX, tx.x), min(minY, tx.y)
				maxX, maxY = max(maxX, tx.x+tx.w), max(maxY, tx.y+tx.h)
			}
			if minX < x0+6 || minY < y0+6 || maxX > x1-6 || maxY > y1-6 {
				t.Errorf("%s/%d: the texts %.0f,%.0f-%.0f,%.0f leave the panel box %.0f,%.0f-%.0f,%.0f", lang, forces, minX, minY, maxX, maxY, x0+6, y0+6, x1-6, y1-6)
			}
		}
	}
}

// Props are not characters: nothing else hides the HUD; the ambulance line follows the HUD.
func TestHUDHideAmbulanceFollows(t *testing.T) {
	g := reviveGame()
	g.hud.reset(g)
	rec := func() int {
		textRec = &textRecorder{}
		defer func() { textRec = nil }()
		img := ebiten.NewImage(screenW, screenH)
		defer img.Deallocate()
		g.hud.bottom = hudY + hudBaseH + 3
		g.drawReviveHud(img)
		n := 0
		for _, tx := range textRec.texts {
			if !strings.Contains(tx.s, "AMBULANCIA") && !strings.Contains(tx.s, "[") { // the label over the phone belongs to the map
				n++
			}
		}
		return n
	}
	if rec() == 0 {
		t.Skip("no ambulance line to draw")
	}
	g.hud.hide = 1
	if n := rec(); n != 0 {
		t.Errorf("the ambulance line is drawn (%d texts) with the HUD hidden", n)
	}
	kill(g, g.p2, "Un asesino te ha dado caza")
	if n := rec(); n != 0 {
		t.Errorf("the ambulance notice is drawn (%d texts) with the HUD hidden", n)
	}
	g.hud.hide = 0
	if rec() == 0 {
		t.Errorf("the notice is not drawn with the HUD shown")
	}
}

// Outside the play mode (menus, scenes) a character does not touch the HUD state.
func TestHUDTouchOnlyInPlay(t *testing.T) {
	g := hudPlay(t, false)
	g.mode = ModeScenes
	g.hudTouch(0, 0, 500, 500)
	if g.hud.hit {
		t.Error("a character outside the play mode hid the HUD")
	}
}
