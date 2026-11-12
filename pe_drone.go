package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "drone" (DRON CON LUCES): now and then a small drone with blinking coloured lights flies over the party: it
// comes in from one side, makes a slow loop over the middle and leaves by the other side. Its shadow slides on the
// floor. Some guests look up and point at it with a short line. Rule of the owner: "nada de derribar, sea solo para
// verlo": it is pure ambience. It is NOT an object, a guest or an animal: it does not collide, it cannot be used,
// attacked or shot down, and it is drawn before the HUD (partyOver), so it never covers the panel or the notices.
// A soft buzz when it starts (synthesised, "buzz", not heard by a person).
//
// WHEN: open-air parties (not the station, not the island), 18 % of them, between 40 and 260 s, once
// per party. Group "sky".
// ---------------------------------------------------------------------------

const (
	droneDur    = 26.0 // seconds over the party
	droneAlt    = 52.0 // height (world px) over its shadow
	droneLookT  = 2.6  // a guest points at it this long
	droneLookGp = 3.2  // seconds between two guests pointing
)

var droneLines = pevTexts("¡Mira, un dron!", "¿Nos está grabando?", "¡Saluda, que salimos en la tele!", "¡Qué lucecitas!",
	"¿Eso es un ovni?", "¡Que no se caiga encima!")

type droneState struct {
	from, to Vec // the ground track (its shadow) enters at from and leaves at to
	loopC    Vec
	t0       float64
	look     []*pevMem
	lookAt   []float64
	nextLook float64
	said     int
}

func (s *droneState) pevHas(gu *Guest) bool { return pevHasIn(s.look, gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "drone", Name: "Dron con luces", Group: "sky",
		Prob: 0.12, AtMin: 40, AtMax: 260, Dur: droneDur,
		Eligible: func(g *Game, t partyTraits) bool {
			return t.outdoor && t.icon != "station" && t.icon != "grave" && t.roam && t.guests >= 10
		},
		Start: droneStart, Items: droneItems, Over: droneOver,
		Update: pevGuard(droneUpdate, func(g *Game, a *partyAct) { droneEnd(g, a.D.(*droneState)) })})
}

func droneStart(g *Game, a *partyAct) bool {
	r := partyDice()
	l := g.lvl
	W, H := l.Def.W, l.Def.H
	if W < 200 || H < 160 {
		return false
	}
	y0 := H*0.3 + r.Float64()*H*0.4
	y1 := H*0.3 + r.Float64()*H*0.4
	from, to := Vec{-30, y0}, Vec{W + 30, y1}
	if r.Float64() < 0.5 {
		from, to = to, from
	}
	c := Vec{W*0.35 + r.Float64()*W*0.3, H*0.35 + r.Float64()*H*0.3}
	if pevDebugOn("drone") {
		c = g.p.pos
	}
	a.D = &droneState{from: from, to: to, loopC: c, t0: g.time, nextLook: g.time + 3}
	g.treatSfx()
	g.snd.Play("buzz")
	if pevDebugOn("drone") {
		pevDebugLog(g, "drone", c)
	}
	return true
}

// droneAt: where its shadow is (world) u seconds after it came in: in, a loop of 8 s around the middle, out.
func (s *droneState) at(u float64) Vec {
	const in, loop = 9.0, 8.0
	r := 34.0
	entry := Vec{s.loopC.X, s.loopC.Y - r} // the loop starts at its top
	switch {
	case u < in:
		k := u / in
		return Vec{s.from.X + (entry.X-s.from.X)*k, s.from.Y + (entry.Y-s.from.Y)*k + math.Sin(k*math.Pi*2)*8}
	case u < in+loop:
		a := -math.Pi/2 + (u-in)/loop*2*math.Pi
		return Vec{s.loopC.X + math.Cos(a)*r*1.4, s.loopC.Y + math.Sin(a)*r}
	}
	k := math.Min(1, (u-in-loop)/(droneDur-in-loop))
	return Vec{entry.X + (s.to.X-entry.X)*k, entry.Y + (s.to.Y-entry.Y)*k + math.Sin(k*math.Pi*2)*8}
}

func droneEnd(g *Game, s *droneState) {
	pevReleaseAll(s.look, partyDice())
	s.look, s.lookAt = nil, nil
}

func droneUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*droneState)
	r := partyDice()
	u := g.time - s.t0
	sh := s.at(u)
	// the ones pointing: held a moment, facing it, then let go
	keep, keepT := s.look[:0], s.lookAt[:0]
	for i, m := range s.look {
		if !m.mine() || g.time-s.lookAt[i] > droneLookT {
			pevLetGo(m)
			continue
		}
		m.keep()
		m.gu.dir = faceTo(m.gu.pos, sh)
		keep, keepT = append(keep, m), append(keepT, s.lookAt[i])
	}
	s.look, s.lookAt = keep, keepT
	if g.time >= s.nextLook && len(s.look) < 2 && u < droneDur-3 {
		s.nextLook = g.time + droneLookGp*(0.7+0.6*r.Float64())
		var best *Guest
		for _, gu := range g.guests {
			if g.pevFreeGuest(gu) && dist(gu.pos, sh) < 130 && (best == nil || dist(gu.pos, sh) < dist(best.pos, sh)) {
				best = gu
			}
		}
		if best != nil {
			m := &pevMem{gu: best}
			m.pose(AnimPoint)
			best.dir = faceTo(best.pos, sh)
			if best.icon == "" {
				best.icon, best.iconT = droneLines[s.said%len(droneLines)], 2.2
				s.said++
			}
			s.look, s.lookAt = append(s.look, m), append(s.lookAt, g.time)
		}
	}
	if u >= droneDur {
		droneEnd(g, s)
		return true
	}
	return false
}

// droneItems: the shadow on the floor (sorted with everything on the ground).
func droneItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*droneState)
	sh := s.at(g.time - s.t0)
	return append(items, drawItem{-1e6, func() { // on the floor, under everything that stands
		x, y := g.pevXY(sh)
		k := g.pevK()
		fillRect(dst, float64(x-k(7)), float64(y-k(1)), float64(k(14)), float64(maxI(1, k(2))), color.RGBA{0, 0, 0, 60})
		fillRect(dst, float64(x-k(4)), float64(y-k(2)), float64(k(8)), float64(maxI(1, k(4))), color.RGBA{0, 0, 0, 50})
	}})
}

// drawDronePix: the drone (body 8x3, four arms with spinning rotors and blinking lights) centred on (x, y).
func drawDronePix(plot plotFn, x, y, px int, t float64) {
	body := color.RGBA{0x2a, 0x2c, 0x36, 255}
	plot(x-4*px, y-px, 8*px, 3*px, body)
	plot(x-2*px, y-2*px, 4*px, px, color.RGBA{0x4a, 0x4e, 0x5a, 255})
	plot(x-px, y+2*px, 2*px, px, color.RGBA{0x10, 0x10, 0x14, 255}) // the camera
	for _, sx := range []int{-1, 1} {
		plot(x+sx*4*px-boolI(sx < 0)*3*px, y-px, 3*px, px, body) // arms
		rx := x + sx*7*px
		if int(t*20)%2 == 0 {
			plot(rx-3*px, y-2*px, 6*px, px, color.RGBA{0xb8, 0xbc, 0xc8, 255})
		} else {
			plot(rx-px, y-2*px, 2*px, px, color.RGBA{0xb8, 0xbc, 0xc8, 255})
		}
	}
	lights := []color.RGBA{{0xff, 0x30, 0x40, 255}, {0x30, 0xff, 0x70, 255}, {0x40, 0x9a, 0xff, 255}, {0xff, 0xf0, 0x80, 255}}
	ph := int(t * 4)
	for i, lx := range []int{-6, -2, 2, 6} {
		if (ph+i)%3 != 0 {
			plot(x+lx*px, y+px, px, px, lights[(ph+i)%len(lights)])
		}
	}
}

// droneOver: the drone itself, over everything of the world (after the darkness of a blackout: its lights shine) and
// before the HUD, which is always drawn on top.
func droneOver(g *Game, dst *ebiten.Image, a *partyAct) {
	s := a.D.(*droneState)
	u := g.time - s.t0
	sh := s.at(u)
	bob := math.Round(math.Sin(g.time*3) * 2)
	x, y := g.pevXY(Vec{sh.X, sh.Y - droneAlt*g.lifeScale() + bob})
	plot := g.plotter(dst)
	px := g.pevPx()
	ph := int(g.time * 4)
	glow := []color.RGBA{{255, 60, 80, 40}, {60, 255, 120, 40}, {80, 150, 255, 40}}[ph%3]
	g.glowAt(dst, Vec{sh.X, sh.Y - droneAlt*g.lifeScale() + bob + 4}, 7, glow) // a soft halo under it, the drone stays sharp on top
	drawDronePix(plot, x, y, px, g.time)
}
