package main

import (
	"encoding/json"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// fake input of the pause window; put back by t.Cleanup
type pauseFake struct{ in pauseInput }

func pauseHW(t *testing.T) *pauseFake {
	f := &pauseFake{in: pauseInput{mx: 5, my: 5}}
	old, oldNow, oldFrozen := pauseInFn, pauseNow, progressFrozen
	pauseInFn = func() pauseInput { return f.in }
	progressFrozen = true // nothing here may write the real progress.json
	t.Cleanup(func() { pauseInFn, pauseNow, progressFrozen = old, oldNow, oldFrozen })
	return f
}

// press: one frame with only that input, then the input is released.
func (f *pauseFake) press(g *Game, in pauseInput) {
	if in.mx == 0 && in.my == 0 {
		in.mx, in.my = f.in.mx, f.in.my
	}
	f.in = in
	if err := g.Update(); err != nil {
		panic(err)
	}
	f.in = pauseInput{mx: in.mx, my: in.my}
}

func pauseGame(t *testing.T, two bool) (*Game, *pauseFake) {
	g := helpGame(t, two)
	g.fromEditor = false
	g.pause = pauseState{}
	t.Cleanup(func() { g.pause = pauseState{} })
	return g, pauseHW(t)
}

func idle(g *Game, n int) {
	for i := 0; i < n; i++ {
		if err := g.Update(); err != nil {
			panic(err)
		}
	}
}

// Esc opens the window and freezes the party; Esc again resumes it, and the party moves again.
func TestPauseEscFreezesAndResumes(t *testing.T) {
	for _, two := range []bool{false, true} {
		g, f := pauseGame(t, two)
		if two && g.p2 == nil {
			t.Fatal("two players: no player 2")
		}
		idle(g, 2)
		f.press(g, pauseInput{toggle: true})
		if !g.pause.open || g.mode != ModePlay {
			t.Fatalf("two=%v: Esc did not open the pause window (mode %d)", two, g.mode)
		}
		before := helpSnap(g)
		idle(g, 180) // three seconds paused
		if helpSnap(g) != before {
			t.Fatalf("two=%v: the party moved while paused", two)
		}
		f.press(g, pauseInput{toggle: true})
		if g.pause.open || g.mode != ModePlay {
			t.Fatalf("two=%v: Esc again did not resume (open=%v mode=%d)", two, g.pause.open, g.mode)
		}
		t0 := g.time
		idle(g, 10)
		if g.time <= t0 {
			t.Fatalf("two=%v: the party clock does not run after resuming", two)
		}
	}
}

// CONTINUAR (keyboard and mouse) resumes; arrows wrap around the three rows.
func TestPauseContinueAndNavigation(t *testing.T) {
	g, f := pauseGame(t, false)
	f.press(g, pauseInput{toggle: true})
	f.press(g, pauseInput{up: true})
	if g.pause.sel != pauseRowQuit {
		t.Fatalf("up from the first row should wrap to the last, sel=%d", g.pause.sel)
	}
	f.press(g, pauseInput{down: true})
	if g.pause.sel != pauseRowContinue {
		t.Fatalf("down should wrap back to CONTINUAR, sel=%d", g.pause.sel)
	}
	f.press(g, pauseInput{ok: true})
	if g.pause.open || g.mode != ModePlay {
		t.Fatal("CONTINUAR (Enter) did not resume")
	}
	// mouse: hovering a row selects it, a click on CONTINUAR resumes
	f.press(g, pauseInput{toggle: true})
	y := int(pauseRowY0+2*pauseRowDY) + 5
	f.press(g, pauseInput{mx: int(pauseRowX) + 20, my: y})
	if g.pause.sel != pauseRowQuit || !g.pause.open {
		t.Fatalf("hover did not select the row under the cursor, sel=%d", g.pause.sel)
	}
	f.press(g, pauseInput{mx: int(pauseRowX) + 20, my: int(pauseRowY0) + 5, click: true})
	if g.pause.open || g.mode != ModePlay {
		t.Fatal("a click on CONTINUAR did not resume")
	}
}

// AJUSTES opens the settings; while there the party clock is held, and their ATRÁS comes back to the
// pause window (not the main menu) with the same party loaded.
func TestPauseSettingsComesBackToPause(t *testing.T) {
	g, f := pauseGame(t, false)
	ob, oc := backClickFn, backCursorFn
	t.Cleanup(func() { backClickFn, backCursorFn = ob, oc })
	click := false
	backClickFn = func() bool { return click }
	backCursorFn = func() (int, int) { return int(backBtnX) + 4, int(backBtnY) + 4 }
	idle(g, 3)
	lvl := g.lvl
	f.press(g, pauseInput{toggle: true})
	f.press(g, pauseInput{down: true})
	f.press(g, pauseInput{ok: true})
	if g.mode != ModeSettings || !g.pause.inSet {
		t.Fatalf("AJUSTES did not open the settings (mode %d)", g.mode)
	}
	before := helpSnap(g)
	idle(g, 120)
	if helpSnap(g) != before {
		t.Fatalf("the party moved (or its clock ran) while the settings were open\n%.200s\n%.200s", before, helpSnap(g))
	}
	click = true // the ATRÁS button of the settings
	idle(g, 1)
	click = false
	if g.mode != ModePlay || !g.pause.open || g.pause.inSet {
		t.Fatalf("leaving the settings did not come back to the pause window (mode %d open %v)", g.mode, g.pause.open)
	}
	if g.lvl != lvl || helpSnap(g) != before {
		t.Fatal("the party was lost or moved after the settings")
	}
	idle(g, 30)
	if helpSnap(g) != before {
		t.Fatal("the party is not paused any more after the settings")
	}
}

// SALIR A ELEGIR FIESTA: the party list opens and nothing is added to the progress (no points banked, no
// defeat, RETO DE HOY not locked). One and two players.
func TestPauseQuitToPartyListWritesNothing(t *testing.T) {
	for _, two := range []bool{false, true} {
		for _, daily := range []bool{false, true} {
			g, f := pauseGame(t, two)
			oldUnl := g.progress.Unlocked
			t.Cleanup(func() { g.progress.Unlocked = oldUnl; g.dc, g.dcArm, g.challenge = nil, nil, false })
			if daily {
				g.startDaily(true, 0)
				g.lvl.graded = true
				g.titleT = 0
				if g.dc == nil {
					t.Fatal("the RETO DE HOY did not start")
				}
			}
			g.progress.Unlocked = 5
			g.levelIdx = 3
			g.points = 777 // points of the party being abandoned
			snap, _ := json.Marshal(g.progress)
			f.press(g, pauseInput{toggle: true})
			f.press(g, pauseInput{up: true}) // the last row
			f.press(g, pauseInput{ok: true})
			if g.mode != ModeLevelSelect || g.pause.open {
				t.Fatalf("two=%v daily=%v: SALIR did not open the party list (mode %d)", two, daily, g.mode)
			}
			if g.menuSel != 3 {
				t.Fatalf("the party list should point at the abandoned party, sel=%d", g.menuSel)
			}
			idle(g, 3) // the party list frames (it banks nothing either)
			after, _ := json.Marshal(g.progress)
			if string(after) != string(snap) {
				t.Fatalf("two=%v daily=%v: leaving the party changed the progress:\n%s\n%s", two, daily, snap, after)
			}
			if daily && g.dailyState() != dailyOpen {
				t.Fatal("leaving the RETO DE HOY locked it")
			}
			g.mode = ModePlay
		}
	}
}

// Start of any pad opens it (with two players any of them can); it never opens outside an active party,
// in a test run from the editor (Esc goes back to the editor there) or over the help window.
func TestPauseOnlyInAParty(t *testing.T) {
	g, _ := pauseGame(t, true)
	pauseInFn = func() pauseInput { return pauseInput{toggle: pauseStartPressed()} } // the real Start reading
	oids, oj := padIDsFn, padJustFn
	t.Cleanup(func() { padIDsFn, padJustFn = oids, oj })
	start := false
	padIDsFn = func() []ebiten.GamepadID { return []ebiten.GamepadID{1, 2} }
	padJustFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool {
		return start && id == 2 && b == ebiten.StandardGamepadButtonCenterRight
	}
	start = true
	if !g.updatePause() || !g.pause.open {
		t.Fatal("Start of the second pad did not open the pause window")
	}
	g.pause = pauseState{}
	for _, m := range []Mode{ModeMenu, ModeLevelDone, ModeLose, ModeEnding, ModeIntro, ModeInterlude, ModeTitle, ModeLevelSelect, ModeSettings} {
		g.mode = m
		if g.updatePause() || g.pause.open {
			t.Fatalf("opened in mode %d", m)
		}
	}
	g.mode = ModePlay
	g.fromEditor = true
	if g.updatePause() {
		t.Fatal("opened in a test run from the editor")
	}
	g.fromEditor = false
	g.helpOv = true
	if g.updatePause() {
		t.Fatal("opened over the help window")
	}
	g.helpOv = false
	start = false
	pauseNow = true // -pausenow
	g.time = 1
	if !g.updatePause() || !g.pause.open || pauseNow {
		t.Fatal("-pausenow did not open the window")
	}
}
