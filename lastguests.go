package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// LAST GUESTS MARKER. When lastGuestsMax or fewer people are left alive (the same count that ends the party,
// g.aliveCount(): guests, guards and hunting killers, not the VIP staff nor the arsonist), every one of them is
// marked so that he stands out of the crowd:
//   - on the floor: a double ring (gold + white, dark outline) that pulses at 2 Hz (drawLastRings, before the people);
//   - over the head: a big gold arrow (down) with a dark outline that bobs 0..3 px in whole steps (drawLastArrows,
//     over the people, under the HUD); with ONE person left the arrow is bigger and a small "ÚLTIMO"/"LAST" sits on it;
//   - off screen: a big pulsing chevron stuck to the border of the screen pointing his way;
//   - on the minimap: his 2x2 dot blinks gold/white (minimap.go).
// An emote never covers the arrow (emoteBlocked reads lastMarks). Not in STEALTH, nor in the cinematics.
// ---------------------------------------------------------------------------

const (
	lastGuestsMax = 3    // from this many left, they are marked
	lastEdgeM     = 20.0 // distance of the chevron to the border
	lastHz        = 2.0  // pulse rate of ring and arrow
	lastArrowHalf = 7    // half width of the arrow (15 px wide); with one person left lastArrowHalfOne
	lastArrowOne  = 10
)

var (
	lastGold = color.RGBA{255, 214, 64, 255}
	lastPale = color.RGBA{255, 250, 220, 255}
	lastDark = color.RGBA{14, 12, 24, 255}
)

func init() {
	for k, v := range map[string]string{"ÚLTIMO": "LAST"} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

// lastMark: what the last frame drew (the tests read it; the emotes read the rectangle to keep clear of it).
type lastMark struct {
	arrow      bool    // chevron on the border (else ring + arrow over the head)
	x, y       float64 // chevron position / screen position of the person for a ring
	ang        float64 // direction of the chevron
	alpha      float64
	cellsDrawn int     // pixels of the ring painted
	rx, ry     float64 // rectangle of the arrow (and its text) over the head
	rw, rh     float64
	tipY       float64 // y of the tip of the arrow over the head
	half       int     // half width of the arrow
	label      bool    // "ÚLTIMO" over the arrow
}

// lastTargets: where the people that are still to fall are (feet), only while the marker applies. The slice is reused.
func (g *Game) lastTargets() []Vec {
	if g.mode != ModePlay || g.stealth || g.lvl == nil {
		return nil
	}
	n := g.aliveCount()
	if n < 1 || n > lastGuestsMax {
		return nil
	}
	out := g.lastBuf[:0]
	for _, gu := range g.guests {
		if !gu.out() && gu.py == nil {
			out = append(out, gu.pos)
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && gd.vip == 0 {
			out = append(out, gd.pos)
		}
	}
	for _, s := range g.allSlashers() {
		if s.state == slHunt || s.state == slArrive {
			out = append(out, s.pos)
		}
	}
	g.lastBuf = out
	return out
}

// lastScreen: where the target's body is on the screen.
func (g *Game) lastScreen(pos Vec) (float64, float64) {
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	return g.wx(pos.X), g.wy(pos.Y - 14*sc)
}

// lastEdgeArrow: for a point (sx, sy) outside the screen, where the chevron goes (on the border, lastEdgeM inside)
// and the angle it points to (radians, 0 = right, y down). ok is false when the point is on the screen.
func lastEdgeArrow(sx, sy float64) (x, y, ang float64, ok bool) {
	if sx >= 0 && sx <= screenW && sy >= 0 && sy <= screenH {
		return 0, 0, 0, false
	}
	cx, cy := float64(screenW)/2, float64(screenH)/2
	dx, dy := sx-cx, sy-cy
	t := math.Inf(1)
	if dx != 0 {
		t = math.Min(t, (cx-lastEdgeM)/math.Abs(dx))
	}
	if dy != 0 {
		t = math.Min(t, (cy-lastEdgeM)/math.Abs(dy))
	}
	return math.Round(cx + dx*t), math.Round(cy + dy*t), math.Atan2(dy, dx), true
}

// lastPhase: sign of the 2 Hz square pulse (true = first half).
func (g *Game) lastPhase() bool { return math.Sin(2*math.Pi*lastHz*g.time) > 0 }

// lastBob: the arrow rises 0..3 px in whole steps.
func (g *Game) lastBob() float64 {
	return math.Round(1.5 + 1.5*math.Sin(2*math.Pi*lastHz*g.time))
}

// lastAlpha: 0.85..1, fast pulse.
func (g *Game) lastAlpha() float64 { return 0.925 + 0.075*math.Sin(2*math.Pi*lastHz*g.time) }

// ring colour classes
const (
	lcNone = iota
	lcDark
	lcGold
	lcPale
)

// lastRingClass: the class of a point at normalised distance d (diamond metric) from the centre: outer ring,
// dark gap, inner ring, each with a dark outline. a and b are the colours of the outer and inner rings.
func lastRingClass(d float64, a, b int) int {
	switch {
	case d > 1.09:
		return lcNone
	case d > 1.0:
		return lcDark
	case d > 0.86:
		return a
	case d > 0.78:
		return lcDark
	case d > 0.64:
		return b
	case d > 0.56:
		return lcDark
	}
	return lcNone
}

// drawLastRings: the double ring on the floor under every marked person who is on screen. It also records, for
// the emotes, the rectangle of the arrow that goes over his head.
func (g *Game) drawLastRings(dst *ebiten.Image) {
	g.lastMarks = g.lastMarks[:0]
	ts := g.lastTargets()
	if len(ts) == 0 {
		return
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	one := len(ts) == 1
	a := g.lastAlpha()
	ph := g.lastPhase()
	ca, cb := lcGold, lcPale
	rx := 21 * sc
	if ph {
		ca, cb = lcPale, lcGold
		rx = 24 * sc
	}
	ry := rx * 0.52
	cols := [4]color.RGBA{{}, premul(lastDark, a), premul(lastGold, a), premul(lastPale, a)}
	l := g.lvl
	for _, p := range ts {
		sx, sy := g.lastScreen(p)
		if _, _, _, off := lastEdgeArrow(sx, sy); off {
			continue
		}
		n := 0
		for dy := -math.Ceil(ry); dy <= math.Ceil(ry); dy++ {
			runX, runW, runC := 0.0, 0.0, lcNone
			for dx := -math.Ceil(rx); dx <= math.Ceil(rx)+1; dx++ {
				c := lcNone
				if dx <= math.Ceil(rx) {
					c = lastRingClass(math.Abs(dx)/rx+math.Abs(dy)/ry, ca, cb)
					x, y := math.Round(p.X+dx), math.Round(p.Y+dy)
					if c != lcNone && !l.sbFree(x, y) {
						c = lcNone
					}
				}
				x := math.Round(p.X + dx)
				if c == runC && c != lcNone && runX+runW == x {
					runW++
					continue
				}
				if runC != lcNone {
					fillRect(dst, g.wx(runX), g.wy(math.Round(p.Y+dy)), runW*wk, wk, cols[runC])
					n += int(runW)
				}
				runX, runW, runC = x, 1, c
			}
		}
		m := lastMark{x: sx, y: sy, alpha: a, cellsDrawn: n, label: one, half: lastArrowHalf}
		if one {
			m.half = lastArrowOne
		}
		m.rows(g, p, sc)
		g.lastMarks = append(g.lastMarks, m)
	}
}

// rows fills the geometry of the arrow over the head of the person at p (feet).
func (m *lastMark) rows(g *Game, p Vec, sc float64) {
	cx := math.Round(g.wx(p.X))
	head := math.Round(g.wy(p.Y)) - math.Round(40*sc)
	tip := head - 3 - g.lastBob()
	top := tip - float64(m.half+1)
	if m.label {
		top -= 13
	}
	if top < 3 { // near the top of the screen: the arrow slides down over the head instead of out of the picture
		tip += 3 - top
		top = 3
	}
	w := float64(2*m.half + 5)
	if m.label && w < 48 {
		w = 48
	}
	m.tipY = tip
	m.rx, m.ry, m.rw, m.rh = cx-math.Round(w/2), top-1, w, tip-top+3
}

// drawDownArrow: a big arrow pointing down, its tip at (cx, tipY), with a 1 px dark outline.
func drawDownArrow(dst *ebiten.Image, cx, tipY float64, half int, a float64) {
	dark := premul(lastDark, a)
	gold := premul(lastGold, a)
	pale := premul(lastPale, a)
	rows := half + 1
	y0 := tipY - float64(rows) + 1
	fillRect(dst, cx-float64(half)-1, y0-1, float64(2*half+3), 1, dark) // top edge
	for k := 0; k < rows; k++ {
		w := float64(2*(half-k) + 1)
		fillRect(dst, cx-w/2+0.5-1, y0+float64(k), w+2, 1, dark)
	}
	fillRect(dst, cx-1, tipY+1, 3, 1, dark) // the point
	for k := 0; k < rows; k++ {
		w := float64(2*(half-k) + 1)
		c := gold
		if k < 2 {
			c = pale
		}
		fillRect(dst, cx-float64(half-k), y0+float64(k), w, 1, c)
	}
}

// drawLastArrows: the big arrow over every marked head, and a big chevron on the border for those off screen.
func (g *Game) drawLastArrows(dst *ebiten.Image) {
	ts := g.lastTargets()
	if len(ts) == 0 {
		return
	}
	a := g.lastAlpha()
	reach := 10.0
	if len(ts) == 1 {
		reach = 14
	}
	if g.lastPhase() {
		reach += 3
	}
	for i := range g.lastMarks { // those on screen, found by drawLastRings
		m := &g.lastMarks[i]
		if m.arrow {
			continue
		}
		cx := m.rx + math.Round(m.rw/2)
		drawDownArrow(dst, cx, m.tipY, m.half, 1)
		if m.label {
			g.bigText(dst, "ÚLTIMO", cx, m.ry, lastGold, lastDark, 1)
		}
	}
	for _, p := range ts {
		if x, y, ang, off := lastEdgeArrow(g.lastScreen(p)); off {
			g.lastMarks = append(g.lastMarks, lastMark{arrow: true, x: x, y: y, ang: ang, alpha: a})
			drawChevron(dst, x, y, ang, a, reach)
		}
	}
}

// drawChevron: a big ">" turned to ang, at (x, y), thick, gold with a dark outline; it grows with the fast pulse
// (and more with one person left).
func drawChevron(dst *ebiten.Image, x, y, ang, a, reach float64) {
	thick := 4.0
	if reach > 12 {
		thick = 5
	}
	tx, ty := x+math.Cos(ang)*6, y+math.Sin(ang)*6
	for pass := 0; pass < 2; pass++ {
		col, grow := premul(lastDark, a), 2.0
		if pass == 1 {
			col, grow = premul(lastGold, a), 0
		}
		for _, side := range []float64{-1, 1} {
			back := ang + math.Pi + side*0.75
			for k := 0.0; k <= reach; k += 1.5 {
				px := math.Round(tx + math.Cos(back)*k)
				py := math.Round(ty + math.Sin(back)*k)
				fillRect(dst, px-thick/2-grow/2, py-thick/2-grow/2, thick+grow, thick+grow, col)
			}
		}
	}
}

// dbgLastGuests (-lastguests N): half a second into the party everybody but N people disappears, to look at the
// marker (N 1..3) or at the slow motion of the last kill (N 0: the party ends by itself). -1 = off. Debug flags freeze progress.
var dbgLastGuests = -1

func (g *Game) lastGuestsDemo() {
	if dbgLastGuests < 0 || g.time < 0.5 || g.mode != ModePlay {
		return
	}
	keep := dbgLastGuests
	dbgLastGuests = -1
	for _, gu := range g.guests {
		if gu.out() || gu.py != nil {
			continue
		}
		if keep > 0 && gu.perf == "" {
			keep--
			continue
		}
		gu.gone = true
	}
	for _, gd := range g.guards {
		if gd.vip == 0 {
			gd.dead = true
		}
	}
	g.checkDone()
}
