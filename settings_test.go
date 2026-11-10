package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// no scrolling: every row of each tab fits inside the panel, under the column titles and above the notice line
func TestSettingsTabsFitWithoutScroll(t *testing.T) {
	for tab := 0; tab < 2; tab++ {
		l := settingsLayoutTab(tab)
		if bot := setTop + settingsContentH(l); bot > setNoteY-2 {
			t.Errorf("tab %d ends at y=%.0f, over the notice line (%.0f)", tab, bot, setNoteY)
		}
		if setTop < setColHeadY+12 {
			t.Errorf("the first section (%.0f) touches the column titles (%.0f)", setTop, setColHeadY)
		}
		for _, it := range l {
			if it.row >= 0 && rowTab(it.row) != tab {
				t.Errorf("tab %d shows row %d of tab %d", tab, it.row, rowTab(it.row))
			}
		}
	}
}

// the number of guests is no longer in the settings but it is in DEBUG
func TestGuestRowsOnlyInDebug(t *testing.T) {
	for _, it := range settingsLayout() {
		if it.title == "Invitados mínimo" || it.title == "Invitados máximo" {
			t.Fatal("guest limits in the settings")
		}
	}
	if setRows != int(actCount)+4+actPlayerCount {
		t.Fatalf("settings rows: %d", setRows)
	}
	g := &Game{}
	n := 0
	for _, r := range g.debugRows() {
		if r.label == "Invitados mínimo" || r.label == "Invitados máximo" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("debug guest rows: %d", n)
	}
}

// sections have a title each and every settings row sits under exactly one, in one of the two tabs (the
// language and the tabs row live outside them); the first section of J1 is CONTROLES J1, like CONTROLES J2
func TestSettingsSections(t *testing.T) {
	seen := map[int]int{}
	var titles []string
	for tab := 0; tab < 2; tab++ {
		for _, it := range settingsLayoutTab(tab) {
			if it.row < 0 {
				titles = append(titles, it.title)
			} else {
				seen[it.row]++
			}
		}
	}
	want := []string{"CONTROLES J1", "ACCIONES Y PODERES", "ENTRADA", "CONTROLES J2", "ACCIONES Y PODERES", "ENTRADA"}
	if fmt.Sprint(titles) != fmt.Sprint(want) {
		t.Fatalf("titles: %v, want %v", titles, want)
	}
	for r := 0; r < setRows; r++ {
		n := seen[r]
		if r == setRowLang || r == setRowTabs {
			n++ // outside the tabs
		}
		if n != 1 {
			t.Fatalf("row %d appears %d times", r, seen[r])
		}
	}
}

// no "Volver", "Créditos" nor "Restablecer" rows and no "Tecla 2" column; the language is a row of its own
func TestSettingsNoExtraRows(t *testing.T) {
	g := auditGame(t)
	auditReset(g)
	t.Cleanup(func() { auditReset(g); g.mode = ModeMenu; settingsTab = 0 })
	for _, sel := range []int{0, setRowLang, setRowTabs, setRowP2} {
		g.mode, g.setSel = ModeSettings, sel
		textRec = &textRecorder{}
		img := ebiten.NewImage(screenW, screenH)
		g.Draw(img)
		img.Deallocate()
		texts := textRec.texts
		textRec = nil
		has := map[string]bool{}
		for _, r := range texts {
			has[r.s] = true
			for _, bad := range []string{"VOLVER", "CRÉDITOS", "RESTAURAR", "Tecla 2", "Tecla 1", "Atajos fijos", "Esc:", "Enter:", "Supr", "↑↓", "IDIOMA / LANGUAGE"} {
				if strings.Contains(r.s, bad) {
					t.Errorf("sel=%d: %q is drawn in the settings", sel, r.s)
				}
			}
		}
		for _, w := range []string{"Tecla", "Mando", "JUGADOR 1", "JUGADOR 2", "◄ Español ►", "IDIOMA"} {
			if !has[w] {
				t.Errorf("sel=%d: %q is not drawn", sel, w)
			}
		}
		head := "CONTROLES J1"
		if sel == setRowP2 {
			head = "CONTROLES J2"
		}
		if !has[head] {
			t.Errorf("sel=%d: section %q missing", sel, head)
		}
	}
}

// the tabs: Tab / ←→ on the tab row / a click switch them; ↑↓ walk only the rows of the tab shown, with the
// language and the tabs on top; the language box changes the language from the keyboard and with the mouse
func TestSettingsTabsAndLanguage(t *testing.T) {
	pg, pu := gameLang, uiLang
	t.Cleanup(func() { gameLang, uiLang = pg, pu; settingsTab = 0 })
	settingsTab = 0
	if settingsRowAt(setRowLang, 1) != setRowTabs || settingsRowAt(setRowTabs, 1) != 0 || settingsRowAt(setRowLang, -1) != setRowDev1 {
		t.Fatalf("J1 order: lang->%d tabs->%d up from lang->%d", settingsRowAt(setRowLang, 1), settingsRowAt(setRowTabs, 1), settingsRowAt(setRowLang, -1))
	}
	settingsTab = 1
	if settingsRowAt(setRowTabs, 1) != setRowP2 || settingsRowAt(setRowLang, -1) != setRowDev2 {
		t.Fatalf("J2 order: tabs->%d up from lang->%d", settingsRowAt(setRowTabs, 1), settingsRowAt(setRowLang, -1))
	}
	// the arrows go round the language, the tabs and every row of the tab, once each
	for tab := 0; tab < 2; tab++ {
		settingsTab = tab
		seen := map[int]bool{}
		r := setRowLang
		for i := 0; i < 40 && !seen[r]; i++ {
			seen[r] = true
			r = settingsRowAt(r, 1)
		}
		if want := 2 + len(settingsLayoutTab(tab)) - 3; len(seen) != want || r != setRowLang {
			t.Errorf("tab %d: the arrows visit %d rows, want %d", tab, len(seen), want)
		}
	}
	settingsTab = 0
	if setTabAt(setTabX[1], setTabY+5) != 1 || setTabAt(setTabX[0], setTabY+5) != 0 || setTabAt(setTabX[0], setTabY-3) != -1 {
		t.Error("the mouse does not find the tabs")
	}
	if !setLangOver(setLangX+5, setLangY+5) || setLangOver(setLangX+setLangW+2, setLangY+5) {
		t.Error("the mouse does not find the language box")
	}
	// the language box is clear of the logo and of the panel, under the logo
	const logoBottom = 30 + 34 + 70 + 4 + 3
	if setLangY < logoBottom || setLangX+setLangW > 230 {
		t.Errorf("language box (%.0f,%.0f) over the logo or the panel", setLangX, setLangY)
	}
	g := &Game{}
	g.snd = &Sounds{}
	g.setSel = setRowTabs
	g.settingsSetTab(1)
	if settingsTab != 1 || g.setSel != setRowTabs {
		t.Errorf("tab %d sel %d", settingsTab, g.setSel)
	}
	g.setSel = 3 // a row of J1 while J2 is shown: the selection goes to the tabs
	g.settingsSetTab(0)
	g.setSel = 3
	g.settingsSetTab(1)
	if g.setSel != setRowTabs {
		t.Errorf("switching tab left the selection on a row of the other tab (%d)", g.setSel)
	}
	settingsSyncTab(setRowP2 + 2)
	if settingsTab != 1 {
		t.Error("a row of J2 selected does not show the J2 tab")
	}
	settingsSyncTab(setRowLang)
	if settingsTab != 1 {
		t.Error("the language row changed the tab")
	}
	settingsSyncTab(0)
	if settingsTab != 0 {
		t.Error("a row of J1 selected does not show the J1 tab")
	}
	before := gameLang
	noA := noAssets
	noAssets = true
	t.Cleanup(func() { noAssets = noA })
	toggleLanguage()
	if gameLang == before || uiLang != gameLang {
		t.Errorf("language %q -> %q (ui %q)", before, gameLang, uiLang)
	}
}

// settings.json: an old file with two keys per action still loads (the second key is kept as it was, not shown)
// and is written back the same
func TestSettingsSecondKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	prev, pb, pb2 := settingsFileOverride, binds, binds2
	t.Cleanup(func() { settingsFileOverride, binds, binds2 = prev, pb, pb2 })
	settingsFileOverride = filepath.Join(dir, "settings.json")
	old := `{"up":{"keys":["W","ArrowUp"],"pad":12},"attack":{"keys":["Space","J"],"pad":0},"run":{"keys":["ShiftLeft"],"pad":7}}`
	if err := os.WriteFile(settingsFileOverride, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	b := loadBindings()
	if b[actUp].Keys[0] != ebiten.KeyW || b[actUp].Keys[1] != ebiten.KeyArrowUp || b[actAttack].Keys[1] != ebiten.KeyJ {
		t.Fatalf("old file: %+v %+v", b[actUp], b[actAttack])
	}
	if b[actRun].Keys[0] != ebiten.KeyShiftLeft || b[actRun].Keys[1] != noKey {
		t.Fatalf("one key: %+v", b[actRun])
	}
	binds = b
	saveBindings()
	if b2 := loadBindings(); b2 != b {
		t.Fatal("the bindings change after a save and a load")
	}
}

// the two tabs list exactly the same actions (by name, in the same order): player 2 has the actions AND the
// powers of player 1, each row bound to his own binding
func TestSettingsBothTabsSameActions(t *testing.T) {
	var names [2][]string
	for tab := 0; tab < 2; tab++ {
		for _, it := range settingsLayoutTab(tab) {
			if it.row < 0 || !isBindRow(it.row) {
				continue
			}
			bd, a := rowBinding(it.row)
			want := &binds
			if tab == 1 {
				want = &binds2
			}
			if bd != &want[a] {
				t.Errorf("tab %d row %d edits the binding of the other player", tab, it.row)
			}
			names[tab] = append(names[tab], actionNames[a])
		}
	}
	if len(names[0]) != int(actCount) || fmt.Sprint(names[0]) != fmt.Sprint(names[1]) {
		t.Fatalf("J1 %v\nJ2 %v", names[0], names[1])
	}
	for _, w := range []string{"Atrás / cancelar", "Confirmar", "Provocar suceso", "Poder: inmunidad", "Poder: traer famosos", "Poder: enviar asesino", "Poder: sucesos"} {
		if !strings.Contains(fmt.Sprint(names[1]), w) {
			t.Errorf("J2 has no %q", w)
		}
	}
}

// the fixed shortcuts of the game, which no default binding of player 2 may take
var fixedShortcutKeys = []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7,
	ebiten.Key8, ebiten.Key9, ebiten.KeyR, ebiten.KeyG, ebiten.KeyM, ebiten.KeyF11, ebiten.KeyF2, ebiten.KeyEscape,
	ebiten.KeyTab, ebiten.KeyF1, ebiten.KeyBackspace, ebiten.KeyDelete}

// the default keys: every action of player 2 has a key and a pad button; no key of player 2 is a key of player 1
// or a fixed shortcut; the main key of each action is different from the others of the same player (only the
// second key may repeat one of his own: Space / . also confirm)
func TestBindingDefaultsNoCollisions(t *testing.T) {
	b1, b2 := defaultBindings(), defaultBindings2()
	for _, b := range []Bindings{b1, b2} {
		seen := map[ebiten.Key]Action{}
		for a := Action(0); a < actCount; a++ {
			k := b[a].Keys[0]
			if k == noKey || b[a].Pad == noPad {
				t.Errorf("action %s has no default key or button", actionNames[a])
			}
			if o, ok := seen[k]; ok {
				t.Errorf("key %s is the main key of %s and %s", keyLabel(k), actionNames[o], actionNames[a])
			}
			seen[k] = a
		}
	}
	for a := Action(0); a < actCount; a++ {
		for _, k := range b2[a].Keys {
			if k == noKey {
				continue
			}
			for _, f := range fixedShortcutKeys {
				if k == f {
					t.Errorf("J2 %s uses the fixed shortcut %s", actionNames[a], keyLabel(k))
				}
			}
			for c := Action(0); c < actCount; c++ {
				for s, k1 := range b1[c].Keys {
					if k1 == k && !(s == 1 && isArrowKey(k1)) {
						t.Errorf("J1 %s and J2 %s share %s", actionNames[c], actionNames[a], keyLabel(k))
					}
				}
			}
		}
		if lbl := keyLabel(b2[a].Keys[0]); len([]rune(lbl)) > 12 {
			t.Errorf("J2 %s: label %q too long for the key column", actionNames[a], lbl)
		}
	}
}

// settings.json keeps every action of player 2 (the powers too): a save and a load give the same bindings
func TestSettingsP2RoundTrip(t *testing.T) {
	withSettingsFile(t, "")
	binds, binds2 = defaultBindings(), defaultBindings2()
	binds2[actImmune] = Binding{[2]ebiten.Key{ebiten.KeyZ, noKey}, ebiten.StandardGamepadButtonCenterLeft}
	binds2[actKiller] = Binding{[2]ebiten.Key{ebiten.KeyX, noKey}, noPad}
	binds2[actBack] = Binding{[2]ebiten.Key{ebiten.KeyQ, noKey}, ebiten.StandardGamepadButtonRightRight}
	want := binds2
	saveBindings()
	data, err := os.ReadFile(settingsFileOverride)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range actionIDs {
		if !strings.Contains(string(data), `"p2_`+id+`"`) {
			t.Errorf("p2_%s not saved", id)
		}
	}
	binds2 = Bindings{}
	binds = loadBindings()
	loadPlayerSettings()
	if binds2 != want {
		t.Fatalf("J2 after a load:\n%+v\nwant\n%+v", binds2, want)
	}
}

// a settings.json written when player 2 only had the movement, run, stab, interact and dance loads: those keep
// what the file says, the new actions of player 2 take their defaults, except a default key that player 1
// (or player 2 himself) already chose, which stays empty instead of taking it away; the file is not written
func TestSettingsOldFileWithoutP2Actions(t *testing.T) {
	old := map[string]savedBinding{
		"event":     {Keys: []string{"U", ""}, Pad: int(ebiten.StandardGamepadButtonFrontTopLeft)}, // J1 chose U (J2's default immunity key)
		"p2_attack": {Keys: []string{"Y", ""}, Pad: int(ebiten.StandardGamepadButtonRightBottom)},  // J2 stabs with Y (his default famous key)
		"p2_run":    {Keys: []string{"ShiftRight", ""}, Pad: int(ebiten.StandardGamepadButtonFrontBottomRight)},
	}
	data, _ := json.Marshal(old)
	withSettingsFile(t, string(data))
	binds = loadBindings()
	loadPlayerSettings()
	def := defaultBindings2()
	if binds2[actAttack].Keys[0] != ebiten.KeyY || binds2[actRun].Keys[0] != ebiten.KeyShiftRight {
		t.Fatalf("saved J2 keys lost: %+v %+v", binds2[actAttack], binds2[actRun])
	}
	if binds[actEvent].Keys[0] != ebiten.KeyU {
		t.Fatalf("J1 lost his key: %+v", binds[actEvent])
	}
	if binds2[actImmune].Keys[0] != noKey || binds2[actFamous].Keys[0] != noKey {
		t.Errorf("a taken default key was given to J2: immune %+v famous %+v", binds2[actImmune], binds2[actFamous])
	}
	for _, a := range []Action{actBack, actConfirm, actEvent, actKiller, actPowerEvent} {
		if binds2[a] != def[a] {
			t.Errorf("J2 %s: %+v, want the default %+v", actionNames[a], binds2[a], def[a])
		}
	}
	if binds2[actImmune].Pad != def[actImmune].Pad {
		t.Error("the pad button of a new action must be the default")
	}
	after, _ := os.ReadFile(settingsFileOverride)
	if string(after) != string(data) {
		t.Error("loading an old file wrote it")
	}
	// nothing in the file: every default
	withSettingsFile(t, "")
	binds = loadBindings()
	loadPlayerSettings()
	if binds2 != defaultBindings2() {
		t.Error("no file: J2 must have every default")
	}
}

// in a party of two the game reads player 2's OWN bindings for the actions and the powers: on two pads his
// pad fires the powers (justEither, the way powers.go and the event key ask) and player 1's pad does not do
// it for him; on one keyboard player 1 does not read player 2's power keys
func TestTwoPlayersPowersUseP2Bindings(t *testing.T) {
	playersTestSetup(t)
	oj := padJustFn
	t.Cleanup(func() { padJustFn = oj })
	fakePads(2)
	twoInput, playerDev = true, [2]int{0, 1}
	binds2[actKiller].Pad = ebiten.StandardGamepadButtonCenterLeft // J2 moved his killer button
	press := map[ebiten.GamepadID]ebiten.StandardGamepadButton{}
	padJustFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool {
		pb, ok := press[id]
		return ok && pb == b
	}
	for _, a := range []Action{actEvent, actImmune, actFamous, actKiller, actPowerEvent, actBack, actConfirm} {
		press = map[ebiten.GamepadID]ebiten.StandardGamepadButton{11: binds2[a].Pad} // J2's pad
		if !a.justEither() || !a.justFor(1) || a.justFor(0) {
			t.Errorf("%s: J2's own button is not his (either %v, J2 %v, J1 %v)", actionNames[a], a.justEither(), a.justFor(1), a.justFor(0))
		}
	}
	press = map[ebiten.GamepadID]ebiten.StandardGamepadButton{11: ebiten.StandardGamepadButtonLeftStick} // J1's killer button, on J2's pad
	if actKiller.justFor(1) {
		t.Error("J2 still fires the killer with player 1's button")
	}
	// one keyboard: each power key belongs to one player only
	fakePads(0)
	playerDev = [2]int{-1, -1}
	for a := Action(0); a < actCount; a++ {
		k := binds2[a].Keys[0]
		if k != noKey && (keyAllowed(0, k) || !keyAllowed(1, k)) {
			t.Errorf("%s: J2's key %s (J1 reads it %v, J2 %v)", actionNames[a], keyLabel(k), keyAllowed(0, k), keyAllowed(1, k))
		}
		if k1 := binds[a].Keys[0]; k1 != noKey && !keyAllowed(0, k1) {
			t.Errorf("%s: J1 cannot read his key %s", actionNames[a], keyLabel(k1))
		}
	}
}
