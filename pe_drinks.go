package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "drinks": a drinking contest. Two guests face each other at a little table in front of the bar and empty
// cups as fast as they can (the number of cups shows over their heads); three more guests egg them on. The first to
// get to six wins and shows off with a crown; the other one is dizzy: stars around his head and a drunken S-shaped
// sway. No points, nobody gets poisoned or sick (the guests' own drink counters are not touched).
//
// WHEN: parties with a bar (bar or minibar) and enough people, 20 % of them, between 60 and 220 s, up to twice.
// Not Big (two people and a table) but it takes the group "bar" so two bar events never overlap.
// ---------------------------------------------------------------------------

const (
	drGoal    = 6
	drGatherT = 24.0
	drResultT = 6.5
)

const (
	drGather = iota
	drCompete
	drResult
)

var drinksLines = pevTexts("¡Concurso de tragos en la barra!", "¡Bebe, bebe, bebe!", "¡Tú puedes!", "¡Dale, dale, dale!", "¡Otro, otro!",
	"¡Soy el rey de la barra!", "Uuuy... el suelo se mueve", "¡Hic!", "¡Camarero, otro!", "¡Increíble!", "Yo ya veo doble...")

type drinksState struct {
	bar   *Prop
	mid   Vec
	c     [2]*pevMem
	spot  [2]Vec
	crowd pevCrowd
	phase int
	t0    float64
	iv    [2]float64
	cnt   [2]int
	win   int
	said  float64

	cheered bool
}

func (d *drinksState) pevHas(gu *Guest) bool {
	return d.c[0] != nil && d.c[0].gu == gu || d.c[1] != nil && d.c[1].gu == gu || d.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "drinks", Name: "Concurso de tragos", Group: "bar",
		Prob: 0.20, AtMin: 60, AtMax: 220, Dur: 26, Cooldown: 160, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool { return t.bar && t.roam && t.guests >= 16 },
		Start:    drinksStart, Items: drinksItems,
		Update: pevGuard(drinksUpdate, func(g *Game, a *partyAct) { drinksEnd(g, a.D.(*drinksState)) })})
}

func drinksStart(g *Game, a *partyAct) bool {
	r := partyDice()
	// the table goes in front of the bar nearest to the middle of the guests that still has room
	var best *Prop
	var mid Vec
	bd := math.MaxFloat64
	for _, pr := range g.lvl.Props {
		if pr.gone || pr.K == nil || pr.Name != "bar" && pr.Name != "minibar" {
			continue
		}
		for _, dx := range []float64{0, 36, -36, 72, -72} {
			for _, dy := range []float64{16, 28, 40} {
				p := Vec{pr.Foot.X + pr.Foot.W/2 + dx, pr.Foot.Y + pr.Foot.H + dy}
				if !g.pevClear(p, 17) || !g.pevPlayerFar(p, 60) {
					continue
				}
				if d := math.Abs(dx) + dy; d < bd {
					bd, best, mid = d, pr, p
				}
			}
		}
	}
	if best == nil {
		return false
	}
	st := &drinksState{bar: best, mid: mid, t0: g.time}
	st.spot = [2]Vec{{mid.X - 13, mid.Y}, {mid.X + 13, mid.Y}}
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && dist(gu.pos, mid) < 200 && g.lvl.PathG(gu.pos, mid) != nil {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 4 {
		return false
	}
	// the two nearest to the table compete
	for k := 0; k < 2; {
		if len(cand) == 0 {
			for _, m := range st.c {
				if m != nil {
					pevRelease(m, r)
				}
			}
			return false
		}
		bi := 0
		for i, gu := range cand {
			if dist(gu.pos, mid) < dist(cand[bi].pos, mid) {
				bi = i
			}
		}
		m := &pevMem{gu: cand[bi]}
		cand = append(cand[:bi], cand[bi+1:]...)
		if g.pevSendTo(m, st.spot[k]) {
			st.c[k] = m
			k++
		}
	}
	// speeds: always one clearly faster than the other
	st.iv[0] = 0.78 + r.Float64()*0.32
	st.iv[1] = 0.78 + r.Float64()*0.32
	if math.Abs(st.iv[0]-st.iv[1]) < 0.12 {
		st.iv[1] += 0.14
	}
	if st.iv[0] < st.iv[1] {
		st.win = 0
	} else {
		st.win = 1
	}
	st.crowd.gather(g, r, mid, 3, 30, 42, func(gu *Guest) bool { return st.pevHas(gu) })
	g.say(drinksLines[0])
	a.D = st
	return true
}

func drinksEnd(g *Game, st *drinksState) {
	r := partyDice()
	for _, m := range st.c {
		if m != nil {
			pevRelease(m, r)
		}
	}
	st.crowd.release(g, r)
}

func drinksUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*drinksState)
	r := partyDice()
	age := g.time - st.t0
	for _, m := range st.c {
		if !m.mine() {
			drinksEnd(g, st) // one of them was frightened away: the contest is over
			return true
		}
	}
	st.crowd.update(g, r, nil, 0, nil)
	for _, m := range st.c {
		m.keep()
	}
	switch st.phase {
	case drGather:
		both := true
		for k, m := range st.c {
			gu := m.gu
			if gu.state == gIdle && gu.timer > 1000 {
				m.want = gIdle
				gu.dir = DirRight
				if k == 1 {
					gu.dir = DirLeft
				}
			} else {
				both = false
			}
		}
		if both {
			st.phase, st.t0 = drCompete, g.time
			for _, m := range st.c {
				m.pt = m.gu.pos
				m.hold(gDrink)
			}
		} else if age > drGatherT {
			drinksEnd(g, st)
			return true
		}
	case drCompete:
		for k, m := range st.c {
			gu := m.gu
			m.want = gDrink
			gu.state, gu.dur = gDrink, 9999
			ph := math.Mod(age/st.iv[k], 1)
			gu.timer = 9999 - (ph*6+0.02)/2.2
			st.cnt[k] = int(age / st.iv[k])
		}
		if g.time-st.said > 2.2 {
			st.said = g.time
			if len(st.crowd.mem) > 0 {
				c := st.crowd.mem[r.Intn(len(st.crowd.mem))].gu
				if c.icon == "" {
					c.icon, c.iconT = pevPick(r, drinksLines[1:5]), 1.6
				}
			}
		}
		if st.cnt[st.win] >= drGoal {
			st.cnt[st.win] = drGoal
			if st.cnt[1-st.win] >= drGoal {
				st.cnt[1-st.win] = drGoal - 1
			}
			st.phase, st.t0 = drResult, g.time
			w, l := st.c[st.win], st.c[1-st.win]
			w.hold(gFun)
			w.gu.fun, w.gu.dir = 7, DirDown
			w.gu.icon, w.gu.iconT = drinksLines[5], 4
			l.pt = l.gu.pos
			l.hold(gFun)
			l.gu.fun, l.gu.dir = 6, DirDown
			l.gu.icon, l.gu.iconT = pevPick(r, []string{drinksLines[6], drinksLines[10]}), 3.5
			g.snd.Play("win")
		}
	case drResult:
		for k, m := range st.c {
			m.keep()
			if k == 1-st.win { // the drunken S: he sways left and right
				m.gu.pos = Vec{m.pt.X + math.Sin(age*2.6)*4, m.pt.Y + math.Sin(age*1.3)*1.2}
				if g.lvl.SolidP(m.gu.pos.X, m.gu.pos.Y) {
					m.gu.pos = m.pt
				}
			}
		}
		if age > 3 && !st.cheered {
			st.cheered = true
			for _, m := range st.crowd.mem {
				m.gu.icon, m.gu.iconT = pevPick(r, drinksLines[8:10]), 1.8
			}
		}
		if age >= drResultT {
			drinksEnd(g, st)
			return true
		}
	}
	return false
}

// drawTablePix: a little round table with two cups, (cx, cy) its feet.
func drawTablePix(plot plotFn, cx, cy, px int, cups int) {
	dark, top, lite := color.RGBA{0x4a, 0x2e, 0x14, 255}, color.RGBA{0xb0, 0x7a, 0x3c, 255}, color.RGBA{0xd0, 0x98, 0x58, 255}
	plot(cx-7*px, cy-9*px, 15*px, 5*px, dark)
	plot(cx-6*px, cy-8*px, 13*px, 3*px, top)
	plot(cx-6*px, cy-8*px, 13*px, px, lite)
	plot(cx-px, cy-4*px, 3*px, 4*px, dark)
	plot(cx-3*px, cy-px, 7*px, px, dark)
	for i := 0; i < cups && i < 2; i++ { // the empty cups on the table
		x := cx - 5*px + i*8*px
		plot(x, cy-12*px, 3*px, 4*px, pevWhite)
		plot(x, cy-12*px, 3*px, px, glassCol)
	}
}

// drawCountPix: the cups counter over a head: a little cup and the digit.
func drawCountPix(plot plotFn, cx, top, px, n int) {
	plot(cx-6*px, top+px, 3*px, 4*px, pevWhite)
	plot(cx-6*px, top+px, 3*px, px, wineCol)
	plot(cx-5*px, top+5*px, px, px, pevWhite)
	drawDigitPix(plot, cx, top, px, n, clapCol)
}

func drinksItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*drinksState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	items = append(items, drawItem{st.mid.Y - 0.1, func() {
		cx, cy := g.pevXY(st.mid)
		drawTablePix(plot, cx, cy+k(3), px, 2)
	}})
	for i, m := range st.c {
		i, m := i, m
		if !m.mine() {
			continue
		}
		gu := m.gu
		items = append(items, drawItem{gu.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(gu.pos)
			switch st.phase {
			case drCompete:
				drawCountPix(plot, cx-px, cy-k(50), px, st.cnt[i])
			case drResult:
				if i == st.win {
					bob := int(math.Floor(math.Mod(g.time*5, 2)))
					drawCrownPix(plot, cx, cy-k(47)-bob)
				} else {
					drawStarsPix(plot, cx, cy-k(41), px, g.time)
				}
			}
		}})
	}
	for _, m := range st.crowd.mem {
		gu := m.gu
		if st.phase == drResult && gu.state == gIdle && gu.timer > 1000 {
			items = append(items, drawItem{gu.pos.Y + 0.3, func() {
				cx, cy := g.pevXY(gu.pos)
				drawClapPix(plot, cx, cy-k(34), px, g.time, int(math.Abs(gu.pos.X)))
			}})
		}
	}
	return items
}
