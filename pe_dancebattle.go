package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "batalla" (idea 253): a DANCE BATTLE. Two ordinary guests challenge each other on the dance floor (or on an open
// spot of the party when there is none); a ring of 5-7 guests gathers around them, claps and shouts. The two take turns
// (each turn ~2.4 s) with different dance animations for 10-20 s, the other one waiting and taunting; then one of them
// wins (cheers) and the other one sulks, and the ring breaks up.
//
// The ring is a fan of points in front of and beside the dancers (pevCrowd.gather), so its back stays OPEN: the player is
// never closed in and the guests never block each other. If anything goes wrong it breaks up and RUNS: one of the two is
// killed or frightened, a fresh body falls near, or the police are called while it goes on (the alarm). Every phase has a
// time limit (walking 14 s, the frame's pevGuard at 2*Dur), so nobody is left stuck.
//
// WHEN: a party event of the "dance" group (never together with the conga, the chairs or the dance contest), Big, chance
// 18 % in the parties with a dance floor or room to roam and 15+ guests, at most once a party (the longest cooldown).
// Never in STEALTH, the RETO DE HOY or the island.
// ---------------------------------------------------------------------------

var (
	battleDare  = pevTexts("¡Te reto a un duelo de baile!", "¿Bailas o qué?", "¡Esta pista es mía!", "¡A ver qué sabes hacer!")
	battleTaunt = pevTexts("¿Eso es todo?", "¡Mira y aprende!", "¡Toma ya!", "¡Supera esto!", "Mi abuela baila mejor")
	battleCrowd = pevTexts("¡Uooo!", "¡Baila, baila!", "¡Duelo!", "¡Qué nivel!", "¡Otra, otra!")
	battleWin   = pevTexts("¡Soy el rey de la pista!", "¡Gané!", "¡Campeón!")
	battleLose  = pevTexts("Quiero la revancha", "Hoy no era mi día", "Me ha dado flato")
)

const (
	battleChance  = 0.12
	battleGap     = 18.0 // between the two dancers
	battleTurn    = 2.4
	battleMinLen  = 10.0
	battleMaxLen  = 20.0
	battleWalkMax = 14.0
)

const (
	btWalk = iota
	btDare
	btDance
	btEnd
	btOver
)

type battleState struct {
	a, b    *pevMem
	crowd   pevCrowd
	phase   int
	t0      float64
	center  Vec
	length  float64
	turn    int     // 0 = a dances, 1 = b dances
	nextT   float64 // when the turn changes
	stepA   int     // the dance animation of each one (two different ones)
	stepB   int
	winner  int
	bodies  int // bodies on the ground when it started (a new one near = alarm)
	police  bool
	fled    bool
	cheerAt float64
}

func (s *battleState) pevHas(gu *Guest) bool {
	return s.a != nil && s.a.gu == gu || s.b != nil && s.b.gu == gu || s.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "batalla", Name: "Batalla de baile", Group: "dance", Big: true,
		Prob: battleChance, AtMin: 70, AtMax: 260, Dur: 30, MaxRuns: 1, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return (t.dance || t.roam) && t.guests >= 15 },
		Start:    battleStart, Items: battleItems,
		Update: pevGuard(battleUpdate, func(g *Game, a *partyAct) { battleEnd(g, a.D.(*battleState)) })})
}

// battleSpot: the middle of a dance floor with room around it, else an open spot of the party.
func (g *Game) battleSpot(r *rand.Rand) (Vec, bool) {
	for _, i := range r.Perm(len(g.lvl.Def.Dance)) {
		d := g.lvl.Def.Dance[i]
		for k := 0; k < 8; k++ {
			p := Vec{d.X + d.W*(0.3+0.4*r.Float64()), d.Y + d.H*(0.3+0.4*r.Float64())}
			if g.pevClear(p, 34) && g.pevPlayerFar(p, 60) {
				return p, true
			}
		}
	}
	return g.pevSpot(r, 34, false)
}

func battleStart(g *Game, a *partyAct) bool {
	r := partyDice()
	c, ok := g.battleSpot(r)
	if !ok {
		return false
	}
	var cand []*Guest
	for _, gu := range g.guests {
		if peleaCanBe(g, gu) && dist(gu.pos, c) < 260 && !fursuitSlow(gu) {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 9 {
		return false
	}
	// the two nearest
	for i := 1; i < len(cand); i++ {
		for j := i; j > 0 && dist(cand[j].pos, c) < dist(cand[j-1].pos, c); j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	pa, pb := Vec{c.X - battleGap/2, c.Y}, Vec{c.X + battleGap/2, c.Y}
	s := &battleState{a: &pevMem{gu: cand[0], pt: pa}, b: &pevMem{gu: cand[1], pt: pb}, center: c, t0: g.time,
		length: battleMinLen + r.Float64()*(battleMaxLen-battleMinLen), police: g.policeCalled, bodies: len(g.bodies)}
	if !g.pevSendTo(s.a, pa) || !g.pevSendTo(s.b, pb) {
		pevReleaseAll([]*pevMem{s.a, s.b}, r)
		return false
	}
	steps := r.Perm(12)
	s.stepA, s.stepB = AnimDance0+steps[0], AnimDance0+steps[1]
	s.winner = r.Intn(2)
	a.D = s
	return true
}

func (s *battleState) both() bool { return s.a.mine() && s.b.mine() }

// alarm: something has frightened the party around the battle.
func (g *Game) battleAlarm(s *battleState) bool {
	if g.policeCalled && !s.police {
		return true
	}
	for i := s.bodies; i < len(g.bodies); i++ {
		if dist(g.bodies[i].pos, s.center) < 160 {
			return true
		}
	}
	return false
}

func battleUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*battleState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase < btEnd && (!s.both() || g.battleAlarm(s)) { // someone killed or frightened one of them, or there is an alarm
		battleFlee(g, s, r)
		return true
	}
	if s.phase >= btDare && s.phase < btOver {
		s.crowd.update(g, r, nil, 0, nil)
		for _, m := range s.crowd.mem {
			if m.gu.icon == "" && r.Float64() < dt*0.25 {
				m.gu.icon, m.gu.iconT = pevPick(r, battleCrowd), 1.4
			}
		}
	}
	switch s.phase {
	case btWalk:
		if peleaArrived(s.a) && peleaArrived(s.b) {
			s.phase, s.t0 = btDare, g.time
			s.a.want, s.b.want = gIdle, gIdle
			s.a.gu.dir, s.b.gu.dir = faceTo(s.a.gu.pos, s.b.gu.pos), faceTo(s.b.gu.pos, s.a.gu.pos)
			s.a.gu.icon, s.a.gu.iconT = pevPick(r, battleDare), 2
			s.crowd.gather(g, r, s.center, 5+r.Intn(3), 34, 50, func(gu *Guest) bool { return gu == s.a.gu || gu == s.b.gu })
		} else if age > battleWalkMax {
			battleEnd(g, s)
			return true
		}
	case btDare:
		s.a.keep()
		s.b.keep()
		if age > 2.2 {
			s.phase, s.t0, s.turn, s.nextT = btDance, g.time, 0, g.time+battleTurn
			s.a.pose(s.stepA)
			s.b.pose(AnimIdle)
		}
	case btDance:
		if g.time >= s.nextT {
			s.turn = 1 - s.turn
			s.nextT = g.time + battleTurn
			on, off, step := s.a, s.b, s.stepA
			if s.turn == 1 {
				on, off, step = s.b, s.a, s.stepB
			}
			on.pose(step)
			off.pose(AnimIdle)
			if r.Float64() < 0.6 && on.gu.icon == "" {
				on.gu.icon, on.gu.iconT = pevPick(r, battleTaunt), 1.5
			}
		}
		s.a.keep()
		s.b.keep()
		dancer, other := s.a.gu, s.b.gu
		if s.turn == 1 {
			dancer, other = s.b.gu, s.a.gu
		}
		dancer.dir = DirDown
		other.dir = faceTo(other.pos, dancer.pos)
		if age > s.length {
			s.phase, s.t0 = btEnd, g.time
			win, lose := s.a, s.b
			if s.winner == 1 {
				win, lose = s.b, s.a
			}
			win.pose(AnimCheer)
			lose.pose(AnimTired)
			win.gu.icon, win.gu.iconT = pevPick(r, battleWin), 2.2
			lose.gu.icon, lose.gu.iconT = pevPick(r, battleLose), 2.2
			s.cheerAt = g.time
		}
	case btEnd:
		if s.a.mine() {
			s.a.keep()
		}
		if s.b.mine() {
			s.b.keep()
		}
		if age > 2.8 {
			battleEnd(g, s)
			return true
		}
	case btOver:
		return true
	}
	return false
}

// battleFlee: the ring and the dancers run off.
func battleFlee(g *Game, s *battleState, r *rand.Rand) {
	s.fled = true
	for _, m := range append([]*pevMem{s.a, s.b}, s.crowd.mem...) {
		if m == nil {
			continue
		}
		if m.mine() {
			pevScare(g, m.gu, r)
		} else {
			pevForget(m)
		}
	}
	s.crowd.mem = nil
	s.phase = btOver
	battleFled++
}

// battleFled: battles broken up by an alarm (tests).
var battleFled int

func battleEnd(g *Game, s *battleState) {
	if s.phase == btOver {
		return
	}
	r := partyDice()
	for _, m := range []*pevMem{s.a, s.b} {
		if m == nil {
			continue
		}
		if m.mine() {
			pevRelease(m, r)
		} else {
			pevForget(m)
		}
	}
	s.crowd.release(g, r)
	s.phase = btOver
}

// battleItems: the ring claps (hands over the heads) while they dance, and notes fly over the one who is dancing.
func battleItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*battleState)
	if s.phase != btDance && s.phase != btEnd {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, m := range s.crowd.mem {
		gu := m.gu
		if gu.state != gIdle || gu.timer < 1000 {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(gu.pos)
			drawClapPix(plot, cx, cy-k(34), px, g.time, int(math.Abs(gu.pos.X)))
		}})
	}
	if s.phase == btDance {
		d := s.a.gu
		if s.turn == 1 {
			d = s.b.gu
		}
		items = append(items, drawItem{d.pos.Y + 0.4, func() {
			cx, cy := g.pevXY(d.pos)
			drawNotesPix(plot, cx, cy-k(40), px, g.time, 3)
		}})
	}
	return items
}
