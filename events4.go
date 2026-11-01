package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The kids' shows, earned by killing each of them (see trophies.go):
//   scooters  Sam-sam and César lead a line of scooter kids across the party
//   skate     Diego's gang does a skateboard show with ollies
//   prank     César's water-balloon rain
// ---------------------------------------------------------------------------

// kidEventID: the reward that unlocks each show.
var kidEventID = map[int]string{evScooters: "ev:scooters", evSkate: "ev:skate", evPrank: "ev:prank"}

func (g *Game) startKidRun(e *Event) bool {
	ride, slugs, n := 1, []string{"sam_sam", "cesar"}, 6
	e.vx = 200
	e.sp = 26
	if e.kind == evSkate {
		ride, slugs, n = 2, []string{"diego"}, 5
		e.vx = 150
		e.sp = 34
	}
	for i := 0; i < n; i++ {
		f := presetByName(slugs[i%len(slugs)])
		if f == nil {
			return false
		}
		spec := f(g.rng)
		spec.Hair = hx(pick(g.rng, hairTones))
		spec.Shirt = hx(pick(g.rng, shirtTones))
		spec.Shirt2 = spec.Shirt
		spec.RideCol = hx(pick(g.rng, shirtTones))
		spec.Ride = ride
		e.sets = append(e.sets, newCharSet(&spec))
	}
	p := g.lvl.RandomRoam(g.rng)
	e.y = math.Max(110, math.Min(g.lvl.H-90, p.Y))
	e.x = -60
	if g.rng.Intn(2) == 0 {
		e.x, e.vx = g.lvl.W+60, -e.vx
	}
	if e.kind == evSkate {
		g.say("¡Una pandilla de skaters hace una exhibición en plena fiesta!")
	} else {
		g.say("¡Un desfile de niños en patinete cruza la fiesta a toda velocidad!")
	}
	g.snd.Play("alert")
	return true
}

// riderPos: where the i-th kid of the line is at this moment (world position and jump height).
func (g *Game) riderPos(e *Event, i int) (Vec, float64) {
	sign := math.Copysign(1, e.vx)
	pos := Vec{e.x - sign*float64(i)*e.sp, e.y + math.Sin(e.t*2.6+float64(i))*14}
	jump := 0.0
	if e.kind == evSkate {
		jump = math.Abs(math.Sin(e.t*4.3+float64(i)*0.9)) * 15
	}
	return pos, jump
}

func (g *Game) stepKidRun(e *Event, dt float64) bool {
	e.x += e.vx * dt
	sign := math.Copysign(1, e.vx)
	e.boltT -= dt
	for i := range e.sets {
		pos, jump := g.riderPos(e, i)
		if g.rng.Float64() < 0.5 && jump < 3 {
			g.parts = append(g.parts, Particle{pos: Vec{pos.X - sign*4, pos.Y - 1}, vel: Vec{-sign * 20, -8 - g.rng.Float64()*16}, life: 0.3, max: 0.3, col: rgb(0xffe080), size: 2, grav: 60})
		}
		if e.boltT <= 0 {
			for _, gu := range g.guests {
				if gu.out() || gu.perf != "" || gu.state == gScared || gu.state == gLift || gu.state == gZombie || gu.state == gKiller {
					continue
				}
				if dist(gu.pos, pos) < 28 && g.rng.Float64() < 0.6 {
					g.scare(gu)
					gu.timer = 2.5
					gu.icon, gu.iconT = []string{"¡Eh, cuidado!", "¡Qué rápido!", "¡Mis pies!"}[g.rng.Intn(3)], 1.4
				} else if dist(gu.pos, pos) < 70 && gu.icon == "" && g.rng.Float64() < 0.08 {
					gu.icon, gu.iconT = []string{"¡OLÉ!", "¡Wiiii!", "¡Bravo!"}[g.rng.Intn(3)], 1.4
				}
			}
		}
	}
	if e.boltT <= 0 {
		e.boltT = 0.3
	}
	tail := e.x - sign*float64(len(e.sets)-1)*e.sp
	if sign > 0 {
		return tail < g.lvl.W+80
	}
	return tail > -80
}

func (g *Game) drawKidRun(dst *ebiten.Image, e *Event) {
	dir := DirRight
	if e.vx < 0 {
		dir = DirLeft
	}
	// the one in front last, so the line overlaps like a real queue
	for i := len(e.sets) - 1; i >= 0; i-- {
		pos, jump := g.riderPos(e, i)
		g.drawActor(dst, e.sets[i], AnimIdle, 0, pos, dir, -jump, false)
	}
}

// ---- water balloons ---------------------------------------------------------

func (g *Game) startPrank(e *Event) bool {
	n := 14
	for i := 0; i < n; i++ {
		var t Vec
		if gu := g.pickGuest(); gu != nil && g.rng.Float64() < 0.8 {
			t = Vec{gu.pos.X + g.rng.Float64()*24 - 12, gu.pos.Y + g.rng.Float64()*10 - 5}
		} else {
			t = g.lvl.RandomRoam(g.rng)
		}
		e.bolts = append(e.bolts, t)
		e.tgt = append(e.tgt, float64(i)*0.42+g.rng.Float64()*0.2)
		e.done = append(e.done, false)
	}
	g.say("¡César ha preparado una lluvia de globos de agua!")
	g.snd.Play("alert")
	return true
}

const balloonFall = 0.9

func (g *Game) stepPrank(e *Event, dt float64) bool {
	last := 0.0
	for k, t0 := range e.tgt {
		last = math.Max(last, t0)
		if e.done[k] || e.t < t0+balloonFall {
			continue
		}
		e.done[k] = true
		at := e.bolts[k]
		g.burst(Vec{at.X, at.Y - 4}, 14, 70, rgb(0x7ac0f0), 0.5, 2, 120)
		g.snd.Play("splash")
		for _, gu := range g.guests {
			if gu.out() || gu.perf != "" || gu.state == gLift || gu.state == gZombie || gu.state == gKiller {
				continue
			}
			if dist(gu.pos, at) < 24 {
				g.scare(gu)
				gu.timer = 3
				gu.icon, gu.iconT = []string{"¡Está fría!", "¡Mi ropa!", "¡CÉSAR!", "¡Me he mojado!"}[g.rng.Intn(4)], 1.6
			}
		}
	}
	return e.t < last+balloonFall+1.2
}

func (g *Game) drawPrank(dst *ebiten.Image, e *Event) {
	cols := []color.RGBA{rgb(0xe83a3a), rgb(0x3a8ae8), rgb(0xf0d020), rgb(0x40c860), rgb(0xe860c0)}
	for k, t0 := range e.tgt {
		age := e.t - t0
		if age < 0 || age >= balloonFall {
			continue
		}
		f := age / balloonFall
		at := e.bolts[k]
		x, y := g.wx(at.X), g.wy(at.Y)
		filledEllipse(dst, x, y, 2+f*5, 1+f*2, color.RGBA{0, 0, 0, uint8(30 + f*50)})
		by := y - (1-f)*(1-f)*230 - 8
		col := cols[k%len(cols)]
		filledEllipse(dst, x, by, 4, 5, col)
		filledEllipse(dst, x-1.5, by-2, 1.4, 1.8, color.RGBA{255, 255, 255, 190})
		fillRect(dst, x-1, by+5, 2, 2, shade(col, 0.7))
	}
}
