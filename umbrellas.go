package main

import (
	"hash/fnv"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// UMBRELLAS IN THE STORM (idea 296, the user: "cuando haya tormentas, que saquen paraguas de muchos colores (que no
// molesten a la jugabilidad)").
//
// While it rains (the storm of weather.go, not the snowfall) SOME of the guests in the open (umbShare, about 45 %)
// open an umbrella of their own colour: red, blue, yellow, green, pink, orange, purple, teal; plain, striped, dotted or
// in two colours. They open one by one as the rain grows (each one at his own intensity) and close one by one when it
// stops; under a roof, swimming, lying, running in panic or in a zombie state it is closed.
//
// It is ONLY a drawing: a small canopy (15x6 px at scale 1) held over the head with a thin handle, inside the box of the
// guest's own sprite (it never reaches farther than the guest himself, so it never covers the HUD more than he does),
// with a soft sway while he walks and a few drops bouncing on it. Nothing is solid, nothing changes for the game (he can
// be killed, he sees and is seen as always). An umbrella that would cover a player is not drawn at that moment.
// No state, no dice: everything comes from the guest (a hash of his name and his offset) and the storm intensity.
// ---------------------------------------------------------------------------

const (
	umbShare  = 0.45 // guests in the open who carry one
	umbOpenK0 = 0.15 // the storm intensity at which the first ones open ...
	umbOpenK1 = 0.50 // ... and the last ones
)

// umbPalette: the colours of the canopies (bright, pretty over the night floors).
var umbPalette = []color.RGBA{
	rgb(0xe8423c), // red
	rgb(0x3a7ae8), // blue
	rgb(0xf4c82a), // yellow
	rgb(0x3cbc5c), // green
	rgb(0xf26aa8), // pink
	rgb(0xf48a2a), // orange
	rgb(0x9a5ae6), // purple
	rgb(0x2abcb2), // teal
}

const (
	umbPlain = iota
	umbStripes
	umbDots
	umbTwoTone
	umbPatterns
)

var (
	umbWhite  = rgb(0xf6f2ea)
	umbHandle = rgb(0x3a2a22)
	umbDrop   = rgb(0xcfe0f6)
)

type umbLook struct {
	col, col2 color.RGBA
	pat       int
	ci        int // index of col in umbPalette (tests)
	side      int // the hand that holds it: +1 right, -1 left
	k         float64
}

// umbHash: a stable number 0..1 of a guest (his name and his offset), n picks one of several.
func umbHash(gu *Guest, n int) float64 {
	h := fnv.New32a()
	h.Write([]byte(gu.name))
	v := math.Sin((gu.off+float64(h.Sum32()%9973)*0.001+float64(gu.style)*0.37)*12.9898+float64(n)*78.233) * 43758.5453
	return v - math.Floor(v)
}

// umbLookOf: the umbrella of a guest (and the intensity at which he opens it).
func umbLookOf(gu *Guest) umbLook {
	ci := int(umbHash(gu, 2) * float64(len(umbPalette)))
	c2 := (ci + 1 + int(umbHash(gu, 3)*float64(len(umbPalette)-1))) % len(umbPalette)
	l := umbLook{col: umbPalette[ci], col2: umbPalette[c2], ci: ci, pat: int(umbHash(gu, 4) * umbPatterns), side: 1,
		k: umbOpenK0 + (umbOpenK1-umbOpenK0)*umbHash(gu, 5)}
	if umbHash(gu, 6) < 0.3 {
		l.side = -1
	}
	return l
}

// umbCarrier: this guest is one of those who brought an umbrella.
func umbCarrier(gu *Guest) bool { return umbHash(gu, 1) < umbShare }

// umbRaining: the storm intensity when it rains (0 = no rain: no storm, a snowfall, or it is over).
func (g *Game) umbRaining() float64 {
	w := &g.wth
	if g.lvl == nil || !w.plan.storm || w.plan.snow || w.done {
		return 0
	}
	return w.k
}

// umbrellaOf: the umbrella the guest holds open now (false = none).
func (g *Game) umbrellaOf(gu *Guest) (umbLook, bool) {
	k := g.umbRaining()
	if k <= 0 || gu == nil || gu.out() || !umbCarrier(gu) {
		return umbLook{}, false
	}
	l := umbLookOf(gu)
	if k < l.k {
		return l, false
	}
	switch gu.state {
	case gToilet, gSleep, gLift, gZombie, gFlee, gScared, gSwim, gKiss, gPunch, gVomit:
		return l, false
	}
	if g.lifeScale() < 0.9 { // the small people of some levels (none of them has a storm): no umbrella too big for them
		return l, false
	}
	if gu.perf != "" || gu.inside != nil || gu.seat != nil || kidRiding(gu) || g.lvl.InPool(gu.pos) || !outdoorAt(g.lvl.Def, gu.pos) {
		return l, false
	}
	return l, true
}

// umbRect: the screen rectangle of an umbrella over a guest whose feet are at (cx, cy) (x0, y0, x1, y1, exclusive end).
func umbRect(cx, cy, px int, k func(float64) int, side int) (int, int, int, int) {
	hx := cx + side*3*px
	return hx - 9*px, cy - k(36) - 6*px, hx + 10*px, cy - k(17) + 2*px
}

// drawUmbrellaPix paints the umbrella: canopy over the head, handle down to the hand. sway: -1, 0, +1 px of the canopy;
// drops: the rain bounces on it (0..1, the storm intensity).
func drawUmbrellaPix(plot plotFn, cx, cy, px int, k func(float64) int, l umbLook, sway int, t, drops float64) {
	ux := cx + l.side*3*px + sway*px
	hx := cx + l.side*3*px // the hand
	bot := cy - k(36)      // the lower edge of the canopy
	// handle: from the canopy straight down to the hand, with the hook at the end
	hTop, hBot := bot, cy-k(17)
	for y := hTop; y < hBot; y += px {
		x := hx
		if y < (hTop+hBot)/2 {
			x = hx + sway*px
		}
		plot(x, y, px, px, umbHandle)
	}
	plot(hx-l.side*px, hBot, px, px, umbHandle)
	// canopy: a dome of 4 rows plus a scalloped edge
	rows := [][2]int{{-3, 3}, {-5, 5}, {-6, 6}, {-7, 7}}
	for r, span := range rows {
		y := bot - (4-r)*px
		for i := span[0]; i <= span[1]; i++ {
			c := umbPanelCol(l, i, r)
			if r == 0 {
				c = shade(c, 1.18)
			}
			plot(ux+i*px, y, px, px, c)
		}
	}
	for _, i := range []int{-7, -6, -1, 0, 1, 6, 7} { // the scallops
		plot(ux+i*px, bot, px, px, shade(umbPanelCol(l, i, 3), 0.72))
	}
	plot(ux, bot-5*px, px, px, umbHandle) // the tip
	// a few drops bounce on it
	if drops > 0.35 {
		for i := 0; i < 2; i++ {
			u := math.Mod(t*1.3+float64(i)*0.5+float64(l.ci)*0.13, 1)
			if u > 0.4 {
				continue
			}
			v := u / 0.4
			s := 1 - 2*i
			x := ux + s*(4+int(math.Round(v*4)))*px
			y := bot - 4*px - int(math.Round(math.Sin(v*math.Pi)*2))*px
			plot(x, y, px, px, umbDrop)
		}
	}
}

// umbPanelCol: the colour of the canopy at column i (-7..7) of row r (patterns).
func umbPanelCol(l umbLook, i, r int) color.RGBA {
	panel := (i + 8) / 4 // 0..3
	switch l.pat {
	case umbStripes:
		if panel%2 == 1 {
			return umbWhite
		}
	case umbTwoTone:
		if panel%2 == 1 {
			return l.col2
		}
	case umbDots:
		if (r == 2 && (i == -4 || i == 3)) || (r == 1 && i == 0) {
			return umbWhite
		}
	}
	return l.col
}

// umbCoversPlayer: the umbrella at that rectangle would cover a player (his sprite box): it is not drawn.
func (g *Game) umbCoversPlayer(x0, y0, x1, y1 int) bool {
	for _, p := range g.players() {
		if p == nil || p.dead {
			continue
		}
		px, py := g.pevXY(p.pos)
		k := g.pevK()
		if x1 > px-k(13) && x0 < px+k(13) && y1 > py-k(45) && y0 < py+k(3) {
			return true
		}
	}
	return false
}

// drawUmbrella: the guest draw loop calls it right after the guest is drawn (draw.go). yoff: sprite px he is lifted.
func (g *Game) drawUmbrella(dst *ebiten.Image, gu *Guest, pos Vec, yoff float64) {
	l, ok := g.umbrellaOf(gu)
	if !ok {
		return
	}
	k := g.pevK()
	px := g.pevPx()
	cx, cy := g.pevXY(pos)
	cy += int(math.Round(yoff * g.lifeScale()))
	x0, y0, x1, y1 := umbRect(cx, cy, px, k, l.side)
	if g.umbCoversPlayer(x0, y0, x1, y1) {
		return
	}
	sway := 0
	if gu.moving {
		sway = int(math.Round(math.Sin(g.time*4 + gu.off*3)))
	}
	drawUmbrellaPix(g.plotter(dst), cx, cy, px, k, l, sway, g.time+gu.off, g.umbRaining())
}
