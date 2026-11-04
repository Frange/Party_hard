package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// playersTestSetup points the settings to a temporary file and restores every global afterwards.
func playersTestSetup(t *testing.T) string {
	t.Helper()
	oldFile, oldTwo, oldPref, oldB, oldB2, oldDev, oldIn, oldPads := settingsFileOverride, twoPlayerMode, devPref, binds, binds2, playerDev, twoInput, padIDsFn
	path := filepath.Join(t.TempDir(), "settings.json")
	settingsFileOverride = path
	twoPlayerMode, devPref = false, [2]int{devAutoPref, devAutoPref}
	binds, binds2, twoInput = defaultBindings(), defaultBindings2(), false
	t.Cleanup(func() {
		settingsFileOverride, twoPlayerMode, devPref, binds, binds2, playerDev, twoInput, padIDsFn = oldFile, oldTwo, oldPref, oldB, oldB2, oldDev, oldIn, oldPads
	})
	return path
}

func fakePads(n int) {
	padIDsFn = func() []ebiten.GamepadID {
		var ids []ebiten.GamepadID
		for i := 0; i < n; i++ {
			ids = append(ids, ebiten.GamepadID(10+i))
		}
		return ids
	}
}

// one player by default, and the switch of the menu changes the number the party asks for
func TestPlayersDefaultAndSwitch(t *testing.T) {
	playersTestSetup(t)
	g := &Game{snd: &Sounds{muted: true}}
	if g.wardTwo() || wardrobeNumPlayers(g) != 1 {
		t.Fatal("the default is one player")
	}
	g.togglePlayers()
	if !g.wardTwo() || wardrobeNumPlayers(g) != 2 {
		t.Fatal("the switch must give two players")
	}
	g.togglePlayers()
	if g.wardTwo() {
		t.Fatal("the switch must go back to one player")
	}
}

// the mode, the devices and the keys of player 2 survive a restart; an old file (only the actions of
// player 1) leaves the defaults and unknown entries are kept
func TestPlayersPersistence(t *testing.T) {
	path := playersTestSetup(t)
	// an old settings file: only the actions of player 1
	old := `{"up": {"keys": ["W", "ArrowUp"], "pad": 12}, "extra": {"x": 1}}`
	os.WriteFile(path, []byte(old), 0o644)
	loadPlayerSettings()
	if twoPlayerMode || devPref != [2]int{devAutoPref, devAutoPref} {
		t.Fatalf("old file: mode %v devs %v", twoPlayerMode, devPref)
	}
	if binds2 != defaultBindings2() {
		t.Fatal("old file: player 2 keys must be the defaults")
	}
	twoPlayerMode, devPref = true, [2]int{-1, 1}
	binds2[actAttack].Keys[0] = ebiten.KeyX
	binds2[actDance].Pad = ebiten.StandardGamepadButtonRightRight
	saveBindings()
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"extra"`) {
		t.Fatal("an entry this version does not know was thrown away")
	}
	// forget everything and read it back
	twoPlayerMode, devPref, binds2 = false, [2]int{devAutoPref, devAutoPref}, defaultBindings2()
	loadPlayerSettings()
	if !twoPlayerMode || devPref != [2]int{-1, 1} {
		t.Fatalf("not saved: mode %v devs %v", twoPlayerMode, devPref)
	}
	if binds2[actAttack].Keys[0] != ebiten.KeyX || binds2[actDance].Pad != ebiten.StandardGamepadButtonRightRight {
		t.Fatal("the keys of player 2 were not saved")
	}
	if b := loadBindings(); b[actUp].Pad != ebiten.StandardGamepadButtonLeftTop && b[actUp].Pad != 12 {
		t.Fatalf("player 1 bindings broke: %v", b[actUp])
	}
}

// who plays with what when nothing is chosen, with any number of pads plugged in
func TestPlayersAutomaticDevices(t *testing.T) {
	playersTestSetup(t)
	for _, c := range []struct {
		pads int
		want [2]int
	}{{0, [2]int{-1, -1}}, {1, [2]int{-1, 0}}, {2, [2]int{0, 1}}, {3, [2]int{0, 1}}} {
		fakePads(c.pads)
		if got := autoDevices(); got != c.want {
			t.Errorf("%d pads: %v, want %v", c.pads, got, c.want)
		}
	}
	// chosen by the player: one pad cannot drive both of them
	fakePads(2)
	devPref = [2]int{1, 1}
	if got := autoDevices(); got[0] != 1 || got[1] != -1 {
		t.Errorf("the same pad twice: %v", got)
	}
	devPref = [2]int{-1, 0}
	if got := autoDevices(); got != [2]int{-1, 0} {
		t.Errorf("keyboard + pad 1: %v", got)
	}
	// a pad that is pulled out: the player is left on a pad that is not there (the help says so)
	fakePads(0)
	if padConnectedFor(0) || !padConnectedFor(-1) {
		t.Fatal("a missing pad must be reported as unplugged")
	}
	// plugging one in is noticed at once
	fakePads(1)
	devPref = [2]int{devAutoPref, devAutoPref}
	twoPlayerMode = true
	refreshPlayerDevices()
	if playerDev != [2]int{-1, 0} {
		t.Fatalf("hot plug: %v", playerDev)
	}
	fakePads(0)
	refreshPlayerDevices()
	if playerDev != [2]int{-1, -1} {
		t.Fatalf("hot unplug: %v", playerDev)
	}
}

// two people on one keyboard: no key does two things, and player 1 keeps W A S D
func TestPlayersKeysDoNotOverlap(t *testing.T) {
	playersTestSetup(t)
	twoInput, playerDev = true, [2]int{-1, -1}
	for a := Action(0); int(a) < actPlayerCount; a++ {
		for _, k2 := range binds2[a].Keys {
			if k2 == noKey {
				continue
			}
			for b := Action(0); int(b) < actPlayerCount; b++ {
				for _, k1 := range binds[b].Keys {
					if k1 == k2 && keyAllowed(0, k1) {
						t.Errorf("key %v is read by player 1 (%v) and by player 2 (%v)", k1, b, a)
					}
				}
			}
		}
	}
	for _, k := range []ebiten.Key{ebiten.KeyW, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD, ebiten.KeySpace, ebiten.KeyE, ebiten.KeyF, ebiten.KeyShiftLeft} {
		if !keyAllowed(0, k) {
			t.Errorf("player 1 lost the key %v", k)
		}
	}
	for _, k := range []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight} {
		if keyAllowed(0, k) {
			t.Errorf("player 1 still reads %v", k)
		}
	}
	// player 2 on a pad: the keyboard is only player 1's, arrows included
	playerDev = [2]int{-1, 0}
	if !keyAllowed(0, ebiten.KeyArrowUp) || keyAllowed(1, ebiten.KeyArrowUp) {
		t.Error("with player 2 on a pad the arrows belong to player 1")
	}
}

// one player: the input is exactly what it was before there were two
func TestPlayersOnePlayerInputUnchanged(t *testing.T) {
	playersTestSetup(t)
	if bindsFor(0) != &binds || bindsFor(1) != &binds {
		t.Fatal("one player must read the normal bindings")
	}
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if !keyAllowed(0, k) {
			t.Fatalf("one player: key %v is filtered out", k)
		}
	}
	// the default bindings are the ones the game always had
	b := defaultBindings()
	if b[actUp].Keys[0] != ebiten.KeyW || b[actAttack].Keys[0] != ebiten.KeySpace || b[actInteract].Keys[0] != ebiten.KeyE || b[actDance].Keys[0] != ebiten.KeyF {
		t.Fatal("the default keys of player 1 changed")
	}
}

// the help of the start of a party names each player's keys or pad, and the unplugged pad
func TestPlayersHelpLines(t *testing.T) {
	playersTestSetup(t)
	fakePads(1)
	kb := playerHelp(0, -1)
	if !strings.Contains(kb, "W/A/S/D") || !strings.Contains(kb, "Espacio") {
		t.Errorf("keyboard help: %s", kb)
	}
	p2 := playerHelp(1, -1)
	if !strings.Contains(p2, "Flechas") || !strings.Contains(p2, ". golpear") {
		t.Errorf("player 2 keyboard help: %s", p2)
	}
	pad := playerHelp(1, 0)
	if !strings.Contains(pad, "Mando 1") {
		t.Errorf("pad help: %s", pad)
	}
	if gone := playerHelp(1, 1); !strings.Contains(gone, "desconectado") {
		t.Errorf("unplugged pad help: %s", gone)
	}
}

// the settings screen: every row once, the arrows walk the rows as laid out, player 2 has its section
func TestPlayersSettingsRows(t *testing.T) {
	seen := map[int]bool{}
	row := 0
	for i := 0; i < setRows; i++ {
		if seen[row] {
			t.Fatalf("the arrows come back to row %d too soon", row)
		}
		seen[row] = true
		row = settingsRowAt(row, 1)
	}
	if row != 0 || len(seen) != setRows {
		t.Fatalf("the rows are not a loop: %d of %d", len(seen), setRows)
	}
	if settingsRowAt(0, -1) != setRowBack {
		t.Fatalf("up from the first row must go to the last one, got %d", settingsRowAt(0, -1))
	}
	titles := map[string]bool{}
	for _, it := range settingsLayout() {
		if it.row < 0 {
			titles[it.title] = true
		}
	}
	for _, want := range []string{"JUGADORES", "CONTROLES DEL JUGADOR 2"} {
		if !titles[want] {
			t.Errorf("section %q missing", want)
		}
	}
	for a := 0; a < actPlayerCount; a++ {
		bd, act := rowBinding(setRowP2 + a)
		if bd != &binds2[a] || int(act) != a || !isBindRow(setRowP2+a) {
			t.Errorf("row of player 2 action %d points elsewhere", a)
		}
	}
	if isBindRow(setRowDev1) || isBindRow(setRowLang) {
		t.Error("device and language rows have no three columns")
	}
}

// the choices of a player's device go round: auto, keyboard, pad 1, pad 2
func TestPlayersDevicePrefCycle(t *testing.T) {
	d := devAutoPref
	var seq []int
	for i := 0; i < 4; i++ {
		d = cycleDevPref(d, 1)
		seq = append(seq, d)
	}
	if seq[0] != -1 || seq[1] != 0 || seq[2] != 1 || seq[3] != devAutoPref {
		t.Fatalf("cycle: %v", seq)
	}
	if cycleDevPref(devAutoPref, -1) != 1 {
		t.Fatal("backwards")
	}
}
