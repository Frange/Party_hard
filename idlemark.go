package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// "WHERE AM I": a player who stands still for idleShowAfter seconds (no movement, no stab, no dance) gets a
// bobbing gold arrow over his head, the same one that marks the last guests. It goes away as soon as he moves
// or acts. Each player has his own timer (two players). The state is the idle time of the Player; no allocation.
// ---------------------------------------------------------------------------

const idleShowAfter = 10.0

// updateIdle runs once per frame in play for every player.
func (g *Game) updateIdle(dt float64) {
	for _, p := range g.players() {
		if p == nil {
			continue
		}
		p.idleT = idleStep(p.idleT, dt, dist(p.pos, p.idlePos) > 0.01 || p.stabT > 0 || p.danceT > 0 || p.knockT > 0 || p.dead || p.hidden)
		p.idlePos = p.pos
	}
}

// idleStep: the idle timer after one frame; any activity resets it.
func idleStep(t, dt float64, active bool) float64 {
	if active {
		return 0
	}
	return t + dt
}

// drawIdleMark paints the arrow over p when he has been idle long enough.
func (g *Game) drawIdleMark(dst *ebiten.Image, p *Player) {
	if p.idleT < idleShowAfter || p.dead || p.hidden || g.mode != ModePlay {
		return
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	bob := math.Floor(math.Mod(g.time*3, 2)) * 2 // whole pixels, no flicker
	drawDownArrow(dst, math.Round(g.wx(p.pos.X)), math.Round(g.wy(p.pos.Y))-44*sc-8+bob, 6, 1)
}
