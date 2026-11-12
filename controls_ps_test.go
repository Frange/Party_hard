package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The PlayStation-style default pad (user, 2026-11-11): ✕ stab / confirm, ○ interact / back, △ dance, R1 run,
// START pause, and the migration of the settings.json files written with the old buttons.

const (
	psX        = ebiten.StandardGamepadButtonRightBottom
	psO        = ebiten.StandardGamepadButtonRightRight
	psSquare   = ebiten.StandardGamepadButtonRightLeft
	psTriangle = ebiten.StandardGamepadButtonRightTop
	psR1       = ebiten.StandardGamepadButtonFrontTopRight
	psR2       = ebiten.StandardGamepadButtonFrontBottomRight
	psStart    = ebiten.StandardGamepadButtonCenterRight
	psSelect   = ebiten.StandardGamepadButtonCenterLeft
)

func TestPadDefaultsPlayStationLayout(t *testing.T) {
	want := map[Action]ebiten.StandardGamepadButton{
		actAttack: psX, actConfirm: psX, actInteract: psO, actBack: psO, actDance: psTriangle, actRun: psR1,
		actImmune: psR2, actEvent: ebiten.StandardGamepadButtonFrontTopLeft,
		actUp: ebiten.StandardGamepadButtonLeftTop, actDown: ebiten.StandardGamepadButtonLeftBottom,
		actLeft: ebiten.StandardGamepadButtonLeftLeft, actRight: ebiten.StandardGamepadButtonLeftRight,
	}
	for p, b := range [2]Bindings{defaultBindings(), defaultBindings2()} {
		for a, pad := range want {
			if b[a].Pad != pad {
				t.Errorf("J%d %s: pad %s, want %s", p+1, actionNames[a], padLabel(b[a].Pad), padLabel(pad))
			}
		}
	}
	for b, l := range map[ebiten.StandardGamepadButton]string{psX: "X", psO: "O", psTriangle: "Triángulo", psSquare: "Cuadrado", psR1: "R1", psR2: "R2", psStart: "START", psSelect: "SELECT"} {
		if padLabel(b) != l {
			t.Errorf("label of %d: %q, want %q", b, padLabel(b), l)
		}
	}
}

// every party action has a pad button, no two party actions share one (only confirm / back of the menus share
// ✕ / ○ with stab / interact), and START, SELECT and the minimap's □ stay free
func TestPadDefaultsNoGapsNoClashes(t *testing.T) {
	for p, b := range [2]Bindings{defaultBindings(), defaultBindings2()} {
		owner := map[ebiten.StandardGamepadButton]Action{}
		for a := Action(0); a < actCount; a++ {
			pad := b[a].Pad
			if pad == noPad {
				t.Errorf("J%d %s has no pad button", p+1, actionNames[a])
				continue
			}
			if pad == psStart || pad == psSelect || pad == minimapPadButton {
				t.Errorf("J%d %s on the fixed button %s", p+1, actionNames[a], padLabel(pad))
			}
			if menuAction(a) {
				continue
			}
			if o, ok := owner[pad]; ok {
				t.Errorf("J%d %s and %s share %s", p+1, actionNames[o], actionNames[a], padLabel(pad))
			}
			owner[pad] = a
		}
		if b[actConfirm].Pad != b[actAttack].Pad || b[actBack].Pad != b[actInteract].Pad {
			t.Errorf("J%d: menus ✕ / ○ do not match stab / interact", p+1)
		}
	}
	if !minimapPadFree() && binds == defaultBindings() && binds2 == defaultBindings2() {
		t.Error("the minimap's □ is taken by a default action")
	}
}

// ---- migration of settings.json ----------------------------------------------------------------------------

// oldFile: a settings.json of v1.0.0 (no controls_rev) where every action has the given pad, the rest the old defaults.
func oldSettingsFile(t *testing.T, custom map[string]ebiten.StandardGamepadButton) string {
	t.Helper()
	m := map[string]any{}
	add := func(prefix string, b Bindings) {
		for a := Action(0); a < actCount; a++ {
			pad := b[a].Pad
			if old, ok := oldPadDefaults[a]; ok {
				pad = old
			}
			if c, ok := custom[prefix+actionIDs[a]]; ok {
				pad = c
			}
			m[prefix+actionIDs[a]] = bindingTo(Binding{b[a].Keys, pad})
		}
	}
	add("", defaultBindings())
	add("p2_", defaultBindings2())
	m["players"] = savedPlayers{Dev1: devAutoPref, Dev2: devAutoPref}
	data, _ := json.Marshal(m)
	return string(data)
}

func loadSettingsForTest(t *testing.T, content string) (before string) {
	t.Helper()
	withSettingsFile(t, content)
	b, _ := os.ReadFile(settingsFileOverride)
	binds = loadBindings()
	loadPlayerSettings()
	return string(b)
}

func TestSettingsMigrationUntouchedOldFile(t *testing.T) {
	before := loadSettingsForTest(t, oldSettingsFile(t, nil))
	if binds != defaultBindings() || binds2 != defaultBindings2() {
		t.Fatalf("an old file with the old defaults did not take the new ones:\nJ1 %+v\nJ2 %+v", binds, binds2)
	}
	if after, _ := os.ReadFile(settingsFileOverride); string(after) != before {
		t.Error("loading rewrote settings.json")
	}
	// saving writes the revision; the next load keeps whatever is there, also a button that was an old default
	binds[actRun].Pad = psR2
	saveBindings()
	data, _ := os.ReadFile(settingsFileOverride)
	if !strings.Contains(string(data), `"controls_rev": 1`) {
		t.Fatalf("no controls_rev in the saved file:\n%s", data)
	}
	binds = loadBindings()
	if binds[actRun].Pad != psR2 {
		t.Error("a file of the new revision was migrated again (run R2 chosen by the player)")
	}
}

func TestSettingsMigrationPartlyCustom(t *testing.T) {
	loadSettingsForTest(t, oldSettingsFile(t, map[string]ebiten.StandardGamepadButton{
		"dance":       ebiten.StandardGamepadButtonLeftStick, // J1 changed dance: kept
		"p2_interact": ebiten.StandardGamepadButtonFrontTopLeft,
		"p2_event":    psTriangle, // J2 swapped interact and event... and dance (△ default) clashes, as he left it
	}))
	def, def2 := defaultBindings(), defaultBindings2()
	if binds[actDance].Pad != ebiten.StandardGamepadButtonLeftStick {
		t.Errorf("J1's own dance button lost: %s", padLabel(binds[actDance].Pad))
	}
	for _, a := range []Action{actRun, actInteract, actImmune, actAttack} {
		if binds[a].Pad != def[a].Pad {
			t.Errorf("J1 %s: %s, want the new default %s", actionNames[a], padLabel(binds[a].Pad), padLabel(def[a].Pad))
		}
	}
	if binds2[actInteract].Pad != ebiten.StandardGamepadButtonFrontTopLeft || binds2[actEvent].Pad != psTriangle {
		t.Errorf("J2's own buttons lost: interact %s event %s", padLabel(binds2[actInteract].Pad), padLabel(binds2[actEvent].Pad))
	}
	if binds2[actRun].Pad != def2[actRun].Pad || binds2[actImmune].Pad != def2[actImmune].Pad {
		t.Errorf("J2 run %s immune %s: want R1 / R2", padLabel(binds2[actRun].Pad), padLabel(binds2[actImmune].Pad))
	}
}

// a new default that would land on a button the player chose for another party action is not taken
func TestSettingsMigrationNoNewClash(t *testing.T) {
	loadSettingsForTest(t, oldSettingsFile(t, map[string]ebiten.StandardGamepadButton{
		"famous": psO, // J1 put "bring famous" on ○: interact keeps □
	}))
	if binds[actFamous].Pad != psO || binds[actInteract].Pad != psSquare {
		t.Errorf("famous %s interact %s, want ○ / □", padLabel(binds[actFamous].Pad), padLabel(binds[actInteract].Pad))
	}
	if binds[actRun].Pad != psR1 || binds[actImmune].Pad != psR2 {
		t.Errorf("run %s immune %s, want R1 / R2", padLabel(binds[actRun].Pad), padLabel(binds[actImmune].Pad))
	}
	// the player had moved run to R1 and immunity to L3 already: both are his and stay
	loadSettingsForTest(t, oldSettingsFile(t, map[string]ebiten.StandardGamepadButton{"run": psR1, "immune": ebiten.StandardGamepadButtonLeftStick}))
	if binds[actRun].Pad != psR1 || binds[actImmune].Pad != ebiten.StandardGamepadButtonLeftStick {
		t.Errorf("custom run %s immune %s", padLabel(binds[actRun].Pad), padLabel(binds[actImmune].Pad))
	}
}

func TestSettingsMigrationNoFileOrCorrupt(t *testing.T) {
	for _, content := range []string{"", "{not json", `{"run": 5`, `[]`} {
		before := loadSettingsForTest(t, content)
		if binds != defaultBindings() || binds2 != defaultBindings2() {
			t.Errorf("file %q: not the new defaults", content)
		}
		if after, _ := os.ReadFile(settingsFileOverride); string(after) != before {
			t.Errorf("file %q: loading wrote it", content)
		}
	}
}

// ---- the menus and the pause with a real game and a fake pad ---------------------------------------------

type psPadHW struct {
	down map[ebiten.StandardGamepadButton]bool
}

func psPadFake(t *testing.T) *psPadHW {
	t.Helper()
	f := &psPadHW{down: map[ebiten.StandardGamepadButton]bool{}}
	oids, oj, oh, oa := padIDsFn, padJustFn, padHeldFn, padAxisFn
	ob, ob2, ot, od, ol, of := binds, binds2, twoInput, playerDev, lastInputPad, progressFrozen
	padIDsFn = func() []ebiten.GamepadID { return []ebiten.GamepadID{7} }
	padJustFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool { return f.down[b] }
	padHeldFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool { return f.down[b] }
	padAxisFn = func(id ebiten.GamepadID, a ebiten.StandardGamepadAxis) float64 { return 0 }
	binds, binds2, progressFrozen = defaultBindings(), defaultBindings2(), true
	t.Cleanup(func() {
		padIDsFn, padJustFn, padHeldFn, padAxisFn = oids, oj, oh, oa
		binds, binds2, twoInput, playerDev, lastInputPad, progressFrozen = ob, ob2, ot, od, ol, of
	})
	return f
}

// press: one frame of the game with only that button down.
func (f *psPadHW) press(t *testing.T, g *Game, b ebiten.StandardGamepadButton) {
	t.Helper()
	f.down = map[ebiten.StandardGamepadButton]bool{b: true}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	f.down = map[ebiten.StandardGamepadButton]bool{}
}

func TestPadMenusXConfirmsOBack(t *testing.T) {
	g := auditGame(t)
	auditReset(g)
	withSettingsFile(t, "")
	f := psPadFake(t)
	t.Cleanup(func() { auditReset(g); g.mode = ModeMenu })
	// main menu: ✕ opens the row (AJUSTES); in the settings ○ goes back
	g.mode = ModeMenu
	for i, id := range g.menuList() {
		if id == menuSettings {
			g.mainSel = i
		}
	}
	f.press(t, g, psX)
	if g.mode != ModeSettings || g.showCredits {
		t.Fatalf("✕ on AJUSTES: mode %d", g.mode)
	}
	f.press(t, g, psO)
	if g.mode != ModeMenu {
		t.Fatalf("○ in AJUSTES: mode %d, want the menu", g.mode)
	}
	// party list: ○ back to the menu, ✕ starts the party
	g.mode, g.menuSel = ModeLevelSelect, 0
	f.press(t, g, psO)
	if g.mode != ModeMenu {
		t.Fatalf("○ in the party list: mode %d", g.mode)
	}
	g.mode, g.menuSel = ModeLevelSelect, 0
	f.press(t, g, psX)
	if g.mode == ModeLevelSelect || g.mode == ModeMenu {
		t.Fatalf("✕ in the party list did not start the party: mode %d", g.mode)
	}
}

func TestPadPauseStartOpensOClosesXPicks(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := helpGame(t, two)
		withSettingsFile(t, "")
		f := psPadFake(t)
		g.fromEditor, g.pause = false, pauseState{}
		t.Cleanup(func() { g.pause = pauseState{} })
		if two {
			playerDev = [2]int{0, -1}
		}
		f.press(t, g, psO) // ○ in a party is interact: no pause, still playing
		if g.pause.open || g.mode != ModePlay {
			t.Fatalf("two=%v: ○ opened the pause (open %v, mode %d)", two, g.pause.open, g.mode)
		}
		f.press(t, g, psStart)
		if !g.pause.open {
			t.Fatalf("two=%v: START did not open the pause", two)
		}
		f.press(t, g, psO)
		if g.pause.open || g.mode != ModePlay {
			t.Fatalf("two=%v: ○ did not close the pause", two)
		}
		f.press(t, g, psStart)
		g.p.cooldown = 0
		f.press(t, g, psX) // CONTINUAR: closes, and the same ✕ does not stab in that frame
		if g.pause.open || g.mode != ModePlay {
			t.Fatalf("two=%v: ✕ on CONTINUAR did not close it", two)
		}
		if g.p.cooldown != 0 {
			t.Fatalf("two=%v: the ✕ that closed the pause also stabbed", two)
		}
		f.press(t, g, psStart)
		f.press(t, g, psStart) // START closes it too
		if g.pause.open {
			t.Fatalf("two=%v: START did not close the pause", two)
		}
	}
}

// with a pad as the last thing used, the written hints name its buttons; with the keyboard, the keys
func TestPadHintsNameButtons(t *testing.T) {
	f := psPadFake(t)
	_ = f
	twoInput = false
	lastInputPad = false
	if actionHint(actInteract) != "E" || actionHint(actAttack) != "Espacio" {
		t.Fatalf("keyboard hints: %q %q", actionHint(actInteract), actionHint(actAttack))
	}
	if padKeyText("R / ENTER: reintentar    ESC: menú") != "R / ENTER: reintentar    ESC: menú" {
		t.Error("keyboard: the ENTER / ESC hint changed")
	}
	lastInputPad = true
	got := []string{actionHint(actAttack), actionHint(actInteract), actionHint(actDance), actionHint(actRun), actionHint(actUp), actKeyName(0, actInteract)}
	if strings.Join(got, " ") != "X O Triángulo R1 ↑ O" {
		t.Fatalf("pad hints: %v", got)
	}
	if s := padKeyText("R / ENTER: reintentar    ESC: menú"); s != "X: reintentar    O: menú" {
		t.Errorf("pad ENTER / ESC hint: %q", s)
	}
	// two players: each one by his own device
	twoInput, playerDev, lastInputPad = true, [2]int{-1, 0}, false
	if actionHintFor(actInteract, 0) != "E" || actionHintFor(actInteract, 1) != "O" {
		t.Errorf("2P hints: J1 %q J2 %q", actionHintFor(actInteract, 0), actionHintFor(actInteract, 1))
	}
	// the device tracker: a key -> keyboard, a pad button -> pad
	twoInput = false
	ok, op := keysJustFn, padButtonsFn
	t.Cleanup(func() { keysJustFn, padButtonsFn = ok, op })
	var keys []ebiten.Key
	var btns []ebiten.StandardGamepadButton
	keysJustFn = func(b []ebiten.Key) []ebiten.Key { return append(b, keys...) }
	padButtonsFn = func(id ebiten.GamepadID, b []ebiten.StandardGamepadButton) []ebiten.StandardGamepadButton {
		return append(b, btns...)
	}
	btns = []ebiten.StandardGamepadButton{psX}
	trackInputDevice()
	if !lastInputPad {
		t.Error("a pad button did not switch the hints to the pad")
	}
	btns, keys = nil, []ebiten.Key{ebiten.KeyE}
	trackInputDevice()
	if lastInputPad {
		t.Error("a key did not switch the hints to the keyboard")
	}
}

// in a menu ○ only goes back: in the wardrobe it does not also switch J1 / J2 (□ does that on a pad)
func TestPadMenuInteractDoesNotStealBack(t *testing.T) {
	f := psPadFake(t)
	twoInput = false
	f.down = map[ebiten.StandardGamepadButton]bool{psO: true}
	if menuInteractJust() {
		t.Error("○ counts as interact in a menu")
	}
	if !actBack.just() {
		t.Error("○ is not back")
	}
	f.down = map[ebiten.StandardGamepadButton]bool{psSquare: true}
	if !menuInteractJust() {
		t.Error("□ does not switch the player in a menu")
	}
	binds[actInteract].Pad = ebiten.StandardGamepadButtonLeftStick // a custom interact button works as before
	f.down = map[ebiten.StandardGamepadButton]bool{ebiten.StandardGamepadButtonLeftStick: true}
	if !menuInteractJust() {
		t.Error("a custom interact button is ignored in the menus")
	}
}
