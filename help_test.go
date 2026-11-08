package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// fake hardware for the help window; everything is put back by t.Cleanup
type helpFake struct {
	keys    map[ebiten.Key]bool
	click   bool
	cx, cy  int
	selectP bool
}

func helpHW(t *testing.T) *helpFake {
	f := &helpFake{keys: map[ebiten.Key]bool{}, cx: 400, cy: 300}
	ok, oc, op, oids, oj := helpKeyFn, helpClickFn, helpCursorFn, padIDsFn, padJustFn
	helpKeyFn = func(k ebiten.Key) bool { return f.keys[k] }
	helpClickFn = func() bool { return f.click }
	helpCursorFn = func() (int, int) { return f.cx, f.cy }
	padIDsFn = func() []ebiten.GamepadID { return []ebiten.GamepadID{3} }
	padJustFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool {
		return f.selectP && b == ebiten.StandardGamepadButtonCenterLeft
	}
	t.Cleanup(func() { helpKeyFn, helpClickFn, helpCursorFn, padIDsFn, padJustFn = ok, oc, op, oids, oj })
	return f
}

func helpGame(t *testing.T, two bool) *Game {
	g := layerGame(t, two)
	g.loadLevel(0)
	g.lvl.graded = true // gradeBG reads pixels, which cannot be done without a window
	g.titleT = 0
	g.mode = ModePlay
	g.helpOv = false
	t.Cleanup(func() { g.helpOv = false })
	return g
}

// snapshot of what must not move while the window is open
func helpSnap(g *Game) string {
	var sb strings.Builder
	sb.WriteString(strings.Join([]string{ftoa(g.time), ftoa(g.p.pos.X), ftoa(g.p.pos.Y), ftoa(g.wanted)}, "|"))
	for _, gu := range g.guests {
		sb.WriteString("|" + ftoa(gu.pos.X) + "," + ftoa(gu.pos.Y) + "," + ftoa(gu.timer))
	}
	return sb.String()
}

func ftoa(f float64) string { return fmt.Sprintf("%.4f", f) }

func TestHelpOpensWithButtonF1AndSelect(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := helpGame(t, two)
		f := helpHW(t)
		if g.updateHelp() || g.helpOv {
			t.Fatal("opened with nothing pressed")
		}
		f.keys[ebiten.KeyF1] = true
		if !g.updateHelp() || !g.helpOv {
			t.Fatal("F1 did not open")
		}
		g.helpOv = false
		f.keys[ebiten.KeyF1] = false
		f.click, f.cx, f.cy = true, 10, 10 // a click elsewhere does nothing
		if g.updateHelp() {
			t.Fatal("a click away from the button opened it")
		}
		f.cx, f.cy = int(helpBtnX)+2, int(helpBtnY)+2
		if !g.updateHelp() || !g.helpOv {
			t.Fatal("the button did not open")
		}
		g.helpOv, f.click = false, false
		f.selectP = true
		if !g.updateHelp() || !g.helpOv {
			t.Fatal("Select of the pad did not open")
		}
	}
}

func TestHelpClosesAndTheGameIsPaused(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := helpGame(t, two)
		f := helpHW(t)
		before := helpSnap(g)
		g.helpOv = true
		for i := 0; i < 120; i++ { // two seconds of frames with the window open
			if !g.updateHelp() {
				t.Fatal("the window closed by itself")
			}
		}
		if helpSnap(g) != before {
			t.Fatal("the game moved with the window open")
		}
		// every way to close it
		closers := map[string]func(){
			"esc":     func() { f.keys[ebiten.KeyEscape] = true },
			"f1":      func() { f.keys[ebiten.KeyF1] = true },
			"select":  func() { f.selectP = true },
			"button":  func() { f.click, f.cx, f.cy = true, int(helpCloseX)+4, int(helpCloseY)+4 },
			"outside": func() { f.click, f.cx, f.cy = true, 2, 2 },
		}
		for name, press := range closers {
			g.helpOv = true
			f.keys, f.click, f.selectP = map[ebiten.Key]bool{}, false, false
			f.cx, f.cy = 400, 300
			if !g.updateHelp() || !g.helpOv {
				t.Fatalf("%s: closed with nothing pressed (or the frame was not skipped)", name)
			}
			f.click, f.cx, f.cy = true, 400, 300 // a click inside the panel keeps it open
			g.updateHelp()
			if !g.helpOv {
				t.Fatalf("%s: a click inside the window closed it", name)
			}
			f.click = false
			press()
			if !g.updateHelp() {
				t.Fatalf("%s: the closing frame must still be skipped (the key is consumed)", name)
			}
			if g.helpOv {
				t.Fatalf("%s: did not close", name)
			}
		}
		if helpSnap(g) != before {
			t.Fatal("state differs after closing")
		}
	}
}

// The window reads no game input: with it open, Esc does not leave the level and R does not restart it.
func TestHelpDoesNotFeedTheGame(t *testing.T) {
	g := helpGame(t, false)
	helpHW(t)
	g.helpOv = true
	mode, idx := g.mode, g.levelIdx
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.mode != mode || g.levelIdx != idx || !g.helpOv {
		t.Fatal("Update with the window open changed the game")
	}
}

func TestHelpOnlyInMenuAndLevels(t *testing.T) {
	g := helpGame(t, false)
	for m, want := range map[Mode]bool{ModeMenu: true, ModePlay: true, ModeShop: false, ModeSettings: false, ModeEnding: false, ModeInterlude: false, ModeTitle: false, ModeIntro: false} {
		g.mode = m
		if g.helpShown() != want {
			t.Fatalf("mode %d: shown=%v want %v", m, g.helpShown(), want)
		}
	}
}

// The window shows every action, with the keys of the CURRENT bindings (not hardcoded).
func TestHelpListsEveryActionFromTheSettings(t *testing.T) {
	ob, ob2 := binds, binds2
	t.Cleanup(func() { binds, binds2 = ob, ob2 })
	binds = defaultBindings()
	binds2 = defaultBindings2()
	binds[actInteract].Keys[0] = ebiten.KeyQ
	binds2[actDance].Keys[0] = ebiten.KeyB
	binds[actDance].Pad = ebiten.StandardGamepadButtonLeftStick
	rows := func(rs []helpRow) map[string]string {
		m := map[string]string{}
		for _, r := range rs {
			m[r.label] = strings.Join(r.caps, "+")
		}
		return m
	}
	j1, j2, pad := rows(helpKeyboardRows(0)), rows(helpKeyboardRows(1)), rows(helpPadRows())
	for _, lab := range []string{"Mover", "Correr", "Golpear", "Usar / interactuar", "Bailar / señuelo", "Ayuda", "Menú / salir"} {
		for n, m := range []map[string]string{j1, j2, pad} {
			if m[lab] == "" {
				t.Fatalf("column %d lacks %q", n, lab)
			}
		}
	}
	if j1["Usar / interactuar"] != "Q" || j2["Bailar / señuelo"] != "B" || pad["Bailar / señuelo"] != "L3" {
		t.Fatalf("keys not read from the settings: %v %v %v", j1, j2, pad)
	}
	if j1["Mover"] != "W+A+S+D" || j2["Mover"] != "↑+←+↓+→" || pad["Usar / interactuar"] != "X" || pad["Ayuda"] != "SELECT" {
		t.Fatalf("default keys: %v %v %v", j1, j2, pad)
	}
	// the new features each have an action with a pad button that nothing else uses
	need := []string{"caja", "señuelo", "Trampas", "Traje", "VIP", "Bailar", "Sigilo"}
	for _, n := range need {
		found := false
		for _, m := range helpModes {
			if strings.Contains(m.text, n) {
				found = true
				if binds[m.act].Pad == noPad {
					t.Fatalf("%s has no pad button", n)
				}
			}
		}
		if !found {
			t.Fatalf("the help does not mention %q", n)
		}
	}
}

// Select (Back) is not used by any action, so opening the help never clashes with the game.
func TestHelpPadButtonIsFree(t *testing.T) {
	for _, b := range []Bindings{defaultBindings(), defaultBindings2()} {
		for a := Action(0); a < actCount; a++ {
			if b[a].Pad == ebiten.StandardGamepadButtonCenterLeft {
				t.Fatalf("action %d uses Select", a)
			}
		}
	}
}

func TestHelpTooltipOnHover(t *testing.T) {
	g := helpGame(t, false)
	f := helpHW(t)
	for _, over := range []bool{false, true} {
		if over {
			f.cx, f.cy = int(helpBtnX)+3, int(helpBtnY)+3
		}
		textRec = &textRecorder{}
		img := ebiten.NewImage(screenW, screenH)
		g.Draw(img)
		img.Deallocate()
		found := false
		for _, tr := range textRec.texts {
			if strings.Contains(tr.s, "F1") && strings.Contains(tr.s, "Select") {
				found = true
			}
		}
		textRec = nil
		if found != over {
			t.Fatalf("tooltip shown=%v with the mouse over=%v", found, over)
		}
	}
}
