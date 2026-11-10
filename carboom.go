package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The F1 car keeps doing exactly what it always did (it comes in from the side,
// weaving, and cuts across the party). The only addition: where it used to
// slip out of the screen and vanish, it now blows up with a short animation.
// The blast is only a show: it hurts nobody (the car still runs over people on
// its way, as before) and leaves nothing behind.
// ---------------------------------------------------------------------------

const f1BoomTime = 6.0 // seconds the pieces and the smoke last

type carPart struct {
	x, y, z    float64
	vx, vy, vz float64
	kind       int // 0 wheel, 1 body panel, 2 wing, 3 helmet
	col        color.RGBA
	spin, rot  float64
	rest       bool
}

type f1Boom struct {
	t     float64
	parts []carPart
}

// f1AtEdge: the car has reached the far edge of the screen (where it used to leave).
func (g *Game) f1AtEdge(e *Event) bool {
	if e.vx > 0 {
		return e.x >= g.lvl.W-60
	}
	return e.x <= 60
}

// f1Explode: the car blows up (called once).
func (g *Game) f1Explode(e *Event) {
	e.boom = &f1Boom{}
	d := f1Drivers[e.n]
	g.snd.Play("boom")
	g.decals = append(g.decals, Decal{pos: Vec{e.x, e.y}, kind: 1, r: 15}) // the burnt spot stays (damage.go stamps it)
	g.shake = 0.45
	g.burst(Vec{e.x, e.y - 12}, 30, 130, rgb(0xff9030), 0.8, 3, 60)
	g.burst(Vec{e.x, e.y - 10}, 20, 150, rgb(0xffd060), 0.5, 2, 140)
	g.burst(Vec{e.x, e.y - 14}, 16, 70, rgb(0x2a2a2e), 1.3, 4, -26)
	// the pieces of the car (its own colours) fly off in every direction
	body, acc, hel := c(d.body), c(d.accent), c(d.helmet)
	mk := func(kind int, col color.RGBA) {
		a := g.rng.Float64() * 2 * math.Pi
		s := 40 + g.rng.Float64()*90
		e.boom.parts = append(e.boom.parts, carPart{x: e.x, y: e.y, z: 8, vx: math.Cos(a) * s, vy: math.Sin(a) * s * 0.45, vz: 120 + g.rng.Float64()*170,
			kind: kind, col: col, spin: (g.rng.Float64() - 0.5) * 18})
	}
	mk(0, c(0x121216))
	mk(0, c(0x121216))
	mk(0, c(0x121216))
	mk(1, body)
	mk(1, shade(body, 0.8))
	mk(2, acc)
	mk(3, hel)
}

// stepF1Boom runs the aftermath (smoke, a few flames, the pieces bouncing); false when it is over.
func (g *Game) stepF1Boom(e *Event, dt float64) bool {
	b := e.boom
	b.t += dt
	if b.t < 1.6 && g.rng.Float64() < dt*14 {
		g.burst(Vec{e.x + (g.rng.Float64()-0.5)*14, e.y - 4}, 1, 40, rgb(0xff8a20), 0.35, 2, -50)
	}
	if g.rng.Float64() < dt*9*math.Max(0.1, 1-b.t/f1BoomTime) {
		g.burst(Vec{e.x + (g.rng.Float64()-0.5)*10, e.y - 8}, 1, 14, color.RGBA{70, 70, 76, 200}, 1.4, 4, -22)
	}
	for i := range b.parts {
		p := &b.parts[i]
		if p.rest {
			continue
		}
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.z += p.vz * dt
		p.vz -= 420 * dt
		p.rot += p.spin * dt
		if p.z <= 0 {
			p.z = 0
			if math.Abs(p.vz) < 40 {
				p.rest = true
				continue
			}
			p.vz = -p.vz * 0.42
			p.vx *= 0.6
			p.vy *= 0.6
			p.spin *= 0.5
		}
	}
	return b.t < 2.4 // the pieces fade out with the show
}

func fillDisc(dst *ebiten.Image, cx, cy, r float64, col color.RGBA) {
	if r < 1 {
		return
	}
	for dy := -r; dy <= r; dy++ {
		w := math.Sqrt(r*r - dy*dy)
		fillRect(dst, math.Round(cx-w), math.Round(cy+dy), math.Round(2*w), 1, col)
	}
}

// drawF1Boom paints the fireball, the shock ring and the flying pieces.
func (g *Game) drawF1Boom(dst *ebiten.Image, e *Event) {
	b := e.boom
	cx, cy := g.wx(e.x), g.wy(e.y)
	t := b.t
	if t < 0.9 {
		grow := math.Min(1, t/0.3)
		fade := 1 - math.Max(0, (t-0.3)/0.6)
		r := 10 + 24*grow
		a := func(v float64) uint8 { return uint8(255 * math.Max(0, math.Min(1, v*fade))) }
		fillDisc(dst, cx, cy-10-8*grow, r, color.RGBA{232, 70, 16, a(0.9)})
		fillDisc(dst, cx, cy-11-9*grow, r*0.72, color.RGBA{255, 160, 30, a(0.95)})
		fillDisc(dst, cx, cy-12-9*grow, r*0.42, color.RGBA{255, 236, 140, a(1)})
		if t < 0.12 {
			fillDisc(dst, cx, cy-12, r*0.9, color.RGBA{255, 255, 255, 230})
		}
	}
	if t < 0.55 {
		rr := 6 + t*230
		a := uint8(220 * (1 - t/0.55))
		for ang := 0.0; ang < 2*math.Pi; ang += 0.07 {
			fillRect(dst, math.Round(cx+math.Cos(ang)*rr), math.Round(cy-3+math.Sin(ang)*rr*0.42), 2, 1, color.RGBA{255, 240, 200, a})
		}
	}
	fadeOut := uint8(255)
	if t > 1.8 {
		fadeOut = uint8(255 * math.Max(0, 1-(t-1.8)/0.6))
	}
	for _, p := range b.parts {
		x, y := g.wx(p.x), g.wy(p.y)-p.z
		if !p.rest && p.z > 0.5 {
			fillRect(dst, math.Round(x-1), math.Round(g.wy(p.y)), 3, 1, color.RGBA{0, 0, 0, 60}) // shadow under a flying piece
		}
		col := p.col
		col.A = fadeOut
		sw := math.Abs(math.Cos(p.rot)) > 0.5
		switch p.kind {
		case 0:
			fillDisc(dst, math.Round(x), math.Round(y-2), 3, color.RGBA{18, 18, 22, fadeOut})
			fillDisc(dst, math.Round(x), math.Round(y-2), 1.2, color.RGBA{120, 120, 130, fadeOut})
		case 1:
			w, h := 8.0, 3.0
			if sw {
				w, h = 3, 6
			}
			fillRect(dst, math.Round(x-w/2), math.Round(y-h/2), w, h, col)
		case 2:
			fillRect(dst, math.Round(x-3), math.Round(y-1), 6, 2, col)
		default:
			fillDisc(dst, math.Round(x), math.Round(y-2), 2.5, col)
		}
	}
}
