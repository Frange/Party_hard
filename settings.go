package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The settings screen rebinds the keyboard key and the gamepad button of each action, for player 1 and for
// player 2 (one TAB each with the same actions and powers, everything on screen without scrolling), chooses what each player plays with and
// the language (on the left, under the logo). The credits live in the main menu now (menuCredits); they are
// drawn by this screen (g.showCredits) and close back to the menu.
// The second key of a binding (Binding.Keys[1]: the arrows, J, Space...) is no longer shown nor edited: it
// keeps working as a fixed alternative and settings.json keeps it as it was (old files load the same).

const (
	setRowLang = int(actCount)     // language of the game (the box on the left)
	setRowTabs = int(actCount) + 1 // the two tabs J1 / J2 (←→ switch)
	setRowDev1 = int(actCount) + 2 // what player 1 plays with
	setRowDev2 = int(actCount) + 3 // what player 2 plays with
	setRowP2   = int(actCount) + 4 // + the player actions: the keys of player 2
	setRows    = setRowP2 + actPlayerCount
)

// isBindRow: a row with the two columns (key, pad). For player 1 it is the action itself,
// for player 2 the row is setRowP2 + action.
func isBindRow(r int) bool { return r < int(actCount) || (r >= setRowP2 && r < setRows) }

// rowBinding is the binding a bind row edits and its action.
func rowBinding(r int) (*Binding, Action) {
	if r >= setRowP2 {
		a := Action(r - setRowP2)
		return &binds2[a], a
	}
	return &binds[r], Action(r)
}

// the layout: the panel, the title, the tabs, the column titles and the rows (no scrolling: everything fits).
// Every block keeps at least 12 px of air from the next one (TestSettingsLayoutMargins, Spanish and English).
const (
	setTitleY   = 18.0  // AJUSTES (scale 2)
	setRowH     = 19.0  // a row
	setHeadH    = 19.0  // a section title
	setSecGap   = 3.0   // air before each title but the first
	setTop      = 126.0 // first section title
	setColHeadY = 94.0
	setTabY     = 62.0 // the two tabs
	setTabH     = 20.0
	setTabW     = 200.0
	setNoteY    = 506.0 // a notice (key swapped, waiting for a player to join)
	setKeyX     = 600.0 // centre of the key column
	setPadX     = 790.0 // centre of the pad column
	setColW     = 160.0
	// the language box, on the left under the logo
	setLangX, setLangY = 22.0, 150.0
	setLangW, setLangH = 196.0, 70.0
)

var setTabX = [2]float64{470, 690} // centres of the tabs J1 / J2
var setColX = [2]float64{setKeyX, setPadX}

// settingsTab: the tab shown (0 = player 1, 1 = player 2).
var settingsTab int

type setItem struct {
	row   int // the settings row (>= 0) or -1 for a section title
	title string
	y, h  float64 // position under setTop
}

// rowTab: the tab a row belongs to (-1: the language and the tabs, shared by both).
func rowTab(r int) int {
	switch {
	case r == setRowLang || r == setRowTabs:
		return -1
	case r == setRowDev2 || r >= setRowP2:
		return 1
	}
	return 0
}

// settingsLayoutTab lays the sections of one tab out: CONTROLES J1 / J2, ACCIONES Y PODERES, ENTRADA (the
// same rows for both players).
func settingsLayoutTab(tab int) []setItem {
	var out []setItem
	y := 0.0
	section := func(title string, rows ...int) {
		if len(out) > 0 {
			y += setSecGap
		}
		out = append(out, setItem{-1, title, y, setHeadH})
		y += setHeadH
		for _, r := range rows {
			out = append(out, setItem{r, "", y, setRowH})
			y += setRowH
		}
	}
	// both tabs show the same actions in the same order: player 2 rebinds exactly what player 1 does
	base, head, dev := 0, "CONTROLES J1", setRowDev1
	if tab == 1 {
		base, head, dev = setRowP2, "CONTROLES J2", setRowDev2
	}
	var ctrl, powers []int
	for a := actUp; a < actCount; a++ {
		if a < actEvent {
			ctrl = append(ctrl, base+int(a))
		} else {
			powers = append(powers, base+int(a))
		}
	}
	section(head, ctrl...)
	section("ACCIONES Y PODERES", powers...)
	section("ENTRADA", dev)
	return out
}

// settingsLayout: the layout of the tab shown.
func settingsLayout() []setItem { return settingsLayoutTab(settingsTab) }

// settingsSyncTab shows the tab of the selected row (a row of the other tab selected by a flag or a test).
func settingsSyncTab(sel int) {
	if t := rowTab(sel); t >= 0 {
		settingsTab = t
	}
}

// settingsRowAt moves from a row to the previous / next one as the screen shows them: the language, the
// tabs and the rows of the tab shown (the rows are not numbered in the order they appear).
func settingsRowAt(sel, dir int) int {
	order := []int{setRowLang, setRowTabs}
	for _, it := range settingsLayout() {
		if it.row >= 0 {
			order = append(order, it.row)
		}
	}
	for i, r := range order {
		if r == sel {
			return order[(i+dir+len(order))%len(order)]
		}
	}
	return order[0]
}

// dbgSetSel (-setsel flag, debug): the mouse does not pick rows, so a capture shows the row asked for.
var dbgSetSel bool

// settingsJoin is the player (1 or 2) waiting for a key or a pad button to join with ("Pulsa un botón
// para unirte"); 0 when nobody is.
var settingsJoin int

func settingsContentH(l []setItem) float64 {
	last := l[len(l)-1]
	return last.y + last.h
}

// settingsSetTab shows a tab; the selection stays on the tab row (or goes there when it was a row of the
// other tab).
func (g *Game) settingsSetTab(t int) {
	if t == settingsTab {
		return
	}
	settingsTab = t
	if rt := rowTab(g.setSel); rt >= 0 && rt != t {
		g.setSel = setRowTabs
	}
	g.snd.Play("spark")
}

func setLangOver(mx, my float64) bool {
	return mx >= setLangX && mx < setLangX+setLangW && my >= setLangY && my < setLangY+setLangH
}

// setTabAt: the tab under the mouse, or -1.
func setTabAt(mx, my float64) int {
	if my < setTabY || my >= setTabY+setTabH {
		return -1
	}
	for i, cx := range setTabX {
		if mx >= cx-setTabW/2 && mx < cx+setTabW/2 {
			return i
		}
	}
	return -1
}

func toggleLanguage() {
	gameLang = otherLang(gameLang)
	uiLang = gameLang
	saveLanguages()
}

func (g *Game) updateSettings() {
	if settingsNoteT > 0 {
		settingsNoteT--
	}
	if g.showCredits { // opened from the main menu: closing them goes back there
		if actConfirm.just() || actBack.just() || escJust() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.showCredits = false
			g.mode = ModeMenu
		}
		return
	}
	if settingsJoin > 0 {
		g.updateJoin()
		return
	}
	settingsSyncTab(g.setSel)
	if g.capturing {
		// the next key (or pad button) pressed becomes the binding; Esc cancels
		if escJust() {
			g.capturing = false
			return
		}
		bd, _ := rowBinding(g.setSel)
		if g.setCol == 0 {
			for _, k := range inpututil.AppendJustPressedKeys(nil) {
				assignKey(g.setSel, 0, k)
				g.capturing = false
				saveBindings()
				return
			}
			return
		}
		for _, id := range padList() { // the button of whichever pad is pressed
			for _, b := range inpututil.AppendJustPressedStandardGamepadButtons(id, nil) {
				bd.Pad = b
				g.capturing = false
				saveBindings()
				return
			}
		}
		return
	}
	if g.setCol > 1 {
		g.setCol = 1
	}
	layout := settingsLayout()
	// the mouse picks rows (hover) and acts on them (click): the language box, the tabs and the rows
	mouseConfirm := false
	if click := g.mouseMenu(func(mx, my float64) int {
		if dbgSetSel {
			return -1
		}
		if setLangOver(mx, my) {
			return setRowLang
		}
		if setTabAt(mx, my) >= 0 {
			return setRowTabs
		}
		if mx < 240 || mx > 920 {
			return -1
		}
		for _, it := range layout {
			if it.row < 0 {
				continue
			}
			if y := setTop + it.y; my >= y && my < y+it.h {
				return it.row
			}
		}
		return -1
	}, &g.setSel); click {
		mx, my := ebiten.CursorPosition()
		switch {
		case g.setSel == setRowTabs:
			if t := setTabAt(float64(mx), float64(my)); t >= 0 {
				g.settingsSetTab(t)
			}
		case isBindRow(g.setSel):
			for c, cx := range setColX {
				if math.Abs(float64(mx)-cx) < setColW/2 {
					g.setCol, mouseConfirm = c, true
				}
			}
		default:
			mouseConfirm = true
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.settingsSetTab(1 - settingsTab)
	}
	if actDown.just() {
		g.setSel = settingsRowAt(g.setSel, 1)
	}
	if actUp.just() {
		g.setSel = settingsRowAt(g.setSel, -1)
	}
	if g.setSel == setRowLang && (actRight.just() || actLeft.just()) {
		toggleLanguage()
	}
	if g.setSel == setRowTabs {
		if actRight.just() {
			g.settingsSetTab(1)
		} else if actLeft.just() {
			g.settingsSetTab(0)
		}
	}
	if g.setSel == setRowDev1 || g.setSel == setRowDev2 {
		p, dir := g.setSel-setRowDev1, 0
		if actRight.just() {
			dir = 1
		} else if actLeft.just() {
			dir = -1
		}
		if dir != 0 {
			devPref[p] = cycleDevPref(devPref[p], dir)
			refreshPlayerDevices()
			saveBindings()
		}
	}
	if isBindRow(g.setSel) && (actRight.just() || actLeft.just()) {
		g.setCol = 1 - g.setCol
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if isBindRow(g.setSel) {
			bd, _ := rowBinding(g.setSel)
			if g.setCol == 0 {
				bd.Keys[0] = noKey
			} else {
				bd.Pad = noPad
			}
			saveBindings()
		}
	}
	if actConfirm.just() || mouseConfirm {
		switch {
		case isBindRow(g.setSel):
			if g.setCol == 1 && !padConnected() {
				return // no gamepad connected: nothing to capture
			}
			g.capturing = true
		case g.setSel == setRowDev1:
			settingsJoin = 1
		case g.setSel == setRowDev2:
			settingsJoin = 2
		case g.setSel == setRowLang:
			toggleLanguage()
		case g.setSel == setRowTabs:
			if !mouseConfirm {
				g.settingsSetTab(1 - settingsTab)
			}
		}
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

// updateJoin: the player that is joining takes the device of the next key or pad button pressed.
func (g *Game) updateJoin() {
	if escJust() {
		settingsJoin = 0
		return
	}
	p := settingsJoin - 1
	for i, id := range padList() {
		if len(inpututil.AppendJustPressedStandardGamepadButtons(id, nil)) > 0 {
			devPref[p], settingsJoin = i, 0
			refreshPlayerDevices()
			saveBindings()
			g.snd.Play("spark")
			return
		}
	}
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		devPref[p], settingsJoin = -1, 0
		refreshPlayerDevices()
		saveBindings()
		g.snd.Play("spark")
	}
}

// drawSettingsLang: the language, on the left under the logo (◄ Español ►).
func (g *Game) drawSettingsLang(dst *ebiten.Image) {
	sel := g.setSel == setRowLang
	edge, tc := rgba(0x6a4ab0, 200), rgb(0xffe060)
	fillRect(dst, setLangX, setLangY, setLangW, setLangH, rgba(0x05070e, 225))
	if sel {
		edge = rgb(0xff6a8a)
		fillRect(dst, setLangX, setLangY, setLangW, setLangH, rgba(0x6a4ab0, 70))
	}
	fillRect(dst, setLangX, setLangY, setLangW, 1, edge)
	fillRect(dst, setLangX, setLangY+setLangH-1, setLangW, 1, edge)
	fillRect(dst, setLangX, setLangY, 1, setLangH, edge)
	fillRect(dst, setLangX+setLangW-1, setLangY, 1, setLangH, edge)
	hc := rgb(0xffa030)
	if sel {
		hc = rgb(0xff6a8a)
	}
	cx := setLangX + setLangW/2
	title, label, ty, ly := g.settingsLangLayout()
	g.drawTextC(dst, title, cx, ty, hc, 1)
	g.drawTextC(dst, label, cx, ly, tc, 2)
}

// settingsLangLabel: the text of the language selector for a language ("◄ Español ►").
func settingsLangLabel(lang string) string {
	if lang == "en" {
		return "◄ English ►"
	}
	return "◄ Español ►"
}

// settingsLangLayout: the title and the selector of the language box with their whole-pixel top y
// (the title at scale 1, the selector at scale 2), both lines together centred vertically in the box.
func (g *Game) settingsLangLayout() (title, label string, ty, ly float64) {
	title, label = "IDIOMA", settingsLangLabel(gameLang)
	_, h1 := textMeasure(g, tr(title))
	_, h2 := textMeasure(g, tr(label))
	h1, h2 = h1*pxScale(1), h2*pxScale(2)
	const gap = 4.0
	top := math.Round(setLangY + (setLangH-(h1+gap+h2))/2)
	return title, label, top, math.Round(top + h1 + gap)
}

func (g *Game) drawSettings(dst *ebiten.Image) {
	g.drawBedroom(dst)
	g.drawLogo(dst, logoSideCX, 30) // like the party list: the HARDY of the logo keeps clear of the panel
	if g.showCredits {              // the credits cover everything: nothing of the settings is drawn under them
		g.drawCredits(dst)
		return
	}
	settingsSyncTab(g.setSel)
	g.drawSettingsLang(dst)
	fillRect(dst, 230, 14, 700, 512, rgba(0x05070e, 232))
	fillRect(dst, 230, 14, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 525, 700, 1, rgba(0x6a4ab0, 160))
	g.drawTextC(dst, "AJUSTES", 580, setTitleY, rgb(0xff6a8a), 2)
	// the tabs
	for i, cx := range setTabX {
		on := i == settingsTab
		x := cx - setTabW/2
		bg, edge, tc := rgba(0x05070e, 200), rgba(0x6a4ab0, 140), rgb(0x8a86a0)
		if on {
			bg, edge, tc = rgba(0x2a1a4a, 240), playerColor(i), rgb(0xffffff)
		}
		if g.setSel == setRowTabs && on {
			edge, tc = rgb(0xff6a8a), rgb(0xff6a8a)
		}
		fillRect(dst, x, setTabY, setTabW, setTabH, bg)
		fillRect(dst, x, setTabY, setTabW, 1, edge)
		fillRect(dst, x, setTabY+setTabH-1, setTabW, 1, edge)
		fillRect(dst, x, setTabY, 1, setTabH, edge)
		fillRect(dst, x+setTabW-1, setTabY, 1, setTabH, edge)
		g.drawTextC(dst, []string{"JUGADOR 1", "JUGADOR 2"}[i], cx, setTabY+2, tc, 1)
	}
	for i, h := range []string{"Tecla", "Mando"} {
		g.drawTextC(dst, h, setColX[i], setColHeadY, rgb(0x8a86a0), 1)
	}
	fillRect(dst, 240, setTop-4, 680, 1, rgba(0x6a4ab0, 110))
	for _, it := range settingsLayout() {
		y := setTop + it.y
		if it.row < 0 {
			g.drawText(dst, it.title, 252, y+3, rgb(0xffa030), 1)
			x := 252 + g.textW(it.title, 1) + 10
			fillRect(dst, x, y+10, 920-x, 1, rgba(0xffa030, 90))
			continue
		}
		i := it.row
		sel := i == g.setSel
		col := rgb(0x3fb59a)
		if sel {
			col = rgb(0xff6a8a)
			fillRect(dst, 242, y, 676, it.h-2, rgba(0x6a4ab0, 60))
		}
		ty := y + 1
		if i == setRowDev1 || i == setRowDev2 {
			p := i - setRowDev1
			dc := rgb(0xffe060)
			g.drawText(dst, "Entrada J"+fmt.Sprint(p+1), 262, ty, col, 1)
			fillRect(dst, 242, y+3, 4, it.h-7, playerColor(p))
			name := devPrefName(devPref[p])
			if devPref[p] != devAutoPref && !padConnectedFor(devPref[p]) {
				name, dc = name+" ("+tr("desconectado")+")", rgb(0xff6a6a)
			} else if devPref[p] == devAutoPref {
				name += " → " + deviceName(autoDevices()[p])
			}
			g.drawTextC(dst, "◄ "+name+" ►", (setKeyX+setPadX)/2, ty, dc, 1)
			continue
		}
		bd, a := rowBinding(i)
		g.drawText(dst, actionNames[a], 262, ty, col, 1)
		fillRect(dst, 242, y+3, 4, it.h-7, playerColor(rowTab(i)))
		vals := [2]string{keyLabel(bd.Keys[0]), padLabel(bd.Pad)}
		for c := 0; c < 2; c++ {
			vc := rgb(0xb8b0d0)
			if sel && g.setCol == c {
				vc = rgb(0xffe060)
				fillRect(dst, setColX[c]-setColW/2, y+1, setColW, it.h-4, rgba(0x6a4ab0, 110))
				if g.capturing {
					vals[c] = "pulsa..."
				}
			}
			g.drawTextC(dst, vals[c], setColX[c], ty, vc, 1)
		}
	}
	// a notice: a key swapped with the other player, or a player joining
	note := ""
	if settingsJoin > 0 {
		note = "Jugador " + fmt.Sprint(settingsJoin) + ": pulsa un botón para unirte (una tecla = teclado)"
	} else if settingsNoteT > 0 && !g.capturing {
		note = settingsNote
	}
	if note != "" {
		g.drawTextC(dst, note, 580, setNoteY, rgb(0xf0e060), 1)
	}
}

func (g *Game) drawCredits(dst *ebiten.Image) {
	fillRect(dst, 240, 60, 680, 430, rgb(0x05070e))
	fillRect(dst, 240, 60, 680, 1, rgb(0xff6a8a))
	fillRect(dst, 240, 489, 680, 1, rgb(0xff6a8a))
	g.drawTextC(dst, "CRÉDITOS", 580, 84, rgb(0xff6a8a), 2.2)
	g.drawTextC(dst, "Party Hardy · Fiesta Nocturna", 580, 122, rgb(0xf4f0ff), 1.6)
	g.drawTextC(dst, versionLabel()+" · 2026", 580, 146, rgb(0x8a86a0), 1.2)
	g.drawWrapped(dst, "Juego hecho como homenaje al Party Hard original, desarrollado por Pinokl Games y distribuido por tinyBuild. Este proyecto es de fans y no tiene ninguna relación oficial con ellos.", 290, 178, 580, rgb(0xb8b0d0))
	rows := [][2]string{{"Director", "Frange"}, {"Testers", "Diego, César y Samuel"}, {"Año", "2026"}}
	for i, r := range rows {
		y := 262 + float64(i)*34
		g.drawText(dst, r[0], 330, y, rgb(0x3fb59a), 1.6)
		g.drawText(dst, r[1], 500, y, rgb(0xffe060), 1.6)
	}
	g.drawTextC(dst, "Gracias por jugar", 580, 420, rgb(0xff6a8a), 1.4)
}

var _ = fmt.Sprint

// settingsNote: a notice on the hint line of the settings (a key taken from the other player), settingsNoteT
// frames left.
var (
	settingsNote  string
	settingsNoteT int
)

// otherPlayerBinds: the bindings of the player who does NOT own the settings row r.
func otherPlayerBinds(r int) *Bindings {
	if r >= setRowP2 {
		return &binds
	}
	return &binds2
}

// assignKey puts key k in slot of the bind row r. A key is never shared between the two players: if the
// other player has it, he gets the key this slot had (a swap; nothing if the slot was empty) and a notice
// says so. Within one player a key may serve several actions (Space attacks and confirms).
func assignKey(r, slot int, k ebiten.Key) {
	bd, _ := rowBinding(r)
	old := bd.Keys[slot]
	bd.Keys[slot] = k
	other := otherPlayerBinds(r)
	for a := 0; a < actPlayerCount; a++ {
		for s := 0; s < 2; s++ {
			if other[a].Keys[s] == k {
				other[a].Keys[s] = old
				if old == k {
					other[a].Keys[s] = noKey
				}
				who := "J2"
				if r >= setRowP2 {
					who = "J1"
				}
				settingsNote = fmt.Sprintf("La tecla %s ya era del %s: se intercambian", keyLabel(k), who)
				settingsNoteT = 240
			}
		}
	}
}

func init() {
	langDataEN["La tecla %s ya era del %s: se intercambian"] = "Key %s was already %s's: they swap"
	langDataEN["Jugador 1: pulsa un botón para unirte (una tecla = teclado)"] = "Player 1: press a button to join in (a key = keyboard)"
	langDataEN["Jugador 2: pulsa un botón para unirte (una tecla = teclado)"] = "Player 2: press a button to join in (a key = keyboard)"
}
