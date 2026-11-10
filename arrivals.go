package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Late arrivals: limousines (street levels) and elevators (roof levels) bring new guests,
// which raises the number of people you have to kill.
// ---------------------------------------------------------------------------

type Elevator struct {
	pos   Vec // centre of the door at the foot of the wall
	t     float64
	n     int
	state int    // 0 waiting for the ding, 1 opening, 2 people leave, 3 closing
	kid   string // a kid arriving instead of ordinary guests
	out   int
}

func (g *Game) updateArrivals(dt float64) {
	if g.testVictim != nil || g.arrivals >= 3 || g.guestRoom() < 2 || g.time < g.arrivalT || len(g.guests) < 4 {
		g.updateElevators(dt)
		return
	}
	g.arrivals++
	g.arrivalT = g.time + 45 + g.rng.Float64()*45
	n := 2 + g.rng.Intn(4)
	def := g.lvl.Def
	if def.Police == "heli" {
		if e := g.makeElevator(n); e != nil {
			g.elevators = append(g.elevators, e)
			g.say("Se oye el ascensor: llegan más invitados")
		}
	} else {
		cnt := 0
		for _, o := range g.veh {
			if o.state != 2 {
				cnt++
			}
		}
		// pad (unused on ground vehicles) = the paint: the second limousine of the night is silver (limo.go)
		g.veh = append(g.veh, &Vehicle{kind: 6, x: def.CarFrom.X + 60, y: def.CarFrom.Y, tx: def.CarTo.X - 40 - float64(cnt)*110, ty: def.CarFrom.Y, alt: n, pad: (g.arrivals + 1) % 2})
		g.say("Una limusina llega a la fiesta")
		g.snd.Play("ring")
	}
	g.updateElevators(dt)
}

func (g *Game) makeElevator(n int) *Elevator {
	for tries := 0; tries < 60; tries++ {
		p := g.lvl.RandomRoam(g.rng)
		if dist(p, g.p.pos) < 180 {
			continue
		}
		y := p.Y
		for y > 40 && !g.lvl.SolidP(p.X, y-6) {
			y -= 4
		}
		if y <= 40 {
			continue
		}
		// need open floor in front and room on both sides
		front := Vec{p.X, y + 12}
		if g.lvl.SolidP(front.X, front.Y) || g.lvl.SolidP(p.X-14, y+4) || g.lvl.SolidP(p.X+14, y+4) {
			continue
		}
		return &Elevator{pos: Vec{p.X, y}, n: n}
	}
	return nil
}

func (g *Game) updateElevators(dt float64) {
	out := g.elevators[:0]
	for _, e := range g.elevators {
		e.t += dt
		switch e.state {
		case 0:
			if e.t > 2.0 {
				e.state, e.t = 1, 0
				g.snd.Play("alert")
			}
		case 1:
			if e.t > 0.8 {
				e.state, e.t = 2, 0
			}
		case 2:
			if e.out < e.n && e.t > 0.6 {
				e.t = 0
				e.out++
				if e.kid != "" {
					g.addKidAt(e.kid, Vec{e.pos.X, e.pos.Y + 12})
				} else {
					g.addGuestAt(Vec{e.pos.X + float64(e.out%3-1)*5, e.pos.Y + 12})
				}
			}
			if e.out >= e.n && e.t > 1.0 {
				e.state, e.t = 3, 0
			}
		case 3:
			if e.t > 0.8 {
				continue
			}
		}
		out = append(out, e)
	}
	g.elevators = out
}

func (g *Game) addGuestAt(pos Vec) {
	if g.guestRoom() < 1 {
		return // the party is full
	}
	pos = g.lvl.NearestFree(pos)
	cs, name := g.newGuestCS()
	if vc := g.venueGuestCS(g.lvl.Def); vc != nil { // dressed for the party (venuecrowd.go)
		cs, name = vc, ""
	}
	gu := &Guest{pos: pos, dir: DirDown, cs: cs, name: name, style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 0.5, role: g.rng.Intn(2)}
	g.guests = append(g.guests, gu)
	g.total++
}

// limoArrives: the car has parked and the guests step out of its side doors (limo.go).
func (g *Game) limoArrives(v *Vehicle) {
	n := v.alt
	for k := 0; k < n; k++ {
		g.addGuestAt(Vec{v.x + limoDoorWorld(k) + float64(k/len(limoDoorX))*6, v.y - 24 + float64(k%2)*8})
	}
	g.say("Llegan más invitados: +" + itoa(n))
	g.snd.Play("alert")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func (g *Game) drawElevator(dst *ebiten.Image, e *Elevator) {
	x, y := math.Round(g.wx(e.pos.X)), math.Round(g.wy(e.pos.Y))
	fillRect(dst, x-13, y-34, 26, 34, color.RGBA{60, 62, 70, 255})
	fillRect(dst, x-11, y-31, 22, 31, color.RGBA{20, 20, 26, 255})
	open := 0.0
	switch e.state {
	case 1:
		open = e.t / 0.8
	case 2:
		open = 1
	case 3:
		open = 1 - e.t/0.8
	}
	if open > 0 {
		fillRect(dst, x-10, y-30, 20, 30, color.RGBA{250, 230, 150, 255})
	}
	w := 10 * (1 - open)
	fillRect(dst, x-10, y-30, w, 30, color.RGBA{150, 156, 168, 255})
	fillRect(dst, x+10-w, y-30, w, 30, color.RGBA{150, 156, 168, 255})
	fillRect(dst, x-3, y-37, 6, 2, color.RGBA{255, 200, 60, 255})
}

// ---------------------------------------------------------------------------
// Little gags: selfies, air guitar, air drums, trips, and a streaker now and then.
// ---------------------------------------------------------------------------

func (g *Game) guestFun(gu *Guest) {
	gu.state = gFun
	gu.fun = g.rng.Intn(11)
	gu.path = nil
	switch gu.fun {
	case 0:
		gu.timer = 4
		gu.icon, gu.iconT = "CLIC!", 1
	case 1:
		gu.timer = 5
		gu.icon, gu.iconT = "GUITARRAZO!", 2
	case 2:
		gu.timer = 5
		gu.icon, gu.iconT = "TATATA!", 2
	case 3:
		gu.timer = 3.4
	case 5:
		gu.timer = 4
		gu.icon, gu.iconT = "JAJAJA", 2
	case 6:
		gu.timer = 3.5
		gu.icon, gu.iconT = "¡OLÉ!", 2
	case 7:
		gu.timer = 3
		gu.icon, gu.iconT = "¡MIRA!", 2
	case 8:
		gu.timer = 6
		gu.icon, gu.iconT = "¡BREAK!", 2
	case 9:
		gu.timer = 6
		gu.icon, gu.iconT = "BIP BOP", 2
	case 10:
		gu.timer = 5
		gu.icon, gu.iconT = "♥", 2
	default:
		gu.timer = 4
		gu.icon, gu.iconT = "WOOOO!", 2
	}
}

func (g *Game) updateFun(gu *Guest, dt float64) {
	gu.timer -= dt
	switch gu.fun {
	case 11, 12:
		g.updateAccident(gu, dt)
	case 13:
		g.updateCheerer(gu, dt)
	case 14:
		g.updateBitten(gu, dt)
	case 0:
		if int(gu.timer*2)%2 == 0 && g.rng.Float64() < 0.1 {
			v := dirVec(gu.dir)
			g.burst(Vec{gu.pos.X + v.X*10, gu.pos.Y - 28}, 5, 40, rgb(0xffffff), 0.15, 2, 0)
		}
	case 3:
		if gu.timer < 2.4 && gu.timer > 2.3 {
			gu.icon, gu.iconT = "¡PUM!", 1.2
			g.snd.Play("thud")
		}
		if gu.timer <= 0 {
			gu.icon, gu.iconT = "...", 1.5
		}
	}
	if gu.timer <= 0 {
		gu.state, gu.timer = gIdle, 2
	}
}

func (g *Game) updateStreaker() {
	if g.streaked || g.time < g.streakT || g.testVictim != nil || g.mode != ModePlay || g.guestRoom() < 1 {
		return
	}
	g.streaked = true
	spec := genericSpec(g.rng)
	spec.Top, spec.Bottom, spec.Pants, spec.Hat, spec.Glasses = TopBare, BotTrunks, hx(0xf0f0f0), HatNone, GlassNone
	spec.Shirt, spec.Shirt2 = spec.Skin, spec.Skin
	spec.Female = false
	cs := newCharSet(&spec)
	pos := g.lvl.NearestFree(g.lvl.Def.Start)
	gu := &Guest{pos: pos, dir: DirUp, cs: cs, style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gRun, timer: 3, role: 0}
	gu.runUntil = g.time + 16
	gu.icon, gu.iconT = "¡AAAAH!", 3
	g.guests = append(g.guests, gu)
	g.total++
	g.say("¡Un espontáneo cruza la fiesta en calzoncillos!")
	g.snd.Play("alert")
	for _, o := range g.guests {
		if !o.out() && o != gu && o.perf == "" && g.canWatch(o) && dist(o.pos, gu.pos) < 260 && g.rng.Float64() < 0.5 {
			o.icon, o.iconT = []string{"JAJAJA", "¡ESE ES!", "OLE!", "¡EH!"}[g.rng.Intn(4)], 2.5
		}
	}
}
