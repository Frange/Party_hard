package main

import "math"

// ---------------------------------------------------------------------------
// What must keep clear of the HUD (v1.2.0, rule "SIEMPRE MARGEN"): the panel at the top left and the notes stacked
// under it (countdown, OBJETO, stamina, ambulance, jail) are one block. The speech bubbles never cover it (nor the
// "[E] ..." use prompts: flushBubbles), and the banner of a band hangs at least hudClearGap below it or beside it (on
// the BARCO PIRATA with two players the banner sat 3 px under "Ambulancia: J1 1 · J2 1").
// ---------------------------------------------------------------------------

// hudClearGap: the free space between the HUD block and a banner of the world.
const hudClearGap = 12.0

// hudBlockRect: the screen area the HUD takes while playing (panel + the notes under it). ok = false when nothing is
// shown (not playing, or the panel faded away because somebody walks under it).
func (g *Game) hudBlockRect() (r Rect, ok bool) {
	if g.mode != ModePlay || g.lvl == nil || g.hud.hide >= 0.5 {
		return Rect{}, false
	}
	panelB := g.hud.bottom - 3
	b := panelB
	obj, _, _, jail := g.hudSide()
	if col0 := obj - g.cdLineH(); jail > col0 { // something is stacked under the panel: the last row of text is
		b = jail + 2 // 16 px tall (line height of the font) where the column only moved 14
	}
	b += 12 * float64(g.jailHudLines())
	return Rect{hudX, hudY, hudW, math.Ceil(b - hudY)}, true
}

// jailHudLines: the lines drawJailHud (twojail.go) writes under the ambulance row.
func (g *Game) jailHudLines() int {
	if !g.is2P() || g.mode != ModePlay {
		return 0
	}
	n := 0
	for _, p := range g.players() {
		switch {
		case p.dead:
		case p.jail.held:
			n += 2
		case p.jail.lastLife:
			n++
		}
	}
	return n
}

// bandSignClear: the shift (dx, dy) that keeps the band banner (its box x, y, w, h, poles included) hudClearGap away
// from the HUD block. It first hangs lower on shorter poles (up to the length of the poles, so it never covers the
// musicians' heads); if that is not enough it slides to the right of the block.
func (g *Game) bandSignClear(x, y, w, h, poles float64) (dx, dy float64) {
	hr, ok := g.hudBlockRect()
	if !ok {
		return 0, 0
	}
	grow := Rect{hr.X - hudClearGap, hr.Y - hudClearGap, hr.W + 2*hudClearGap, hr.H + 2*hudClearGap}
	if !rectsOverlap(grow, Rect{x, y, w, h}) {
		return 0, 0
	}
	need := hr.Y + hr.H + hudClearGap - y
	if need <= poles {
		return 0, math.Ceil(need)
	}
	dy = math.Floor(poles)
	return math.Ceil(hr.X + hr.W + hudClearGap - x), dy
}
