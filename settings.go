package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The settings screen rebinds keyboard keys (two slots per action) and the
// gamepad button of each action. Rows: the actions, then "restaurar", "volver".

const (
	setRowStart   = int(actCount)     // people at the start of a party
	setRowMax     = int(actCount) + 1 // people in the whole party
	setRowLang    = int(actCount) + 2 // language of the game
	setRowCredits = int(actCount) + 3
	setRowReset   = int(actCount) + 4
	setRowBack    = int(actCount) + 5
	setRows       = int(actCount) + 6
)

// the rows are packed tight so all the actions, the limits and the buttons fit
const setRowH = 17.0

func settingsRowY(i int) float64 { return 104 + float64(i)*setRowH }

func (g *Game) updateSettings() {
	if g.showCredits {
		if actConfirm.just() || actBack.just() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.showCredits = false
		}
		return
	}
	if g.capturing {
		// the next key (or pad button) pressed becomes the binding; Esc cancels
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.capturing = false
			return
		}
		a := Action(g.setSel)
		if g.setCol < 2 {
			for _, k := range inpututil.AppendJustPressedKeys(nil) {
				binds[a].Keys[g.setCol] = k
				g.capturing = false
				saveBindings()
				return
			}
			return
		}
		if id, ok := activeGamepad(); ok {
			for _, b := range inpututil.AppendJustPressedStandardGamepadButtons(id, nil) {
				binds[a].Pad = b
				g.capturing = false
				saveBindings()
				return
			}
		}
		return
	}
	// the mouse picks rows (hover) and acts on them (click)
	mouseStep, mouseConfirm := 0, false
	if click := g.mouseMenu(func(mx, my float64) int {
		if mx < 240 || mx > 920 {
			return -1
		}
		for i := 0; i < setRows; i++ {
			if y := settingsRowY(i); my >= y-3 && my < y+setRowH-3 {
				return i
			}
		}
		return -1
	}, &g.setSel); click {
		mx, _ := ebiten.CursorPosition()
		switch {
		case g.setSel < int(actCount):
			for c, cx := range [3]float64{540, 680, 820} {
				if math.Abs(float64(mx)-cx) < 65 {
					g.setCol, mouseConfirm = c, true
				}
			}
		case g.setSel == setRowStart || g.setSel == setRowMax:
			if float64(mx) < 760 {
				mouseStep = -5
			} else {
				mouseStep = 5
			}
		default:
			mouseConfirm = true
		}
	}
	if actDown.just() {
		g.setSel = (g.setSel + 1) % setRows
	}
	if actUp.just() {
		g.setSel = (g.setSel + setRows - 1) % setRows
	}
	if g.setSel == setRowStart || g.setSel == setRowMax {
		step := 0
		if actRight.just() {
			step = 5
		}
		if actLeft.just() {
			step = -5
		}
		if mouseStep != 0 {
			step = mouseStep
		}
		if step != 0 {
			if g.setSel == setRowStart {
				lo := clampI(g.maxStart()+step, 10, 100)
				g.progress.GuestMin = lo
				g.progress.GuestMax = maxI(g.guestMax(), lo) // the minimum never goes above the maximum
			} else {
				g.progress.GuestMax = clampI(g.guestMax()+step, g.maxStart(), 100)
			}
			saveProgress(g.progress)
		}
	}
	if g.setSel == setRowLang && (actRight.just() || actLeft.just()) {
		gameLang = otherLang(gameLang)
		uiLang = gameLang
		saveLanguages()
	}
	if g.setSel < int(actCount) {
		if actRight.just() {
			g.setCol = (g.setCol + 1) % 3
		}
		if actLeft.just() {
			g.setCol = (g.setCol + 2) % 3
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if g.setSel < int(actCount) {
			if g.setCol < 2 {
				binds[g.setSel].Keys[g.setCol] = noKey
			} else {
				binds[g.setSel].Pad = noPad
			}
			saveBindings()
		}
	}
	if actConfirm.just() || mouseConfirm {
		switch {
		case g.setSel < int(actCount):
			if g.setCol == 2 && !padConnected() {
				return // no gamepad connected: nothing to capture
			}
			g.capturing = true
		case g.setSel == setRowStart || g.setSel == setRowMax:
			// changed with the left/right keys
		case g.setSel == setRowLang:
			gameLang = otherLang(gameLang)
			uiLang = gameLang
			saveLanguages()
		case g.setSel == setRowCredits:
			g.showCredits = true
		case g.setSel == setRowReset:
			binds = defaultBindings()
			saveBindings()
		default:
			g.mode = ModeMenu
		}
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

func (g *Game) drawSettings(dst *ebiten.Image) {
	g.drawBedroom(dst)
	g.drawLogo(dst, 160, 30)
	fillRect(dst, 230, 40, 700, 460, rgba(0x05070e, 225))
	fillRect(dst, 230, 40, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 499, 700, 1, rgba(0x6a4ab0, 160))
	g.drawTextC(dst, "AJUSTES", 580, 52, rgb(0xff6a8a), 2)
	cols := [3]float64{540, 680, 820}
	for i, h := range []string{"Tecla 1", "Tecla 2", "Mando"} {
		g.drawTextC(dst, h, cols[i], 90, rgb(0x8a86a0), 1.2)
	}
	for i := 0; i < setRows; i++ {
		y := settingsRowY(i)
		sel := i == g.setSel
		col := rgb(0x3fb59a)
		if sel {
			col = rgb(0xff6a8a)
		}
		if i == setRowStart || i == setRowMax {
			label, val := "Invitados mínimo", g.maxStart()
			if i == setRowMax {
				label, val = "Invitados máximo", g.guestMax()
			}
			g.drawText(dst, label, 252, y, col, 1.3)
			g.drawTextC(dst, fmt.Sprintf("◄ %d ►", val), 760, y, rgb(0xffe060), 1.3)
			continue
		}
		if i < int(actCount) {
			g.drawText(dst, actionNames[i], 252, y, col, 1.3)
			vals := [3]string{keyLabel(binds[i].Keys[0]), keyLabel(binds[i].Keys[1]), padLabel(binds[i].Pad)}
			for c := 0; c < 3; c++ {
				vc := rgb(0xb8b0d0)
				if sel && g.setCol == c {
					vc = rgb(0xffe060)
					fillRect(dst, cols[c]-60, y-2, 120, 17, rgba(0x6a4ab0, 90))
					if g.capturing {
						vals[c] = "pulsa..."
					}
				}
				g.drawTextC(dst, vals[c], cols[c], y, vc, 1.15)
			}
			continue
		}
		if i == setRowLang {
			g.drawText(dst, "Idioma / Language", 252, y, col, 1.3)
			name := "Español"
			if gameLang == "en" {
				name = "English"
			}
			g.drawTextC(dst, "◄ "+name+" ►", 760, y, rgb(0xffe060), 1.3)
			continue
		}
		label := "RESTAURAR VALORES"
		if i == setRowBack {
			label = "VOLVER"
		}
		if i == setRowCredits {
			label = "CRÉDITOS"
		}
		g.drawText(dst, label, 252, y, col, 1.3)
	}
	hint := "↑↓ fila   ←→ columna / valor   Enter: cambiar   Supr: borrar   Esc: volver"
	if g.capturing {
		hint = "Pulsa la nueva tecla o botón...   Esc: cancelar"
	}
	g.drawTextC(dst, hint, 580, 466, rgb(0xf0e060), 1.1)
	g.drawText(dst, "Atajos fijos: 1-9 nivel · R aleatoria · G generado · M sonido · F11 pantalla · F2 editor", 238, 486, rgb(0x5a5670), 1)
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
