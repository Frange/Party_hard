package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "choir" (CORO ESPONTÁNEO): three to six guests get together in a gentle arc facing the camera (the
// "audience"), put their arms over each other's shoulders and sing swaying from side to side, with notes floating
// over them, for 10-20 s; then they break up and go back to the party. If one of them is killed, the rest jump
// (a panic pose) with a funny line and scatter.
// No points, no blame. Few groups and a long cooldown; the guests who do not arrive in time are left out and the
// whole thing has a hard time limit (the frame's 2*Dur), so nobody is ever stuck.
//
// WHEN: any party with room and people (not the station), 18 % of them, between 45 and 260 s, once
// per party. Big (only one big group thing at a time), group "sing" (never with the karaoke-like ones).
// ---------------------------------------------------------------------------

const (
	choirWalkMax = 14.0
	choirSlotDX  = 11.0
	choirReactT  = 1.6
)

var choirLines = pevTexts("¡Todos juntos!", "¡La la laaa!", "¡Oh, oh, oooh!", "¡Otra vez el estribillo!", "¡Qué bonito!",
	"¡Se acabó el concierto!", "¡Que alguien llame a un médico!", "¡Desafinaba, pero tanto no!", "¡Corred!", "¡Nunca más canto!")

const (
	chWalk = iota
	chSing
	chReact
)

type choirState struct {
	mem    []*pevMem
	center Vec
	phase  int
	t0     float64
	singT  float64
	sung   int
}

func (s *choirState) pevHas(gu *Guest) bool { return pevHasIn(s.mem, gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "choir", Name: "Coro espontáneo", Group: "sing", Big: true,
		Prob: 0.12, AtMin: 45, AtMax: 260, Dur: 34,
		Eligible: func(g *Game, t partyTraits) bool { return t.icon != "station" && t.roam && t.guests >= 20 },
		Start:    choirStart, Items: choirItems,
		Update: pevGuard(choirUpdate, func(g *Game, a *partyAct) { choirEnd(g, a.D.(*choirState)) })})
}

// choirSlot: the place of singer i of n (a gentle arc open towards the camera).
func choirSlot(c Vec, i, n int) Vec {
	off := float64(i) - float64(n-1)/2
	return Vec{c.X + off*choirSlotDX, c.Y + math.Round(off*off*1.1)}
}

func choirStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var c Vec
	ok := false
	if pevDebugOn("choir") {
		c, ok = g.pevNearPlayerSpot(34, false)
	} else {
		c, ok = g.pevSpot(r, 34, false)
	}
	if !ok {
		return false
	}
	n := 3 + r.Intn(4)
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && dist(gu.pos, c) < 260 {
			cand = append(cand, gu)
		}
	}
	for i := 1; i < len(cand); i++ { // the nearest ones join
		for j := i; j > 0 && dist(cand[j].pos, c) < dist(cand[j-1].pos, c); j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	s := &choirState{center: c, t0: g.time, singT: 10 + r.Float64()*10}
	used := 0
	for i := 0; i < n && used < len(cand); i++ {
		want := choirSlot(c, i, n)
		pt := g.lvl.NearestFree(want)
		if dist(pt, want) > 3 {
			continue
		}
		for used < len(cand) {
			gu := cand[used]
			used++
			m := &pevMem{gu: gu, pt: pt, idx: i}
			if g.pevSendTo(m, pt) {
				s.mem = append(s.mem, m)
				break
			}
		}
	}
	if len(s.mem) < 3 {
		pevReleaseAll(s.mem, r)
		return false
	}
	s.mem[0].gu.icon, s.mem[0].gu.iconT = choirLines[0], 2.2
	a.D = s
	if pevDebugOn("choir") {
		pevDebugLog(g, "choir", c)
	}
	return true
}

func choirEnd(g *Game, s *choirState) {
	pevReleaseAll(s.mem, partyDice())
	s.mem = nil
}

// choirScatter: after a death they run off to far places of the party (no holding: they decide again when they get there).
func choirScatter(g *Game, s *choirState) {
	r := partyDice()
	for _, m := range s.mem {
		gu := m.gu
		if gu.out() || !m.mine() {
			pevForget(m)
			continue
		}
		var path []Vec
		for i := 0; i < 12 && path == nil; i++ {
			t := g.lvl.RandomRoam(r)
			if dist(t, gu.pos) > 110 {
				path = g.lvl.PathG(gu.pos, t)
			}
		}
		if path == nil {
			pevRelease(m, r)
			continue
		}
		gu.state, gu.path, gu.next, gu.dur, gu.timer, gu.moving = gGoto, path, gIdle, 1+r.Float64()*2, 0, false
	}
	s.mem = nil
}

func choirUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*choirState)
	r := partyDice()
	age := g.time - s.t0
	// a singer killed: the rest jump and run
	if s.phase != chReact {
		for _, m := range s.mem {
			if m.gu.dead {
				s.phase, s.t0 = chReact, g.time
				var keep []*pevMem
				for _, o := range s.mem {
					if !o.gu.out() && o.mine() {
						o.pose(AnimPanic)
						o.gu.icon, o.gu.iconT = pevPick(r, choirLines[5:]), 2
						keep = append(keep, o)
					} else {
						pevForget(o)
					}
				}
				s.mem = keep
				return len(s.mem) == 0
			}
		}
	}
	switch s.phase {
	case chWalk:
		s.mem = pevPruneMems(s.mem)
		here := 0
		for _, m := range s.mem {
			m.keep()
			if m.gu.state == gIdle && m.gu.timer > 1000 {
				m.want = gIdle
				m.gu.dir = DirDown
				here++
			}
		}
		if len(s.mem) < 3 && age > 2 {
			choirEnd(g, s)
			return true
		}
		if here == len(s.mem) || age > choirWalkMax {
			var keep []*pevMem
			for _, m := range s.mem { // the late ones are left out
				if m.gu.state == gIdle && m.gu.timer > 1000 {
					keep = append(keep, m)
				} else {
					pevLetGo(m)
				}
			}
			s.mem = keep
			if len(s.mem) < 3 {
				choirEnd(g, s)
				return true
			}
			for _, m := range s.mem {
				m.hold(gIdle)
				m.gu.dir = DirDown
			}
			s.phase, s.t0 = chSing, g.time
			g.snd.Play("hype")
		}
	case chSing:
		s.mem = pevPruneMems(s.mem)
		if len(s.mem) < 2 {
			choirEnd(g, s)
			return true
		}
		for _, m := range s.mem {
			m.keep()
			m.gu.dir = DirDown
		}
		if k := int(age / 4); k > s.sung && k <= 4 {
			s.sung = k
			m := s.mem[r.Intn(len(s.mem))]
			if m.gu.icon == "" {
				m.gu.icon, m.gu.iconT = pevPick(r, choirLines[1:4]), 2
			}
		}
		if age >= s.singT {
			for _, gu := range g.guests { // a bit of applause from somebody near
				if !gu.out() && gu.icon == "" && !pevHasIn(s.mem, gu) && dist(gu.pos, s.center) < 110 {
					gu.icon, gu.iconT = choirLines[4], 1.8
					break
				}
			}
			choirEnd(g, s)
			return true
		}
	case chReact:
		if age >= choirReactT {
			choirScatter(g, s)
			return true
		}
	}
	return false
}

// choirSway: the side offset (whole pixels) of the whole row while it sings.
func choirSway(t float64) float64 { return math.Round(math.Sin(t*2.4) * 2) }

// drawGuest: the singers swaying together (pe_fx.go asks).
func (s *choirState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if s.phase != chSing || gu.out() {
		return false
	}
	for _, m := range s.mem {
		if m.gu == gu && m.mine() {
			pos.X += choirSway(g.time)
			g.drawActor(dst, gu.cs, AnimSing, g.time*1.2+float64(m.idx)*0.1, pos, DirDown, 0, false)
			return true
		}
	}
	return false
}

func choirItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*choirState)
	if s.phase != chSing || len(s.mem) < 2 {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	maxY := s.center.Y
	for _, m := range s.mem {
		maxY = math.Max(maxY, m.gu.pos.Y)
	}
	mem := append([]*pevMem(nil), s.mem...)
	items = append(items, drawItem{maxY + 0.3, func() {
		sw := int(choirSway(g.time)) * px
		// the arms over the shoulders: from each one's shoulder to the next one's
		for i := 0; i+1 < len(mem); i++ {
			a, b := mem[i].gu, mem[i+1].gu
			if a.out() || b.out() || dist(a.pos, b.pos) > choirSlotDX*1.8 {
				continue
			}
			ax, ay := g.pevXY(a.pos)
			bx, by := g.pevXY(b.pos)
			drawLinePix(plot, ax+k(3)+sw, ay-k(25), bx-k(2)+sw, by-k(26), px, choirSkin(a))
			drawLinePix(plot, bx-k(3)+sw, by-k(24), ax+k(2)+sw, ay-k(25), px, choirSkin(b))
		}
		cx, cy := g.pevXY(s.center)
		drawNotesPix(plot, cx, cy-k(46), px, g.time, minI(len(mem), 4))
	}})
	return items
}

// choirSkin: the colour of a singer's arm (characters drawn from asset files carry no spec: a plain skin tone).
func choirSkin(gu *Guest) color.RGBA {
	if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Skin.A != 0 {
		return gu.cs.Spec.Skin
	}
	return hx(0xe8b890)
}
