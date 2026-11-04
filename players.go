package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// One or two players: the switch of the main menu, who plays with what, and the
// rows of the settings screen for player 2. The rest of the two-player game lives
// in input2.go (how the keys and pads are read), wardrobe.go (the clothes of each
// player) and the core of the game.
//
//   twoPlayerMode   the 1 / 2 switch. Saved in settings.json ("players"); 1 by default.
//   devPref         what each player plays with: -2 automatic, -1 keyboard, n >= 0 the
//                   n-th gamepad. Automatic: two pads -> a pad each; one pad -> player 1 on
//                   the keyboard and player 2 on the pad; no pad -> both share the keyboard
//                   (player 1 W A S D, player 2 the arrow keys).
//   binds2          the keys and pad buttons of player 2 (input2.go): arrows, right Shift
//                   and . , / by default. Player 2 only has the player actions (move, run,
//                   stab, interact, dance); back, confirm, the event key and the powers
//                   are shared and belong to player 1.
//   autoDevices()   the device of each player (input2.go): what the game uses when a party
//                   starts, and again each frame while two play, so a pad that is plugged in
//                   or pulled out takes effect at once.
//   wardrobeNumPlayers (wardrobe.go) answers 2 when this switch says so (twoPlayerMode).
// ---------------------------------------------------------------------------

// the player actions: movement, run, stab, interact and dance (player 2 has its own keys for them)
const actPlayerCount = int(actDance) + 1

const devAutoPref = -2 // "choose for me" in devPref

var (
	twoPlayerMode bool
	devPref       = [2]int{devAutoPref, devAutoPref}
)

type savedPlayers struct {
	Two  bool `json:"two"`
	Dev1 int  `json:"dev1"`
	Dev2 int  `json:"dev2"`
}

// loadPlayerSettings reads the player mode, the devices and the keys of player 2 (settings.json).
// A file written before there were two players has none of them: the defaults stay.
func loadPlayerSettings() {
	raw := readSettingsFile()
	binds2 = defaultBindings2()
	for i := 0; i < actPlayerCount; i++ {
		var sb savedBinding
		if m, ok := raw["p2_"+actionIDs[i]]; ok && json.Unmarshal(m, &sb) == nil {
			binds2[i] = bindingFrom(sb, binds2[i])
		}
	}
	twoPlayerMode, devPref = false, [2]int{devAutoPref, devAutoPref}
	sp := savedPlayers{Dev1: devAutoPref, Dev2: devAutoPref}
	if m, ok := raw["players"]; ok && json.Unmarshal(m, &sp) == nil {
		twoPlayerMode = sp.Two
		for i, d := range [2]int{sp.Dev1, sp.Dev2} {
			if d >= devAutoPref && d < 8 {
				devPref[i] = d
			}
		}
	}
}

// resolveDevices turns the preferences into the device of each player, given the connected pads.
func resolveDevices(pref [2]int, pads int) [2]int {
	d, auto := pref, smartDevices(pads)
	for i := range d {
		if d[i] == devAutoPref {
			d[i] = auto[i]
		}
	}
	if d[0] == d[1] && d[0] >= 0 { // one pad cannot drive two people
		d[1] = -1
	}
	return d
}

// refreshPlayerDevices keeps playerDev up to date: pads can be plugged in or pulled out at any time.
func refreshPlayerDevices() {
	if twoPlayerMode {
		playerDev = autoDevices()
	}
}

func deviceName(d int) string {
	if d < 0 {
		return tr("Teclado")
	}
	return tr("Mando") + fmt.Sprintf(" %d", d+1)
}

func devPrefName(d int) string {
	if d == devAutoPref {
		return tr("Auto")
	}
	return deviceName(d)
}

// cycleDevPref moves a player's choice through Auto, Teclado, Mando 1, Mando 2.
func cycleDevPref(d, dir int) int {
	opts := []int{devAutoPref, -1, 0, 1}
	i := 0
	for k, o := range opts {
		if o == d {
			i = k
		}
	}
	return opts[(i+dir+len(opts))%len(opts)]
}

// togglePlayers is the 1 / 2 switch of the main menu.
func (g *Game) togglePlayers() {
	twoPlayerMode = !twoPlayerMode
	refreshPlayerDevices()
	saveBindings()
	g.snd.Play("spark")
}

// padConnectedFor says whether the pad a player plays with is plugged in.
func padConnectedFor(dev int) bool {
	return dev < 0 || dev < len(padList())
}

// ---- the help that appears at the start of a party with two players --------------------------------------

func keyShort(k ebiten.Key) string {
	if k == noKey {
		return "-"
	}
	return keyLabel(k)
}

// playerHelp is the line of controls of one player, for the help panel.
func playerHelp(p int, dev int) string {
	b := &binds
	if p == 1 {
		b = &binds2
	}
	if dev >= 0 {
		if !padConnectedFor(dev) {
			return tr("Mando desconectado") + " (" + deviceName(dev) + ")"
		}
		return strings.Join([]string{
			tr("Mando") + fmt.Sprintf(" %d", dev+1) + ": " + tr("palanca") + " " + tr("mover"),
			padLabel(b[actAttack].Pad) + " " + tr("golpear"),
			padLabel(b[actInteract].Pad) + " " + tr("usar"),
			padLabel(b[actDance].Pad) + " " + tr("bailar"),
			padLabel(b[actRun].Pad) + " " + tr("correr"),
		}, " · ")
	}
	move := strings.Join([]string{keyShort(b[actUp].Keys[0]), keyShort(b[actLeft].Keys[0]), keyShort(b[actDown].Keys[0]), keyShort(b[actRight].Keys[0])}, "/")
	if b[actUp].Keys[0] == ebiten.KeyArrowUp && b[actLeft].Keys[0] == ebiten.KeyArrowLeft && b[actDown].Keys[0] == ebiten.KeyArrowDown && b[actRight].Keys[0] == ebiten.KeyArrowRight {
		move = tr("Flechas")
	}
	return strings.Join([]string{
		move + " " + tr("mover"),
		keyShort(b[actAttack].Keys[0]) + " " + tr("golpear"),
		keyShort(b[actInteract].Keys[0]) + " " + tr("usar"),
		keyShort(b[actDance].Keys[0]) + " " + tr("bailar"),
		keyShort(b[actRun].Keys[0]) + " " + tr("correr"),
	}, " · ")
}

// controlsHelpTime is how long the help stays on screen at the start of a party.
const controlsHelpTime = 5.0

// drawControlsHelp: the controls of both players, in blue and red, at the start of a two-player party.
func (g *Game) drawControlsHelp(dst *ebiten.Image) {
	if !twoPlayerMode || g.mode != ModePlay || g.time > controlsHelpTime {
		return
	}
	a := 1.0
	if g.time > controlsHelpTime-1 {
		a = controlsHelpTime - g.time
	}
	devs := autoDevices()
	lines := []string{playerHelp(0, devs[0]), playerHelp(1, devs[1])}
	w := 0.0
	for _, s := range lines {
		w = math.Max(w, g.textW(s, 1))
	}
	w += 34
	x, y := math.Round(screenW/2-w/2), float64(screenH-64)
	fillRect(dst, x, y, w, 40, rgba(0x05070e, uint8(220*a)))
	for i, s := range lines {
		c := playerColor(i)
		c.A = uint8(255 * a)
		fillRect(dst, x+8, y+6+float64(i)*16, 8, 8, c)
		tc := rgb(0xe8e4f4)
		tc.A = uint8(255 * a)
		g.drawText(dst, s, x+24, y+4+float64(i)*16, tc, 1)
	}
}

// drawPlayersRow is the "JUGADORES  ◄ 1 ►" row of the main menu.
func (g *Game) drawPlayersRow(dst *ebiten.Image, label string, y float64, sel bool, col, hot color.RGBA) {
	c := col
	if sel {
		c = hot
	}
	g.drawText(dst, label, 110, y, c, 2.2)
	x := 110 + g.textW(label, 2.2) + 14
	val, note := "1", "tú solito"
	if twoPlayerMode {
		val, note = "2", "¡el doble de caos!"
	}
	arrow := rgb(0x8a86a0)
	if sel {
		arrow = rgb(0xffe060)
	}
	g.drawText(dst, "◄", x, y, arrow, 2.2)
	x += g.textW("◄", 2.2) + 8
	if twoPlayerMode {
		fillRect(dst, x, y+4, 8, 8, playerColor(0))
		fillRect(dst, x+10, y+4, 8, 8, playerColor(1))
		x += 24
	}
	g.drawText(dst, val, x, y, rgb(0xffe060), 2.2)
	x += g.textW(val, 2.2) + 8
	g.drawText(dst, "►", x, y, arrow, 2.2)
	if sel { // a funny line in the gap between this row and the next one, clear of the logo, the window and the bed
		n := len(g.menuList())
		nx, ny := playersNotePos(y, menuRowY(1, n)-menuRowY(0, n))
		g.drawText(dst, note, nx, ny, rgb(0x8a86a0), 1)
	}
}

// playersNotePos is where the funny line of the JUGADORES row goes: aligned with the labels, in the
// gap of the list between the row and the next one (rows are 33 to 40 px apart, the big text is 22 px tall).
func playersNotePos(y, pitch float64) (x, ny float64) { return 110, y + math.Floor(8+pitch/2) }
