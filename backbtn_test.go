package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// backHW fakes the mouse and the focus key / pad for the back button; everything is put back by t.Cleanup.
type backHW struct {
	click  bool
	cx, cy int
	focus  bool
	padA   bool
}

func backFake(t *testing.T) *backHW {
	f := &backHW{cx: 400, cy: 300}
	oc, ok, of, oids, oj := backCursorFn, backClickFn, backFocusFn, padIDsFn, padJustFn
	backCursorFn = func() (int, int) { return f.cx, f.cy }
	backClickFn = func() bool { return f.click }
	backFocusFn = func() bool { return f.focus }
	padIDsFn = func() []ebiten.GamepadID { return []ebiten.GamepadID{7} }
	padJustFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool {
		return f.padA && b == ebiten.StandardGamepadButtonRightBottom
	}
	t.Cleanup(func() {
		backCursorFn, backClickFn, backFocusFn, padIDsFn, padJustFn = oc, ok, of, oids, oj
		backInject, backFocus = false, false
	})
	return f
}

// the menu modes of game.go and how each one is put on screen; ModeMenu, the level, the cinematics and the editor
// have no button (see the test below)
type backCase struct {
	name string
	mode Mode
	set  func(g *Game)
}

func backCases() []backCase {
	return []backCase{
		{"party list", ModeLevelSelect, func(g *Game) { g.menuSel = 3 }},
		{"settings", ModeSettings, func(g *Game) { g.setSel, g.setCol, g.capturing = 0, 0, false }},
		{"shop", ModeShop, func(g *Game) { g.shopTab, g.shopSel, g.shopScroll = 0, 0, 0 }},
		{"wardrobe", ModeShop, func(g *Game) { g.shopTab = shopWardrobe }},
		{"trophies", ModeTrophies, func(g *Game) { g.trophyTab = tabTrophies }},
		{"album", ModeTrophies, func(g *Game) { g.trophyTab = tabChars }},
		{"cinematics list", ModeScenes, func(g *Game) { g.sceneSel = 0 }},
		{"debug", ModeDebug, func(g *Game) { g.dbgSel = 0 }},
		{"erase popup", ModeConfirm, func(g *Game) { g.confirmT, g.confirmSel = 0, 0 }},
	}
}

func backGame(t *testing.T, c backCase) *Game {
	g := auditGame(t)
	auditReset(g)
	t.Cleanup(func() {
		auditReset(g)
		g.mode, g.shopMsg, g.shopTab, g.shopSel = ModeMenu, "", 0, 0
		// the "debug" case presses A on the first row of the DEBUG screen: that switches the immunity on
		// (g.dbg.immune), freezes and replaces the progress... and die() / lose() never fire again for the next tests
		// that share this game (TestTrophyDieWithOneLeft, the RETO DE HOY tests). End the session like the menu does.
		if debugSnap != nil {
			g.debugEnd()
		}
		g.dbg, g.powerImmune = debugState{}, false
	})
	g.mode = c.mode
	g.showCredits, g.capturing = false, false
	c.set(g)
	return g
}

func drawnBack(t *testing.T, g *Game) (found bool, r textRect) {
	t.Helper()
	textRec = &textRecorder{}
	defer func() { textRec = nil }()
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
	for _, x := range textRec.texts {
		if x.s == "← ATRÁS" || x.s == "← BACK" {
			return true, x
		}
	}
	return false, textRect{}
}

// every menu screen draws the button at the same place, in both languages, and the main menu / the level do not
func TestMenuBackButtonDrawnEverywhere(t *testing.T) {
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu })
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang = lang, lang
		for _, c := range backCases() {
			g := backGame(t, c)
			ok, r := drawnBack(t, g)
			if !ok {
				t.Errorf("[%s] %s: no back button", lang, c.name)
				continue
			}
			want := "← ATRÁS"
			if lang == "en" {
				want = "← BACK"
			}
			if r.s != want {
				t.Errorf("[%s] %s: text %q, want %q", lang, c.name, r.s, want)
			}
			if r.scale != 1 || r.x != float64(int(r.x)) || r.y != float64(int(r.y)) {
				t.Errorf("[%s] %s: text not on whole pixels / scale 1: %+v", lang, c.name, r)
			}
			if r.x < backBtnX+4 || r.x+r.w > backBtnX+backBtnW-4 || r.y+3 < backBtnY || r.y+r.h-3 > backBtnY+backBtnH {
				t.Errorf("[%s] %s: text out of the button: %+v", lang, c.name, r)
			}
		}
		for _, m := range []Mode{ModeMenu, ModePlay} {
			g := auditGame(t)
			auditReset(g)
			g.mode = m
			if m == ModePlay {
				auditPlay(g, false)
			}
			if ok, _ := drawnBack(t, g); ok {
				t.Errorf("[%s] mode %d must not show the back button", lang, m)
			}
			g.mode = ModeMenu
		}
	}
}

// the mouse on the button does what Esc does on each screen; a click elsewhere does nothing
func TestMenuBackButtonClickGoesBack(t *testing.T) {
	for _, c := range backCases() {
		g := backGame(t, c)
		f := backFake(t)
		f.click, f.cx, f.cy = true, 2, 2 // a click on the empty corner of the screen, not on the button
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.mode != c.mode {
			t.Errorf("%s: a click elsewhere left mode %d", c.name, g.mode)
		}
		g.mode = c.mode
		c.set(g)
		f.cx, f.cy = int(backBtnX)+3, int(backBtnY)+3
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.mode != ModeMenu {
			t.Errorf("%s: the button left mode %d, want the main menu", c.name, g.mode)
		}
	}
}

// the credits (opened from the main menu) close back to the menu, like Esc; the button works while the settings
// wait for a key
func TestMenuBackButtonCreditsAndCapture(t *testing.T) {
	g := backGame(t, backCases()[1])
	f := backFake(t)
	g.showCredits = true
	f.click, f.cx, f.cy = true, int(backBtnX)+3, int(backBtnY)+3
	g.Update()
	if g.mode != ModeMenu || g.showCredits {
		t.Fatalf("credits: mode %d, credits open %v; want the main menu with the credits closed", g.mode, g.showCredits)
	}
	g.mode, g.capturing = ModeSettings, true
	g.Update()
	if g.capturing || g.mode != ModeSettings {
		t.Fatalf("while a key is awaited the button cancels it (capturing %v, mode %d)", g.capturing, g.mode)
	}
}

// keyboard / pad: the focus reaches the button and A / Enter presses it, not the row of the screen
func TestMenuBackButtonFocusWithPad(t *testing.T) {
	for _, c := range backCases() {
		g := backGame(t, c)
		f := backFake(t)
		started := false
		if c.mode == ModeLevelSelect {
			g.menuSel = 0 // A on the first party would start it
		}
		f.focus = true
		g.Update()
		if !backFocus {
			t.Errorf("%s: LB / Backspace did not focus the button", c.name)
			continue
		}
		f.focus = false
		f.padA = true
		g.Update()
		if g.mode != ModeMenu {
			t.Errorf("%s: A on the focused button left mode %d, want the main menu (started=%v)", c.name, g.mode, started)
		}
		if backFocus {
			t.Errorf("%s: focus kept after pressing", c.name)
		}
	}
}

func TestMenuBackButtonModesListed(t *testing.T) {
	// every Mode of game.go is either in the table or knowingly without button
	without := map[Mode]string{ModeMenu: "main menu", ModePlay: "level", ModeLevelDone: "newspaper", ModeLose: "game over", ModeFinal: "final", ModeEditor: "editor has Volver al juego", ModeEnding: "cinematic", ModeIntro: "cinematic", ModeInterlude: "cinematic", ModeTitle: "title"}
	with := map[Mode]bool{}
	for _, c := range backCases() {
		with[c.mode] = true
	}
	for m := ModeMenu; m <= ModeTitle; m++ {
		_, no := without[m]
		if with[m] == no || with[m] != hasBackButton(m) {
			t.Errorf("mode %d is not classified (button %v, table %v)", m, hasBackButton(m), with[m])
		}
	}
}
