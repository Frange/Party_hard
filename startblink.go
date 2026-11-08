package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Start marker: when a level begins, the ground under each player blinks in his colour (blue / red) so
// that you find your people at once. It lasts sbDur seconds of play, or until somebody moves (but never
// less than sbMin), and fades out at the end. It is painted on the floor layer, before the people.
const (
	sbDur  = 3.0  // seconds of play
	sbMin  = 1.5  // it is shown at least this long, even if the player moves
	sbFade = 0.45 // fade-out
	sbHz   = 7.0  // blinking rate
)

type startBlink struct {
	t     float64
	off   float64 // second at which a player moved (after sbMin), -1 while nobody has
	start [2]Vec
	moved bool
}

func (s *startBlink) reset(g *Game) {
	*s = startBlink{off: -1}
	for i, pl := range g.players() {
		if i < 2 {
			s.start[i] = pl.pos
		}
	}
}

// end is the second at which the marker is gone.
func (s *startBlink) end() float64 {
	if s.off >= 0 {
		return math.Min(sbDur, s.off+sbFade)
	}
	return sbDur
}

func (s *startBlink) active() bool { return s.t < s.end() }

// update counts the play time and notices the first step of any player.
func (s *startBlink) update(g *Game, dt float64) {
	if !s.active() {
		return
	}
	s.t += dt
	if s.off < 0 {
		for i, pl := range g.players() {
			if i < 2 && (pl.moving || dist(pl.pos, s.start[i]) > 3) {
				s.off = math.Max(s.t, sbMin)
				break
			}
		}
	}
}

// alpha is the blink strength (0.35..0.9 while blinking, fading to 0 at the end).
func (s *startBlink) alpha() float64 {
	if !s.active() {
		return 0
	}
	sm := 0.5 + 0.5*math.Sin(2*math.Pi*sbHz*s.t)
	a := 0.4 + 0.55*sm
	if left := s.end() - s.t; left < sbFade {
		a *= left / sbFade
	}
	return a
}

// sbCell is a rectangle of the marker, in world pixels, already cut to the free ground.
type sbCell struct {
	x, y, w float64
	kind    int // 0 outline, 1 body, 2 inner
}

func (l *Level) sbFree(x, y float64) bool {
	if !l.Free(Vec{x, y}) {
		return false
	}
	for _, r := range l.solidsP {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return false
		}
	}
	return true
}

// sbShape lists the rows of the diamond ring around a player's feet, cut to the free ground
// (no walls, no feet of props). pos is the player's feet; sc is the scale of the level.
func (l *Level) sbShape(pos Vec, sc, rx float64) []sbCell {
	ry := rx * 0.52
	var out []sbCell
	for dy := -math.Ceil(ry); dy <= math.Ceil(ry); dy++ {
		var run sbCell
		open := false
		flush := func() {
			if open {
				out = append(out, run)
				open = false
			}
		}
		for dx := -math.Ceil(rx); dx <= math.Ceil(rx); dx++ {
			d := math.Abs(dx)/rx + math.Abs(dy)/ry
			k := -1
			switch {
			case d > 1:
			case d > 0.78:
				k = 0
			case d > 0.3:
				k = 1
			default:
				k = 2
			}
			x, y := math.Round(pos.X+dx), math.Round(pos.Y+dy)
			if k < 0 || !l.sbFree(x, y) {
				flush()
				continue
			}
			if open && run.kind == k && run.x+run.w == x {
				run.w++
				continue
			}
			flush()
			run, open = sbCell{x, y, 1, k}, true
		}
		flush()
	}
	return out
}

func sbRadius(g *Game) float64 {
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	rx := 18 * sc
	if g.is2P() {
		rx = math.Min(rx, playerGap/2-2)
	}
	return rx
}

func premul(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
}

// drawStartBlink paints the markers on the floor (call it before the people are drawn).
func (g *Game) drawStartBlink(dst *ebiten.Image) {
	if g.mode != ModePlay || g.lvl == nil || !g.sb.active() {
		return
	}
	a := g.sb.alpha()
	if a <= 0 {
		return
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	rx := sbRadius(g)
	for i, pl := range g.players() {
		if i > 1 || pl.dead || pl.hidden {
			continue
		}
		col := playerColor(i)
		for _, c := range g.lvl.sbShape(pl.pos, sc, rx) {
			var cc color.RGBA
			switch c.kind {
			case 0:
				cc = premul(color.RGBA{14, 12, 24, 255}, a)
			case 1:
				cc = premul(col, a)
			default:
				cc = premul(color.RGBA{255, 255, 255, 255}, a)
			}
			fillRect(dst, g.wx(c.x), g.wy(c.y), c.w*wk, wk, cc)
		}
		// two small arrows over the head that bob: they point at the player
		bob := math.Round(2 * math.Sin(g.sb.t*9))
		hx, hy := g.wx(pl.pos.X), g.wy(pl.pos.Y-58*sc)+bob
		dark := premul(color.RGBA{14, 12, 24, 255}, a)
		lit := premul(col, a)
		for j := 0; j < 2; j++ {
			y := hy + float64(j)*6
			for k := 0; k < 4; k++ {
				for _, sx := range []float64{-1, 1} {
					x := hx + sx*(6-float64(k)*2) - 1
					fillRect(dst, x-1, y+float64(k)*2-1, 4, 4, dark)
				}
			}
			for k := 0; k < 4; k++ {
				for _, sx := range []float64{-1, 1} {
					x := hx + sx*(6-float64(k)*2) - 1
					fillRect(dst, x, y+float64(k)*2, 2, 2, lit)
				}
			}
		}
	}
}
