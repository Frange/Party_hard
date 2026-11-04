package main

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The settings screen rebinds keyboard keys (two slots per action) and the
// gamepad button of each action, for player 1 and for player 2, and chooses what
// each player plays with. Rows: the actions of player 1, the extras, the devices
// and the actions of player 2 (see the layout).

const (
	setRowLang    = int(actCount) // language of the game
	setRowCredits = int(actCount) + 1
	setRowReset   = int(actCount) + 2
	setRowBack    = int(actCount) + 3
	setRowDev1    = int(actCount) + 4 // what player 1 plays with
	setRowDev2    = int(actCount) + 5 // what player 2 plays with
	setRowP2      = int(actCount) + 6 // + the player actions: the keys of player 2
	setRows       = setRowP2 + actPlayerCount
)

// isBindRow: a row with the three columns (key 1, key 2, pad). For player 1 it is the action itself,
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

// The list is split in titled sections and scrolls: every row has room to breathe.
const (
	setRowH     = 26.0 // a row
	setHeadH    = 30.0 // a section title
	setSecGap   = 10.0 // air before each title but the first
	setTop      = 100.0
	setViewH    = 350.0
	setColHeadY = 76.0
)

type setItem struct {
	row   int // the settings row (>= 0) or -1 for a section title
	title string
	y, h  float64 // position inside the scrolling content
}

// settingsLayout lays the sections out: CONTROLES, ACCIONES Y PODERES, JUGADORES, CONTROLES DEL
// JUGADOR 2, IDIOMA, EXTRAS.
func settingsLayout() []setItem {
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
	var ctrl, powers, p2 []int
	for a := actUp; a < actCount; a++ {
		if a < actEvent {
			ctrl = append(ctrl, int(a))
		} else {
			powers = append(powers, int(a))
		}
	}
	for a := 0; a < actPlayerCount; a++ {
		p2 = append(p2, setRowP2+a)
	}
	section("CONTROLES", ctrl...)
	section("ACCIONES Y PODERES", powers...)
	section("JUGADORES", setRowDev1, setRowDev2)
	section("CONTROLES DEL JUGADOR 2", p2...)
	section("IDIOMA", setRowLang)
	section("EXTRAS", setRowCredits, setRowReset, setRowBack)
	return out
}

// settingsRowAt moves from a row to the previous / next one as the screen shows them (the rows are
// not numbered in the order they appear).
func settingsRowAt(sel, dir int) int {
	var order []int
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

// settingsScroll is how far the list is scrolled (screen pixels).
var settingsScroll float64

// settingsEnsureVisible scrolls so that the selected row (and its title when it is the first of a
// section) is inside the window; it also keeps the scroll inside the content.
func settingsEnsureVisible(sel int) {
	l := settingsLayout()
	for i, it := range l {
		if it.row != sel {
			continue
		}
		top, bot := it.y, it.y+it.h
		if i > 0 && l[i-1].row < 0 {
			top = l[i-1].y // show the section title with its first row
		}
		if top < settingsScroll {
			settingsScroll = top
		}
		if bot > settingsScroll+setViewH {
			settingsScroll = bot - setViewH
		}
	}
	settingsScroll = clampF(settingsScroll, 0, math.Max(0, settingsContentH(l)-setViewH))
}

func (g *Game) updateSettings() {
	if g.showCredits {
		if actConfirm.just() || actBack.just() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.showCredits = false
		}
		return
	}
	if settingsJoin > 0 {
		g.updateJoin()
		return
	}
	if g.capturing {
		// the next key (or pad button) pressed becomes the binding; Esc cancels
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.capturing = false
			return
		}
		bd, _ := rowBinding(g.setSel)
		if g.setCol < 2 {
			for _, k := range inpututil.AppendJustPressedKeys(nil) {
				bd.Keys[g.setCol] = k
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
	layout := settingsLayout()
	// the wheel scrolls the list
	if _, wy := ebiten.Wheel(); wy != 0 {
		settingsScroll = clampF(settingsScroll-wy*30, 0, math.Max(0, settingsContentH(layout)-setViewH))
	}
	// the mouse picks rows (hover) and acts on them (click)
	mouseConfirm := false
	if click := g.mouseMenu(func(mx, my float64) int {
		if dbgSetSel || mx < 240 || mx > 920 || my < setTop || my >= setTop+setViewH {
			return -1
		}
		for _, it := range layout {
			if it.row < 0 {
				continue
			}
			if y := setTop + it.y - settingsScroll; my >= y && my < y+it.h {
				return it.row
			}
		}
		return -1
	}, &g.setSel); click {
		mx, _ := ebiten.CursorPosition()
		if isBindRow(g.setSel) {
			for c, cx := range [3]float64{540, 680, 820} {
				if math.Abs(float64(mx)-cx) < 65 {
					g.setCol, mouseConfirm = c, true
				}
			}
		} else {
			mouseConfirm = true
		}
	}
	if actDown.just() {
		g.setSel = settingsRowAt(g.setSel, 1)
		settingsEnsureVisible(g.setSel)
	}
	if actUp.just() {
		g.setSel = settingsRowAt(g.setSel, -1)
		settingsEnsureVisible(g.setSel)
	}
	if g.setSel == setRowLang && (actRight.just() || actLeft.just()) {
		gameLang = otherLang(gameLang)
		uiLang = gameLang
		saveLanguages()
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
	if isBindRow(g.setSel) {
		if actRight.just() {
			g.setCol = (g.setCol + 1) % 3
		}
		if actLeft.just() {
			g.setCol = (g.setCol + 2) % 3
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if isBindRow(g.setSel) {
			bd, _ := rowBinding(g.setSel)
			if g.setCol < 2 {
				bd.Keys[g.setCol] = noKey
			} else {
				bd.Pad = noPad
			}
			saveBindings()
		}
	}
	if actConfirm.just() || mouseConfirm {
		switch {
		case isBindRow(g.setSel):
			if g.setCol == 2 && !padConnected() {
				return // no gamepad connected: nothing to capture
			}
			g.capturing = true
		case g.setSel == setRowDev1:
			settingsJoin = 1
		case g.setSel == setRowDev2:
			settingsJoin = 2
		case g.setSel == setRowLang:
			gameLang = otherLang(gameLang)
			uiLang = gameLang
			saveLanguages()
		case g.setSel == setRowCredits:
			g.showCredits = true
		case g.setSel == setRowReset:
			binds, binds2 = defaultBindings(), defaultBindings2()
			devPref = [2]int{devAutoPref, devAutoPref}
			refreshPlayerDevices()
			saveBindings()
		default:
			g.mode = ModeMenu
		}
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

// updateJoin: the player that is joining takes the device of the next key or pad button pressed.
func (g *Game) updateJoin() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
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

func (g *Game) drawSettings(dst *ebiten.Image) {
	g.drawBedroom(dst)
	g.drawLogo(dst, 160, 30)
	if g.showCredits { // the credits cover the list: nothing of the settings is drawn under them
		g.drawCredits(dst)
		return
	}
	fillRect(dst, 230, 40, 700, 460, rgba(0x05070e, 232))
	fillRect(dst, 230, 40, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 499, 700, 1, rgba(0x6a4ab0, 160))
	g.drawTextC(dst, "AJUSTES", 580, 50, rgb(0xff6a8a), 2)
	cols := [3]float64{540, 680, 820}
	for i, h := range []string{"Tecla 1", "Tecla 2", "Mando"} {
		g.drawTextC(dst, h, cols[i], setColHeadY, rgb(0x8a86a0), 1.2)
	}
	fillRect(dst, 240, setTop-6, 680, 1, rgba(0x6a4ab0, 110))
	fillRect(dst, 240, setTop+setViewH+2, 680, 1, rgba(0x6a4ab0, 110))
	settingsEnsureVisible(g.setSel)
	layout := settingsLayout()
	// everything that scrolls is clipped to the window
	view := dst.SubImage(image.Rect(230, int(setTop)-4, 930, int(setTop+setViewH)+2)).(*ebiten.Image)
	for _, it := range layout {
		y := setTop + it.y - settingsScroll
		if y+it.h < setTop-4 || y > setTop+setViewH+2 {
			continue
		}
		if it.row < 0 {
			g.drawText(view, it.title, 252, y+8, rgb(0xffa030), 1.2)
			fillRect(view, 252+g.textW(it.title, 1.2)+10, y+16, 920-(252+g.textW(it.title, 1.2)+10), 1, rgba(0xffa030, 90))
			continue
		}
		i := it.row
		sel := i == g.setSel
		col := rgb(0x3fb59a)
		if sel {
			col = rgb(0xff6a8a)
			fillRect(view, 242, y, 676, it.h-2, rgba(0x6a4ab0, 60))
		}
		ty := y + 5
		if i == setRowDev1 || i == setRowDev2 {
			p := i - setRowDev1
			dc := rgb(0xffe060)
			g.drawText(view, "Entrada J"+fmt.Sprint(p+1), 262, ty, col, 1.3)
			fillRect(view, 242, y+3, 4, it.h-8, playerColor(p))
			name := devPrefName(devPref[p])
			if devPref[p] != devAutoPref && !padConnectedFor(devPref[p]) {
				name, dc = name+" ("+tr("desconectado")+")", rgb(0xff6a6a)
			} else if devPref[p] == devAutoPref {
				name += " → " + deviceName(autoDevices()[p])
			}
			g.drawTextC(view, "◄ "+name+" ►", 740, ty, dc, 1.15)
			continue
		}
		if isBindRow(i) {
			bd, a := rowBinding(i)
			g.drawText(view, actionNames[a], 262, ty, col, 1.3)
			if i >= setRowP2 {
				fillRect(view, 242, y+3, 4, it.h-8, playerColor(1))
			}
			vals := [3]string{keyLabel(bd.Keys[0]), keyLabel(bd.Keys[1]), padLabel(bd.Pad)}
			for c := 0; c < 3; c++ {
				vc := rgb(0xb8b0d0)
				if sel && g.setCol == c {
					vc = rgb(0xffe060)
					fillRect(view, cols[c]-60, y+2, 120, it.h-6, rgba(0x6a4ab0, 110))
					if g.capturing {
						vals[c] = "pulsa..."
					}
				}
				g.drawTextC(view, vals[c], cols[c], ty, vc, 1.15)
			}
			continue
		}
		if i == setRowLang {
			g.drawText(view, "Idioma / Language", 262, ty, col, 1.3)
			name := "Español"
			if gameLang == "en" {
				name = "English"
			}
			g.drawTextC(view, "◄ "+name+" ►", 680, ty, rgb(0xffe060), 1.3)
			continue
		}
		label := "RESTAURAR VALORES"
		if i == setRowBack {
			label = "VOLVER"
		}
		if i == setRowCredits {
			label = "CRÉDITOS"
		}
		g.drawText(view, label, 262, ty, col, 1.3)
	}
	// scroll marks
	total := settingsContentH(layout)
	if total > setViewH {
		barH := math.Max(24, setViewH*setViewH/total)
		barY := setTop + (setViewH-barH)*settingsScroll/(total-setViewH)
		fillRect(dst, 924, setTop, 2, setViewH, rgba(0x6a4ab0, 60))
		fillRect(dst, 923, barY, 4, barH, rgba(0xffa030, 190))
		if settingsScroll > 1 {
			g.drawTextC(dst, "↑", 902, setTop-18, rgb(0xffa030), 1.2)
		}
		if settingsScroll < total-setViewH-1 {
			g.drawTextC(dst, "↓", 902, setTop+setViewH+6, rgb(0xffa030), 1.2)
		}
	}
	hint := "↑↓ fila   ←→ columna / valor   Enter: cambiar   Supr: borrar   Esc: volver"
	if g.capturing {
		hint = "Pulsa la nueva tecla o botón...   Esc: cancelar"
	}
	if g.setSel == setRowDev1 || g.setSel == setRowDev2 {
		hint = "←→ elegir   Enter: unirte (pulsa una tecla o un botón del mando)   Esc: volver"
	}
	if settingsJoin > 0 {
		hint = "Jugador " + fmt.Sprint(settingsJoin) + ": pulsa un botón para unirte (una tecla = teclado)   Esc: cancelar"
	}
	g.drawTextC(dst, hint, 580, 466, rgb(0xf0e060), 1.1)
	if !g.showCredits { // under the credits panel it would show half hidden
		g.drawText(dst, "Atajos fijos: 1-9 nivel · R aleatoria · G generado · M sonido · F11 pantalla · F2 editor", 238, 484, rgb(0x5a5670), 1)
	}
	if g.showCredits {
		g.drawCredits(dst)
	}
}

// gameVersion is shown in the main menu and in the credits.
const gameVersion = "1.0"

func (g *Game) drawCredits(dst *ebiten.Image) {
	fillRect(dst, 240, 60, 680, 430, rgb(0x05070e))
	fillRect(dst, 240, 60, 680, 1, rgb(0xff6a8a))
	fillRect(dst, 240, 489, 680, 1, rgb(0xff6a8a))
	g.drawTextC(dst, "CRÉDITOS", 580, 84, rgb(0xff6a8a), 2.2)
	g.drawTextC(dst, "Party Hard · Fiesta Nocturna", 580, 122, rgb(0xf4f0ff), 1.6)
	g.drawTextC(dst, "Versión "+gameVersion+" · 2026", 580, 146, rgb(0x8a86a0), 1.2)
	g.drawWrapped(dst, "Juego hecho como homenaje al Party Hard original, desarrollado por Pinokl Games y distribuido por tinyBuild. Este proyecto es de fans y no tiene ninguna relación oficial con ellos.", 290, 178, 580, rgb(0xb8b0d0))
	rows := [][2]string{{"Director", "Frange"}, {"IA", "Claude"}, {"Testers", "Diego, César y Samuel"}, {"Año", "2026"}}
	for i, r := range rows {
		y := 262 + float64(i)*34
		g.drawText(dst, r[0], 330, y, rgb(0x3fb59a), 1.6)
		g.drawText(dst, r[1], 500, y, rgb(0xffe060), 1.6)
	}
	g.drawTextC(dst, "Gracias por jugar", 580, 420, rgb(0xff6a8a), 1.4)
	g.drawTextC(dst, actionHint(actConfirm)+" / "+actionHint(actBack)+": volver", 580, 446, rgb(0xf0e060), 1.1)
}

var _ = fmt.Sprint
