package main

import "github.com/hajimehoshi/ebiten/v2"

// ---------------------------------------------------------------------------
// The HUD panel (top left) is ALWAYS shown, and hides only while a PLAYER is drawn under it, so that it never
// covers the one you control. Only the players count (owner, v1.1.1): the top left corner of every party is full
// of guests, and when every guest, cop or animal hid it the panel was never seen (v1.1.0 bug). The draw of the
// players (drawPlayerOne, drawPrisoner) arms hudTouch around their own sprite (hudPlayerDraw); updateHUD
// (stepHide) turns that into a short fade (hudFadeT) with a minimum time before the panel may come back
// (hudHoldT), so that somebody walking along the edge does not make it flicker. The notes of the column under the
// panel (ambulance line / notice box) follow the same fade. Nothing here touches the game logic.
// ---------------------------------------------------------------------------

const (
	hudFadeT   = 0.15 // seconds to go from shown to hidden and back
	hudHoldT   = 0.25 // seconds without any character over the panel before it comes back
	hudHideGap = 6.0  // safety margin around the panel
)

// hudHideRect: the screen box that hides the panel when a character touches it (panel + safety margin).
func (g *Game) hudHideRect() (x0, y0, x1, y1 float64) {
	b := g.hud.bottom
	if b < hudY+hudBaseH+3 {
		b = hudY + hudBaseH + 3
	}
	return hudX - hudHideGap, hudY - hudHideGap, hudX + hudW + hudHideGap, b - 3 + hudHideGap
}

// hudTouch: a character is drawn in the screen box (x0,y0)-(x1,y1). Called by drawActor/drawLying; it counts only
// while a player is being drawn (g.hud.armed, see hudPlayerDraw): guests, cops, animals... never hide the panel.
func (g *Game) hudTouch(x0, y0, x1, y1 float64) {
	if !g.hud.armed || g.hud.hit || g.mode != ModePlay {
		return
	}
	rx0, ry0, rx1, ry1 := g.hudHideRect()
	if x0 < rx1 && x1 > rx0 && y0 < ry1 && y1 > ry0 {
		g.hud.hit = true
	}
}

// hudPlayerDraw runs fn (the draw of one player) with hudTouch armed, so that his sprite may hide the panel.
func (g *Game) hudPlayerDraw(fn func()) {
	prev := g.hud.armed
	g.hud.armed = true
	defer func() { g.hud.armed = prev }()
	fn()
}

func (h *hudState) stepHide(dt float64) {
	if h.hit {
		h.hold = hudHoldT
		h.hit = false
	} else if h.hold > 0 {
		h.hold -= dt
	}
	step := dt / hudFadeT
	if h.hold > 0 {
		h.hide = min(1, h.hide+step)
	} else {
		h.hide = max(0, h.hide-step)
	}
}

// hudAlpha: 1 shown ... 0 hidden. Outside the party itself (results, lose screen, pause window) it is always shown.
func (g *Game) hudAlpha() float64 {
	if g.mode != ModePlay || g.pause.open {
		return 1
	}
	return 1 - g.hud.hide
}

// withHudFade paints with fn on dst at the HUD alpha: directly when shown, not at all when hidden, through a
// screen-sized buffer (made once) while it fades.
func (g *Game) withHudFade(dst *ebiten.Image, fn func(*ebiten.Image)) {
	a := g.hudAlpha()
	switch {
	case a >= 1:
		fn(dst)
	case a <= 0:
	default:
		if g.hudBuf == nil {
			g.hudBuf = ebiten.NewImage(screenW, screenH)
		}
		g.hudBuf.Clear()
		fn(g.hudBuf)
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(float32(a))
		dst.DrawImage(g.hudBuf, op)
	}
}
