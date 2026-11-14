package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// PAUSE WINDOW (Esc / START during a party; ○ or START closes it).
//
// Esc (or START of the pad) in an active party opens a small window over the dimmed game with three rows:
// CONTINUAR, AJUSTES and SALIR A ELEGIR FIESTA. Any of the two players can open it. While it is open
// Game.Update returns before anything of the party runs (like the help window): guests, timers, police,
// events and the level clock g.time do not move. The music keeps looping.
//   - CONTINUAR, Esc, ○ or START again: back to the party (✕ picks a row).
//   - AJUSTES: the normal settings screen; its ATRÁS / Esc comes back to this window (the party is still
//     loaded and paused; g.time is held while the settings are open, so the RETO time limit does not run).
//   - SALIR A ELEGIR FIESTA: the party is dropped and the party list opens. Nothing is added: the points of
//     the abandoned party are not banked, no win / defeat is recorded (RETO DE HOY: leaving was already free,
//     only dying / being caught locks it, and that is kept).
// Not in the editor test runs (fromEditor: Esc still goes back to the editor), cinematics, newspaper, game
// over or menus: only ModePlay opens it.
// ---------------------------------------------------------------------------

const (
	pauseRowContinue = iota
	pauseRowSettings
	pauseRowQuit
	pauseRows
)

var pauseLabels = [pauseRows]string{"CONTINUAR", "AJUSTES", "SALIR A ELEGIR FIESTA"}

const (
	pausePanX, pausePanY   = 280.0, 140.0
	pausePanW, pausePanH   = 400.0, 244.0
	pauseRowX, pauseRowW   = 310.0, 340.0
	pauseRowY0, pauseRowDY = 218.0, 50.0
	pauseRowH              = 40.0
)

// pauseState lives in Game.pause.
type pauseState struct {
	open     bool
	sel      int
	inSet    bool    // the settings were opened from this window
	heldT    float64 // g.time when the settings were opened (held while they are open)
	mx, my   int     // last cursor position (hover only changes the row when the mouse moves)
	mouseSet bool
	keepSong bool // SALIR A ELEGIR FIESTA: the song of the abandoned party goes on in the party list (syncMusic)
}

// pauseInput: what the window reads in one frame (a function so the tests can feed it).
type pauseInput struct {
	toggle, up, down, ok, click bool
	back                        bool // ○ / Esc of the menus: only closes the window (in a party ○ is "interact")
	mx, my                      int
}

// pauseNow (-pausenow): opens the window by itself a moment after the party starts (captures).
var pauseNow bool

var pauseInFn = func() pauseInput {
	mx, my := ebiten.CursorPosition()
	in := pauseInput{
		toggle: actBack.justKeysEither() || pauseStartPressed(), // Esc / P / START: the pad's ○ is interact in a party
		back:   actBack.justEither(),
		up:     actUp.justEither(),
		down:   actDown.justEither(),
		ok:     actConfirm.justEither(),
		click:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		mx:     mx, my: my,
	}
	return in
}

// pauseStartPressed: Start of any connected pad.
func pauseStartPressed() bool {
	for _, id := range padList() {
		if padJustFn(id, ebiten.StandardGamepadButtonCenterRight) {
			return true
		}
	}
	return false
}

func pauseRowAt(mx, my int) int {
	fx, fy := float64(mx), float64(my)
	if fx < pauseRowX || fx >= pauseRowX+pauseRowW {
		return -1
	}
	for i := 0; i < pauseRows; i++ {
		y := pauseRowY0 + float64(i)*pauseRowDY
		if fy >= y && fy < y+pauseRowH {
			return i
		}
	}
	return -1
}

// pauseCanOpen: an active party, with no other window on top.
func (g *Game) pauseCanOpen() bool {
	return g.mode == ModePlay && !g.fromEditor && !g.helpOv && !g.trophyOv && !g.iconMode
}

// updatePause opens / runs the window. True while it is open (and on the frame it opens or closes): the rest
// of the frame is skipped, which is what pauses the party.
func (g *Game) updatePause() bool {
	if !g.pause.open {
		if !g.pauseCanOpen() {
			return false
		}
		in := pauseInFn()
		if in.toggle || (pauseNow && g.time > 0.5) {
			pauseNow = false
			g.pause = pauseState{open: true, mx: in.mx, my: in.my, mouseSet: true}
			return true
		}
		return false
	}
	if g.mode != ModePlay { // something else took the screen
		g.pause = pauseState{}
		return false
	}
	in := pauseInFn()
	if in.toggle || in.back {
		g.pause.open = false
		return true
	}
	if in.up {
		g.pause.sel = (g.pause.sel + pauseRows - 1) % pauseRows
	}
	if in.down {
		g.pause.sel = (g.pause.sel + 1) % pauseRows
	}
	pick := in.ok
	moved := !g.pause.mouseSet || in.mx != g.pause.mx || in.my != g.pause.my
	g.pause.mx, g.pause.my, g.pause.mouseSet = in.mx, in.my, true
	if r := pauseRowAt(in.mx, in.my); r >= 0 {
		if moved {
			g.pause.sel = r
		}
		if in.click {
			g.pause.sel, pick = r, true
		}
	}
	if pick {
		g.pauseChoose(g.pause.sel)
	}
	return true
}

func (g *Game) pauseChoose(row int) {
	switch row {
	case pauseRowContinue:
		g.pause.open = false
	case pauseRowSettings:
		g.pause.open, g.pause.inSet, g.pause.heldT = false, true, g.time // pauseAfterSettings opens it again
		g.setSel, g.setCol, g.capturing = 0, 0, false
		settingsTab = 0
		g.mode = ModeSettings
	case pauseRowQuit:
		g.pause = pauseState{keepSong: true}
		lvl := g.levelIdx
		g.banked = true // abandoned: its points are not banked (loadLevel would bank them)
		g.loadLevel(0)
		g.menuSel = g.campaignMenuSel()
		if s := campaignSlotOf(lvl); s >= 0 && lvl < g.progress.Unlocked {
			g.menuSel = s // the row of the abandoned party (campaign_retired.go)
		}
		g.mode = ModeLevelSelect
	}
}

// pauseAfterSettings runs after the settings screen each frame: the level clock stays where it was, and
// leaving the settings (ATRÁS / Esc) comes back to the pause window instead of the main menu.
func (g *Game) pauseAfterSettings() {
	if !g.pause.inSet {
		return
	}
	g.time = g.pause.heldT
	if g.mode == ModeSettings {
		return
	}
	g.pause.inSet = false
	g.showCredits = false
	g.mode = ModePlay
	g.pause.open = true
}

// ---- drawing ---------------------------------------------------------------------------------------------

func (g *Game) drawPause(dst *ebiten.Image) {
	if !g.pause.open || g.mode != ModePlay {
		return
	}
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, 150))
	g.recPanel(pausePanX, pausePanY, pausePanW, pausePanH)
	fillRect(dst, pausePanX, pausePanY, pausePanW, pausePanH, rgba(0x05070e, 235))
	fillRect(dst, pausePanX, pausePanY, pausePanW, 2, rgb(0x6a4ab0))
	fillRect(dst, pausePanX, pausePanY+pausePanH-2, pausePanW, 2, rgb(0x6a4ab0))
	cx := pausePanX + pausePanW/2
	g.drawTextC(dst, "PAUSA", cx, pausePanY+18, rgb(0xff6a8a), 3)
	for i := 0; i < pauseRows; i++ {
		y := pauseRowY0 + float64(i)*pauseRowDY
		col := rgb(0x3fb59a)
		if i == g.pause.sel {
			fillRect(dst, pauseRowX, y, pauseRowW, pauseRowH, rgb(0x2a1a4a))
			fillRect(dst, pauseRowX, y, pauseRowW, 2, rgb(0xffffff))
			fillRect(dst, pauseRowX, y+pauseRowH-2, pauseRowW, 2, rgb(0xffffff))
			col = rgb(0xff6a8a)
		}
		g.drawTextC(dst, pauseLabels[i], cx, y+4, col, 2)
	}
}
