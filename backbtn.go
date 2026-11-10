package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// BACK button of the menu screens: "← ATRÁS" at the bottom left, the same place and size on every screen
// (party list, settings (+ credits), shop (+ wardrobe), trophies (+ album), cinematics list, debug, the
// "erase everything?" popup). It never shows on the main menu (nothing before it), in a level or in the editor
// (which has its own "Volver al juego" button). Pressing it does exactly what Esc / B does on that screen:
// for one frame actBack reads as pressed (backInject), so every screen keeps its own back logic.
// Keyboard / pad focus: Backspace or LB moves the focus to the button, Enter / A presses it, any direction
// gives the focus back to the screen.
// ---------------------------------------------------------------------------

const (
	backBtnX, backBtnY = 16.0, 500.0 // 16 px from the left and bottom edges
	backBtnW, backBtnH = 104.0, 24.0
)

var (
	backCursorFn = ebiten.CursorPosition
	backClickFn  = func() bool { return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) }
	backFocusFn  = func() bool {
		return inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || padJustPressed(ebiten.StandardGamepadButtonFrontTopLeft)
	}
	backInject bool // this frame the button was pressed: actBack.just() and escJust() answer true
	backFocus  bool // the button has the keyboard / pad focus: Enter / A presses it and not the screen's row
)

// escJust: Esc, or the back button pressed this frame (the screens that read Esc directly use this).
func escJust() bool { return backInject || inpututil.IsKeyJustPressed(ebiten.KeyEscape) }

// hasBackButton: the modes that show the button.
func hasBackButton(m Mode) bool {
	switch m {
	case ModeLevelSelect, ModeSettings, ModeShop, ModeTrophies, ModeScenes, ModeDebug, ModeConfirm:
		return true
	}
	return false
}

func backOver(mx, my int) bool {
	fx, fy := float64(mx), float64(my)
	return fx >= backBtnX && fx < backBtnX+backBtnW && fy >= backBtnY && fy < backBtnY+backBtnH
}

// backUpdate runs at the start of the frame of a menu mode and sets backInject when the button was pressed.
func (g *Game) backUpdate() {
	backInject = false
	if !hasBackButton(g.mode) {
		backFocus = false
		return
	}
	if g.capturing { // the settings are waiting for a key: nothing else may take it
		backFocus = false
	}
	pressed := false
	if backClickFn() {
		mx, my := backCursorFn()
		pressed = backOver(mx, my)
	}
	if backFocus {
		held := backFocus
		backFocus = false // read the raw Enter / A, then restore
		if actConfirm.just() {
			pressed = true
		}
		backFocus = held
		if actUp.just() || actDown.just() || actLeft.just() || actRight.just() {
			backFocus = false
		}
	} else if !g.capturing && backFocusFn() {
		backFocus = true
	}
	if pressed {
		backInject = true
		backFocus = false
	}
}

// drawBackButton: the button, drawn over everything else of the screen.
func (g *Game) drawBackButton(dst *ebiten.Image) {
	if !hasBackButton(g.mode) {
		return
	}
	mx, my := backCursorFn()
	hot := backFocus || backOver(mx, my)
	bg, edge, tc := rgba(0x05070e, 225), rgba(0x6a4ab0, 220), rgb(0xf0e060)
	if hot {
		bg, edge, tc = rgba(0x2a1a4a, 240), rgb(0xff6a8a), rgb(0xffffff)
	}
	fillRect(dst, backBtnX, backBtnY, backBtnW, backBtnH, bg)
	fillRect(dst, backBtnX, backBtnY, backBtnW, 1, edge)
	fillRect(dst, backBtnX, backBtnY+backBtnH-1, backBtnW, 1, edge)
	fillRect(dst, backBtnX, backBtnY, 1, backBtnH, edge)
	fillRect(dst, backBtnX+backBtnW-1, backBtnY, 1, backBtnH, edge)
	g.drawTextC(dst, "← ATRÁS", backBtnX+backBtnW/2, backBtnY+4, tc, 1)
}
