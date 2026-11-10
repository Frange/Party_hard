package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---------------------------------------------------------------------------
// HELP BUTTON AND CONTROLS WINDOW.
//
// A small "?" button sits in the top right corner (left of the trophy counter) in the main menu and in the
// levels (normal, stealth, 1 and 2 players). Hovering it shows a one-line tooltip; clicking it, F1 or the
// Select / Back button of any gamepad opens a window with every control: player 1 keyboard, player 2
// keyboard, gamepad, and the modes (stealth box and decoy, traps, gold suit, VIP invitation). The keys
// are read from the CURRENT bindings (settings), never hardcoded.
//
// PAUSE: while the window is open Game.Update returns before anything runs (guests, timers, police, the
// level clock g.time...), exactly like the trophies window: no time passes, so nothing jumps on closing. The
// music keeps looping (it is the menu/level loop, driven by g.time that does not move) and the sound mute
// key still works. The window takes no game input: the keys of the game are not read at all while it is open,
// and the closing key is consumed on the frame it closes (the next frame it is no longer "just pressed").
// Close: the CERRAR button, a click outside, Esc, F1, the Back button (B) or Select.
// ---------------------------------------------------------------------------

const (
	helpBtnW, helpBtnH     = 26.0, 24.0
	helpBtnX, helpBtnY     = trophyHudX - helpBtnW - 12, trophyHudY // 12 px from the trophy counter
	helpPanX, helpPanY     = 30.0, 24.0
	helpPanW, helpPanH     = 900.0, 492.0
	helpCloseW             = 84.0
	helpCloseH             = 22.0
	helpCloseX, helpCloseY = helpPanX + helpPanW - helpCloseW - 12, helpPanY + 10
)

// hardware access for the tests
var (
	helpCursorFn = ebiten.CursorPosition
	helpClickFn  = func() bool { return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) }
	helpKeyFn    = inpututil.IsKeyJustPressed
)

// helpShown: where the button exists (menu and levels; the cinematics do not need it).
func (g *Game) helpShown() bool {
	if g.iconMode {
		return false
	}
	return g.mode == ModeMenu || g.mode == ModePlay
}

// helpPadSelect: Select (Back) of any connected gamepad was just pressed.
func helpPadSelect() bool {
	for _, id := range padList() {
		if padJustFn(id, ebiten.StandardGamepadButtonCenterLeft) {
			return true
		}
	}
	return false
}

func helpOverBtn(mx, my int) bool {
	fx, fy := float64(mx), float64(my)
	return fx >= helpBtnX && fx < helpBtnX+helpBtnW && fy >= helpBtnY && fy < helpBtnY+helpBtnH
}

// updateHelp opens / closes the window. It returns true while the window is open: the rest of the frame
// is skipped, which is what pauses the game.
func (g *Game) updateHelp() bool {
	if g.helpOv {
		if !g.helpShown() {
			g.helpOv = false
			return false
		}
		closeIt := helpKeyFn(ebiten.KeyEscape) || helpKeyFn(ebiten.KeyF1) || actBack.justEither() || helpPadSelect()
		if helpClickFn() {
			mx, my := helpCursorFn()
			fx, fy := float64(mx), float64(my)
			outside := fx < helpPanX || fx > helpPanX+helpPanW || fy < helpPanY || fy > helpPanY+helpPanH
			onX := fx >= helpCloseX && fx < helpCloseX+helpCloseW && fy >= helpCloseY && fy < helpCloseY+helpCloseH
			closeIt = closeIt || outside || onX
		}
		if closeIt {
			g.helpOv = false
		}
		return true
	}
	if !g.helpShown() || g.trophyOv || g.showCredits {
		return false
	}
	open := helpKeyFn(ebiten.KeyF1) || helpPadSelect()
	if helpClickFn() {
		if mx, my := helpCursorFn(); helpOverBtn(mx, my) {
			open = true
		}
	}
	if open {
		g.helpOv = true
		return true
	}
	return false
}

// ---- drawing ---------------------------------------------------------------------------------------------

func (g *Game) drawHelp(dst *ebiten.Image) {
	if g.helpShown() && !g.trophyOv {
		mx, my := helpCursorFn()
		hot := helpOverBtn(mx, my) && !g.helpOv
		bg := rgba(0x05070e, 190)
		if hot {
			bg = rgba(0x2a1a4a, 230)
		}
		fillRect(dst, helpBtnX, helpBtnY, helpBtnW, helpBtnH, bg)
		fillRect(dst, helpBtnX, helpBtnY, helpBtnW, 1, rgba(0x6a4ab0, 200))
		fillRect(dst, helpBtnX, helpBtnY+helpBtnH-1, helpBtnW, 1, rgba(0x6a4ab0, 200))
		qx, qy := helpGlyphPos()
		g.drawText(dst, "?", qx, qy, rgb(0xf0e060), helpGlyphScale)
		if hot { // tooltip under the button, right aligned to the screen edge
			s := "Ayuda: controles (F1 / Select)"
			w := g.textW(s, 1) + 12
			x := float64(screenW) - w - 16
			y := helpBtnY + helpBtnH + 4
			fillRect(dst, x, y, w, 18, rgba(0x05070e, 235))
			fillRect(dst, x, y, w, 1, rgb(0x6a4ab0))
			fillRect(dst, x, y+17, w, 1, rgb(0x6a4ab0))
			g.drawText(dst, s, x+6, y+4, rgb(0xe8e4f4), 1)
		}
	}
	if g.helpOv {
		g.drawHelpWindow(dst)
	}
}

const helpCapH = 18.0

// helpModeRowH: the rows of MODOS Y OBJETOS (eight fit above the footer, a cap is 18 high).
const helpModeRowH = 22.0

// the vertical layout of the window (>= 12 px between blocks, 4 px between key rows)
const (
	helpColsY    = helpPanY + 62.0                                 // the titles of the three columns
	helpRowPitch = 24.0                                            // a cap is 18 high
	helpModesY0  = helpColsY + 24 + 6*helpRowPitch + helpCapH + 12 // the rule above MODOS Y OBJETOS
	helpFooterY  = helpPanY + helpPanH - 18                        // the fixed keys and the closing hint
)

// helpCap draws a key as a key: a rectangle with its label. It returns the width.
func (g *Game) helpCap(dst *ebiten.Image, x, y float64, label string, edge color.RGBA) float64 {
	w := g.textW(label, 1) + 10
	if w < 18 {
		w = 18
	}
	fillRect(dst, x, y, w, helpCapH, edge)
	fillRect(dst, x+1, y+1, w-2, helpCapH-3, rgb(0x2a2540))
	fillRect(dst, x+1, y+1, w-2, 1, rgb(0x4a4470))
	g.drawText(dst, label, x+(w-g.textW(label, 1))/2, y+3, rgb(0xf4f0ff), 1)
	return w
}

// capName is the label of a key inside a cap (short: arrows as arrows).
func capName(k ebiten.Key) string {
	switch k {
	case ebiten.KeyArrowUp:
		return "↑"
	case ebiten.KeyArrowDown:
		return "↓"
	case ebiten.KeyArrowLeft:
		return "←"
	case ebiten.KeyArrowRight:
		return "→"
	}
	return keyLabel(k)
}

// actKeyName is the label of the first key a player has for an action (the real binding, never typed by
// hand), for the written signs of the game. Player 2 with a gamepad would not use the keys: the label of
// the pad button is given then.
func actKeyName(p int, a Action) string {
	b := &binds
	if p == 1 {
		b = &binds2
	}
	for _, k := range b[a].Keys {
		if k != noKey {
			return capName(k)
		}
	}
	return padLabel(b[a].Pad)
}

// helpKeys lists the keys of one action (the first key of each slot that is set).
func helpKeys(b *Binding, all bool) []string {
	var out []string
	for _, k := range b.Keys {
		if k != noKey {
			out = append(out, capName(k))
			if !all {
				break
			}
		}
	}
	return out
}

// helpMoveKeys: the four direction keys of a binding set, up left down right.
func helpMoveKeys(b *Bindings) []string {
	var out []string
	for _, a := range []Action{actUp, actLeft, actDown, actRight} {
		out = append(out, helpKeys(&b[a], false)...)
	}
	return out
}

// helpRow is one line of a column: a label and the caps of its keys.
type helpRow struct {
	label string
	caps  []string
}

// helpKeyboardRows: the controls of one keyboard player (from his current bindings).
func helpKeyboardRows(p int) []helpRow {
	b := &binds
	if p == 1 {
		b = &binds2
	}
	return []helpRow{
		{"Mover", helpMoveKeys(b)},
		{"Correr", helpKeys(&b[actRun], false)},
		{"Golpear", helpKeys(&b[actAttack], true)},
		{"Usar / interactuar", helpKeys(&b[actInteract], true)},
		{"Bailar / señuelo", helpKeys(&b[actDance], true)},
		{"Ayuda", []string{"F1"}},
		{"Menú / salir", helpKeys(&b[actBack], false)},
	}
}

// helpPadRows: the gamepad controls (from the current bindings of the buttons).
func helpPadRows() []helpRow {
	b := &binds
	return []helpRow{
		{"Mover", []string{"Stick", "D-pad"}},
		{"Correr", []string{padLabel(b[actRun].Pad)}},
		{"Golpear", []string{padLabel(b[actAttack].Pad)}},
		{"Usar / interactuar", []string{padLabel(b[actInteract].Pad)}},
		{"Bailar / señuelo", []string{padLabel(b[actDance].Pad)}},
		{"Ayuda", []string{"SELECT"}}, // upper case: tr() would turn the word "Select" into "Elegir"
		{"Menú / salir", []string{padLabel(b[actBack].Pad)}},
	}
}

// helpModeLine: a feature of a mode, the action that drives it and what it does.
type helpModeLine struct {
	act  Action
	text string
}

var helpModes = []helpModeLine{
	{actInteract, "Sigilo: ponerte o quitarte la caja de cartón"},
	{actDance, "Sigilo, 2 jugadores: señuelo, haz ruido junto a un guardia"},
	{actInteract, "Trampas: cerca de una, la armas (2,5 s) y salta al pasar alguien"},
	{actInteract, "Traje dorado: recógelo y nadie te ve durante 8 s"},
	{actInteract, "Invitación VIP: quita la tarjeta dorada; ayuda al VIP que da propina"},
	{actDance, "Bailar: pulsa quieto; si te mueves, paras"},
	{actInteract, helpUseNear}, // (pe_*.go, stash.go, partylife.go)
}

// helpUseNear: the other uses of E in the party: pizza boy, mascot, live DJ, waiter, magician, stashes.
const helpUseNear = "Cerca de: pizzero, mascota, DJ (canción), camarero (copa), mago (propina), escondites"

// helpMinimapText: the minimap row (minimap.go). TAB and Start are not rebindable, so they are fixed caps.
const helpMinimapText = "Minimapa: muestra u oculta el mapa de la fiesta"

// helpGeneral: the fixed keys (not rebindable).
const helpGeneral = "Fijas: R reiniciar fiesta · M sonido · F11 pantalla completa"

func (g *Game) helpColumn(dst *ebiten.Image, x, y float64, title string, edge color.RGBA, rows []helpRow) {
	g.drawText(dst, title, x, y, edge, 1)
	fillRect(dst, x, y+14, 262, 1, edge)
	for i, r := range rows {
		ry := y + 24 + float64(i)*helpRowPitch
		g.drawText(dst, r.label, x, ry+3, rgb(0xd8d0f0), 1)
		cx := x + 128
		for _, c := range r.caps {
			cx += g.helpCap(dst, cx, ry, c, edge) + 3
		}
	}
}

func (g *Game) drawHelpWindow(dst *ebiten.Image) {
	fillRect(dst, 0, 0, float64(screenW), float64(screenH), rgb(0x05030a)) // opaque: the paused game stays hidden behind
	fillRect(dst, helpPanX, helpPanY, helpPanW, helpPanH, rgb(0x0e0a1a))
	fillRect(dst, helpPanX, helpPanY, helpPanW, 2, rgb(0x6a4ab0))
	fillRect(dst, helpPanX, helpPanY+helpPanH-2, helpPanW, 2, rgb(0x6a4ab0))
	g.drawText(dst, "CONTROLES Y AYUDA", helpPanX+20, helpPanY+10, rgb(0xf0e060), 2)
	g.drawText(dst, "El juego está en pausa", helpPanX+20, helpPanY+38, rgb(0xa89cd0), 1)
	// close button
	mx, my := helpCursorFn()
	hot := float64(mx) >= helpCloseX && float64(mx) < helpCloseX+helpCloseW && float64(my) >= helpCloseY && float64(my) < helpCloseY+helpCloseH
	bg := rgba(0x6a1a2a, 230)
	if hot {
		bg = rgba(0xa02a40, 255)
	}
	fillRect(dst, helpCloseX, helpCloseY, helpCloseW, helpCloseH, bg)
	g.drawTextC(dst, "CERRAR", helpCloseX+helpCloseW/2, helpCloseY+5, rgb(0xffd0d8), 1)

	y := helpColsY
	g.helpColumn(dst, helpPanX+20, y, "JUGADOR 1 · TECLADO", playerColor(0), helpKeyboardRows(0))
	g.helpColumn(dst, helpPanX+20+300, y, "JUGADOR 2 · TECLADO", playerColor(1), helpKeyboardRows(1))
	g.helpColumn(dst, helpPanX+20+600, y, "MANDO", rgb(0x70d890), helpPadRows())

	// the modes
	my0 := helpModesY0
	fillRect(dst, helpPanX+20, my0, helpPanW-40, 1, rgb(0x4a4470))
	g.drawText(dst, "MODOS Y OBJETOS", helpPanX+20, my0+8, rgb(0xf0e060), 1)
	if twoPlayerMode {
		g.drawText(dst, "Azul: jugador 1 · Rojo: jugador 2 · Verde: mando", helpPanX+330, my0+8, rgb(0xa89cd0), 1)
	}
	for i, m := range helpModes {
		ly := my0 + 28 + float64(i)*helpModeRowH
		cx := helpPanX + 20.0
		for _, c := range helpKeys(&binds[m.act], false) {
			cx += g.helpCap(dst, cx, ly, c, playerColor(0)) + 3
		}
		if twoPlayerMode {
			for _, c := range helpKeys(&binds2[m.act], false) {
				cx += g.helpCap(dst, cx, ly, c, playerColor(1)) + 3
			}
		}
		cx += g.helpCap(dst, cx, ly, padLabel(binds[m.act].Pad), rgb(0x70d890)) + 10
		g.drawText(dst, m.text, cx, ly+3, rgb(0xe8e4f4), 1)
	}
	{ // the minimap: TAB / Start (fixed keys)
		ly := my0 + 28 + float64(len(helpModes))*helpModeRowH
		cx := helpPanX + 20.0
		cx += g.helpCap(dst, cx, ly, "Tab", playerColor(0)) + 3
		cx += g.helpCap(dst, cx, ly, "Start", rgb(0x70d890)) + 10
		g.drawText(dst, helpMinimapText, cx, ly+3, rgb(0xe8e4f4), 1)
	}
	g.drawText(dst, helpGeneral, helpPanX+20, helpFooterY, rgb(0xa89cd0), 1)
	hint := tr("Cerrar con") + " Esc, F1, " + padLabel(binds[actBack].Pad) + ", Select"
	g.drawText(dst, hint, helpPanX+helpPanW-20-g.textW(hint, 1), helpFooterY, rgb(0xa89cd0), 1)
}

func init() {
	for es, en := range map[string]string{
		"Ayuda: controles (F1 / Select)": "Help: controls (F1 / Select)",
		"CONTROLES Y AYUDA":              "CONTROLS AND HELP",
		"El juego está en pausa":         "The game is paused",
		"CERRAR":                         "CLOSE",
		"Cerrar con":                     "Close with",
		"JUGADOR 1 · TECLADO":            "PLAYER 1 · KEYBOARD",
		"JUGADOR 2 · TECLADO":            "PLAYER 2 · KEYBOARD",
		"MANDO":                          "GAMEPAD",
		"Mover":                          "Move",
		"Correr":                         "Run",
		"Golpear":                        "Smack",
		"Usar / interactuar":             "Use / interact",
		"Bailar / señuelo":               "Dance / decoy",
		"Ayuda":                          "Help",
		"Menú / salir":                   "Menu / quit",
		"MODOS Y OBJETOS":                "MODES AND ITEMS",
		"Azul: jugador 1 · Rojo: jugador 2 · Verde: mando":                     "Blue: player 1 · Red: player 2 · Green: pad",
		"Sigilo: ponerte o quitarte la caja de cartón":                         "Stealth: put the cardboard box on or off",
		"Sigilo, 2 jugadores: señuelo, haz ruido junto a un guardia":           "Stealth, 2 players: decoy, make noise near a guard",
		"Trampas: cerca de una, la armas (2,5 s) y salta al pasar alguien":     "Traps: near one, you arm it (2.5 s) and it goes off when someone passes",
		"Traje dorado: recógelo y nadie te ve durante 8 s":                     "Gold suit: pick it up and nobody sees you for 8 s",
		"Invitación VIP: quita la tarjeta dorada; ayuda al VIP que da propina": "VIP invitation: take the golden card; help the VIP who tips",
		"Bailar: pulsa quieto; si te mueves, paras":                            "Dance: press standing still; move and you stop",
		helpMinimapText: "Minimap: show or hide the map of the party",
		helpGeneral:     "Fixed: R restart party · M sound · F11 full screen",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

// menuCornerMargin: distance of the version lines of the main menu to the right and bottom edges.
const menuCornerMargin = 16.0

// menuVersionLine is the first line of the bottom right corner of the main menu.
func menuVersionLine() string { return versionLabel() + " - Frange" }

const menuHomageLine = "Homenaje a Party Hard (Pinokl Games · tinyBuild)"

// drawMenuCorner: the two lines at the bottom right of the main menu, both right aligned to the same edge.
func (g *Game) drawMenuCorner(dst *ebiten.Image) {
	for i, l := range []struct {
		s string
		c color.RGBA
	}{{menuVersionLine(), rgb(0xb8b0d0)}, {menuHomageLine, rgb(0x7a7690)}} {
		g.drawText(dst, l.s, float64(screenW)-menuCornerMargin-g.textW(l.s, 1), float64(screenH)-43+float64(i)*14, l.c, 1)
	}
}

// helpGlyphScale: the "?" of the button (a whole scale).
const helpGlyphScale = 2.0

var qInk struct {
	done           bool
	x0, y0, x1, y1 float64 // the painted pixels of the glyph, from the top left corner of the text box
}

// glyphInk measures the painted pixels of a glyph of the game font (no window needed), relative to the top left
// corner where text.Draw is given the text: that is what must be centred, not the advance box.
func glyphInk(r rune) (x0, y0, x1, y1 float64) {
	f := bitmapfont.Face
	dr, mask, maskp, _, ok := f.Glyph(fixed.Point26_6{}, r)
	if !ok {
		return 0, 0, 0, 0
	}
	asc := text.NewGoXFace(f).Metrics().HAscent
	minX, minY, maxX, maxY := 1<<30, 1<<30, -1<<30, -1<<30
	for y := dr.Min.Y; y < dr.Max.Y; y++ {
		for x := dr.Min.X; x < dr.Max.X; x++ {
			if _, _, _, a := mask.At(maskp.X+x-dr.Min.X, maskp.Y+y-dr.Min.Y).RGBA(); a > 0 {
				minX, maxX = min(minX, x), max(maxX, x+1)
				minY, maxY = min(minY, y), max(maxY, y+1)
			}
		}
	}
	if maxX < minX {
		return 0, 0, 0, 0
	}
	return float64(minX), float64(minY) + asc, float64(maxX), float64(maxY) + asc
}

// helpGlyphPos is the text position that puts the painted "?" exactly in the middle of the button.
func helpGlyphPos() (float64, float64) {
	if !qInk.done {
		qInk.x0, qInk.y0, qInk.x1, qInk.y1 = glyphInk('?')
		qInk.done = true
	}
	s := pxScale(helpGlyphScale)
	return math.Round(helpBtnX + helpBtnW/2 - (qInk.x0+qInk.x1)/2*s), math.Round(helpBtnY + helpBtnH/2 - (qInk.y0+qInk.y1)/2*s)
}
