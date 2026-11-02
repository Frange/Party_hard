package main

import (
	"fmt"
	"math"
)

// Debug aids for the "invisible wall" bug: a bot that walks at random and a
// detector that reports every place where the player pushes against nothing.

func (g *Game) botMove(dt float64) (float64, float64) {
	g.botT -= dt
	if g.botT <= 0 {
		g.botT = 0.8 + g.rng.Float64()*2.2
		a := g.rng.Float64() * 2 * math.Pi
		g.botDx, g.botDy = math.Cos(a), math.Sin(a)
		if g.rng.Float64() < 0.5 { // straight lines are the common case
			g.botDx, g.botDy = math.Round(g.botDx), math.Round(g.botDy)
		}
	}
	return g.botDx, g.botDy
}

// watchStuck: called every frame the player tries to move. If he has made no
// headway for half a second it logs what the collision box is touching.
func (g *Game) watchStuck(before Vec, dx, dy, dt float64) {
	if !g.walkBot {
		return
	}
	p := g.p
	if math.Abs(p.pos.X-before.X)+math.Abs(p.pos.Y-before.Y) > 0.01 {
		g.stuckT = 0
		return
	}
	// moving along a wall while sliding is fine; only a full stop counts
	g.stuckT += dt
	if g.stuckT < 0.5 || g.stuckT > 0.5+dt {
		return
	}
	box := Rect{p.pos.X - 4, p.pos.Y - 3, 8, 5}
	for i, r := range g.lvl.solidsP {
		if box.Hits(r) || (Rect{p.pos.X + dx*2 - 4, p.pos.Y + dy*2 - 3, 8, 5}).Hits(r) {
			what := "?"
			for _, w := range g.lvl.Walls {
				if w.R == r {
					what = fmt.Sprintf("wall style %d", w.Style)
				}
			}
			for _, pr := range g.lvl.Props {
				if pr.Foot == r {
					what = "prop " + pr.Name
					if pr.gone {
						what += " (GONE)"
					}
				}
			}
			println(fmt.Sprintf("STUCK t=%.1f at %.0f,%.0f dir %.1f,%.1f solid#%d %.0f,%.0f %.0fx%.0f = %s", g.time, p.pos.X, p.pos.Y, dx, dy, i, r.X, r.Y, r.W, r.H, what))
		}
	}
}

// unstick moves a player who is embedded in something solid to the nearest spot where he fits.
func (l *Level) unstick(p *Player) {
	println(fmt.Sprintf("UNSTUCK player at %.0f,%.0f", p.pos.X, p.pos.Y))
	p.pos = l.NearestFree(p.pos)
}

// fitFootprint: a solid footprint must stand on the picture. When the drawing
// ends well above the bottom of the footprint (halfpipe, chalet, cage...) the
// footprint is lifted to the real bottom of the drawing, so nothing invisible
// blocks the way.
func fitFootprint(name string, k *PropKind, sb *pixBuf, oy float64) {
	if k.NoSolid || k.Ambient > 0 || k.FH <= 0 {
		return
	}
	bottom := -1
	for y := sb.h - 1; y >= 0 && bottom < 0; y-- {
		for x := 0; x < sb.w; x++ {
			if sb.get(x, y).A > 40 {
				bottom = y
				break
			}
		}
	}
	if bottom < 0 {
		return
	}
	fy := -k.FH
	if k.hasFoot {
		fy = k.FootDY
	}
	excess := (oy + (fy+k.FH)*wk) - float64(bottom+1) // footprint rows below the last drawn row
	if excess < 3 {
		return
	}
	if !k.hasFoot {
		k.hasFoot = true
		k.FootDX = -k.FW / 2
		k.FootDY = fy
	}
	k.FootDY -= excess / wk
}

var footprintsFitted = map[*PropKind]bool{}

func fitAllFootprints() {
	for name, k := range propKinds {
		if footprintsFitted[k] {
			continue
		}
		footprintsFitted[k] = true
		before := k.FootDY
		if k.Native != nil {
			fitFootprint(name, k, k.Native[0], k.NAY)
		} else if k.Draw != nil {
			fitFootprint(name, k, stylizeSprite(renderPropSprite(name, k, 0)), (float64(k.H)+1)*wk)
		}
		_ = before
	}
}

// botPressE: the bot presses the use key now and then (hides, comes out again, picks things up, arms traps).
func (g *Game) botPressE(dt float64) bool {
	if !g.walkBot {
		return false
	}
	g.botE -= dt
	if g.botE <= 0 {
		g.botE = 0.6 + g.rng.Float64()*4
		return true
	}
	return false
}
