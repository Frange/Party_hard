package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "parejas" (idea 316): a SLOW DANCE. Somebody calls for a slow one and 1 to 3 couples of ordinary guests walk
// onto the dance floor and dance cheek to cheek: face to face, arms round each other, turning slowly round their own
// middle, with little hearts over them; a few guests stand round the floor and watch.
//
// If the player kills one of a couple, the other one stays a moment hugging nobody (the same pose, facing the empty
// spot), then cries and runs off. The other couples go on. If the partner SAW the kill, the game's own rules come first
// (he raises the alarm or calls the police): the event never overrides a witness. Every phase has a time limit
// (walking 14 s, the frame's pevGuard at 2*Dur).
//
// WHEN: a party event of the "dance" group (never with the conga, the pogo, the dance battle or the contest), Big,
// chance 11 % only in the parties with a dance floor and a mood for a slow one (couplesIcons) and 20+ guests, once a
// party. The music itself is not changed (the mp3 player has no tempo): it is the call, the hearts and the dance.
// Capture: -pevent parejas (or parejas:kill).
// ---------------------------------------------------------------------------

// couplesIcons: house party, neon, club, 70s disco, cruise, Eiffel tower, roof, Italy, casino, ranch, pirate ship.
var couplesIcons = map[string]bool{"house": true, "neon": true, "disco": true, "disco70": true, "boat": true, "eiffel": true,
	"roof": true, "italy": true, "casino": true, "ranch": true, "pirate": true}

var (
	coupleCall  = pevTexts("¡Ahora una lenta!", "¡Las parejas a la pista!", "Esta va por los enamorados")
	coupleSweet = pevTexts("Qué bien bailas", "No me pises...", "Te quiero", "¿Bailamos otra?", "Mmm...")
	coupleWatch = pevTexts("¡Qué bonito!", "Ay, el amor...", "¡Que se besen!", "Yo no tengo pareja...")
	coupleAlone = pevTexts("¿Cariño?", "¿Dónde te has ido?")
	coupleCry   = pevTexts("¡NOOOO!", "Snif... ¡no me dejes!", "¡Mi amor!")
)

const (
	coupleChance  = 0.11
	coupleGap     = 4.0 // half the distance between the two of a couple
	coupleSpace   = 34.0
	coupleMinLen  = 18.0
	coupleMaxLen  = 30.0
	coupleWalkMax = 14.0
	coupleAloneT  = 1.6 // hugging nobody
	coupleCryT    = 1.8 // crying, then he runs
)

const (
	cpWalk = iota
	cpDance
	cpOver
)

type couplePair struct {
	a, b    *pevMem
	c       Vec
	th      float64
	spin    float64
	on      bool    // dancing
	widow   *pevMem // the one left alone (his partner was killed)
	widowT  float64 // when he was left
	cried   bool
	stopped bool // the couple is over (fled, released)
}

type couplesState struct {
	pairs   []*couplePair
	crowd   pevCrowd
	phase   int
	t0      float64
	center  Vec
	length  float64
	police  bool
	widows  int // couples broken by a kill (tests)
	alone   int // widows who hugged nobody (tests)
	nextSay float64
	demoK   bool
}

func (s *couplesState) pevHas(gu *Guest) bool {
	for _, p := range s.pairs {
		if p.a.gu == gu || p.b.gu == gu {
			return true
		}
	}
	return s.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "parejas", Name: "Baile lento en pareja", Group: "dance", Big: true,
		Prob: coupleChance, AtMin: 90, AtMax: 300, Dur: 30, MaxRuns: 1, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return couplesIcons[t.icon] && t.dance && t.guests >= 20 },
		Start:    couplesStart, Items: couplesItems,
		Update: pevGuard(couplesUpdate, func(g *Game, a *partyAct) { couplesEnd(g, a.D.(*couplesState)) })})
}

func couplesStart(g *Game, a *partyAct) bool {
	r := partyDice()
	if len(g.lvl.Def.Dance) == 0 {
		return false
	}
	fl := g.lvl.Def.Dance[0]
	for _, f := range g.lvl.Def.Dance {
		if f.W*f.H > fl.W*fl.H {
			fl = f
		}
	}
	center := Vec{fl.X + fl.W/2, fl.Y + fl.H/2}
	cand := g.danceCands(center, 280)
	np := 1 + r.Intn(3)
	np = minI(np, (len(cand)-1)/2)
	if np < 1 {
		return false
	}
	// the places of the couples: on the floor, apart from each other
	var spots []Vec
	for k := 0; k < 60 && len(spots) < np; k++ {
		p := Vec{fl.X + fl.W*(0.15+0.7*r.Float64()), fl.Y + fl.H*(0.15+0.7*r.Float64())}
		if k > 40 { // a small floor: anywhere near it
			p = Vec{center.X + (r.Float64()-0.5)*140, center.Y + (r.Float64()-0.5)*90}
		}
		if !g.pevClear(p, 10) || !g.pevPlayerFar(p, 40) {
			continue
		}
		ok := true
		for _, q := range spots {
			if dist(p, q) < coupleSpace {
				ok = false
			}
		}
		if ok {
			spots = append(spots, p)
		}
	}
	if len(spots) == 0 {
		return false
	}
	s := &couplesState{center: center, t0: g.time, police: g.policeCalled,
		length: coupleMinLen + r.Float64()*(coupleMaxLen-coupleMinLen)}
	used := map[*Guest]bool{}
	for _, c := range spots {
		var two []*Guest
		for _, gu := range cand { // the next two free guests nearest to the floor
			if !used[gu] && len(two) < 2 {
				two = append(two, gu)
			}
		}
		if len(two) < 2 {
			break
		}
		p := &couplePair{a: &pevMem{gu: two[0], pt: Vec{c.X - coupleGap, c.Y}}, b: &pevMem{gu: two[1], pt: Vec{c.X + coupleGap, c.Y}},
			c: c, th: math.Pi, spin: 1}
		if r.Intn(2) == 0 {
			p.spin = -1
		}
		if !g.pevSendTo(p.a, p.a.pt) || !g.pevSendTo(p.b, p.b.pt) {
			pevReleaseAll([]*pevMem{p.a, p.b}, r)
			continue
		}
		used[two[0]], used[two[1]] = true, true
		s.pairs = append(s.pairs, p)
	}
	if len(s.pairs) == 0 {
		return false
	}
	a.D = s
	if pevDemo("parejas") != "" {
		pevDemoLog("parejas", center)
	}
	return true
}

func couplesUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*couplesState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase == cpOver {
		return true
	}
	if g.policeCalled && !s.police { // the police are coming: everybody runs
		couplesFlee(g, s, r)
		return true
	}
	live := 0
	for _, p := range s.pairs {
		if !p.stopped {
			g.coupleStep(s, p, r, dt)
		}
		if !p.stopped {
			live++
		}
	}
	if live == 0 {
		couplesEnd(g, s)
		return true
	}
	switch s.phase {
	case cpWalk:
		all := true
		for _, p := range s.pairs {
			if !p.stopped && !p.on && p.widow == nil {
				all = false
			}
		}
		if all || age > coupleWalkMax {
			for _, p := range s.pairs {
				if !p.stopped && !p.on && p.widow == nil { // never got there
					pevReleaseAll([]*pevMem{p.a, p.b}, r)
					p.stopped = true
				}
			}
			s.phase, s.t0 = cpDance, g.time
			for _, p := range s.pairs {
				if p.on {
					p.a.gu.icon, p.a.gu.iconT = pevPick(r, coupleCall), 2.2
					break
				}
			}
			s.crowd.gather(g, r, s.center, 3+r.Intn(3), 48, 64, func(gu *Guest) bool { return s.pevHas(gu) })
			s.nextSay = g.time + 3
		}
	case cpDance:
		s.crowd.update(g, r, nil, 0, nil)
		if g.time >= s.nextSay {
			s.nextSay = g.time + 2.5 + r.Float64()*3
			if len(s.crowd.mem) > 0 && r.Float64() < 0.5 {
				m := s.crowd.mem[r.Intn(len(s.crowd.mem))]
				if m.gu.icon == "" {
					m.gu.icon, m.gu.iconT = pevPick(r, coupleWatch), 1.6
				}
			} else if p := s.pairs[r.Intn(len(s.pairs))]; p.on && !p.stopped && p.b.gu.icon == "" {
				p.b.gu.icon, p.b.gu.iconT = pevPick(r, coupleSweet), 1.6
			}
		}
		if pevDemo("parejas") == "kill" && !s.demoK && age > 5 {
			s.demoK = true
			for _, p := range s.pairs {
				if p.on && !p.stopped {
					peleaKill(g, p.a.gu) // capture: one of them dies (not by the player)
					break
				}
			}
		}
		if age > s.length {
			couplesEnd(g, s)
			return true
		}
	}
	return false
}

// coupleStep: one couple (walking there, dancing, or the one left alone).
func (g *Game) coupleStep(s *couplesState, p *couplePair, r *rand.Rand, dt float64) {
	if p.widow != nil {
		w := p.widow
		if !w.mine() { // the game took him (he saw the killer, a fright...): it is his business now
			pevForget(w)
			p.stopped = true
			return
		}
		w.keep()
		el := g.time - p.widowT
		if el > coupleAloneT && !p.cried {
			p.cried = true
			w.pose(AnimTired)
			w.gu.icon, w.gu.iconT = pevPick(r, coupleCry), 1.6
		}
		if el > coupleAloneT+coupleCryT {
			pevScare(g, w.gu, r)
			p.stopped = true
		}
		return
	}
	am, bm := p.a.mine(), p.b.mine()
	if !am || !bm {
		var alive, dead *pevMem
		switch {
		case !am && bm:
			dead, alive = p.a, p.b
		case am && !bm:
			dead, alive = p.b, p.a
		}
		if dead != nil && dead.gu.dead && p.on && alive != nil {
			// killed: the other one keeps hugging nobody for a moment
			pevForget(dead)
			p.widow, p.widowT = alive, g.time
			alive.pose(AnimLove)
			alive.gu.icon, alive.gu.iconT = pevPick(r, coupleAlone), 1.4
			s.widows++
			s.alone++
			return
		}
		if dead != nil && dead.gu.dead && alive != nil { // killed while walking there: the other one runs
			s.widows++
			pevForget(dead)
			pevScare(g, alive.gu, r)
			p.stopped = true
			return
		}
		pevReleaseAll([]*pevMem{p.a, p.b}, r)
		p.stopped = true
		return
	}
	if !p.on {
		if peleaArrived(p.a) && peleaArrived(p.b) {
			p.on = true
			p.a.pose(AnimLove)
			p.b.pose(AnimLove)
		}
		return
	}
	p.a.keep()
	p.b.keep()
	p.th += p.spin * dt * 0.7
	off := Vec{math.Cos(p.th) * coupleGap, math.Sin(p.th) * coupleGap * 0.5}
	g.pogoMoveTo(p.a.gu, Vec{p.c.X + off.X, p.c.Y + off.Y}, dt)
	g.pogoMoveTo(p.b.gu, Vec{p.c.X - off.X, p.c.Y - off.Y}, dt)
	p.a.gu.dir = faceTo(p.a.gu.pos, p.b.gu.pos)
	p.b.gu.dir = faceTo(p.b.gu.pos, p.a.gu.pos)
}

func couplesFlee(g *Game, s *couplesState, r *rand.Rand) {
	for _, p := range s.pairs {
		for _, m := range []*pevMem{p.a, p.b} {
			if m.mine() {
				pevScare(g, m.gu, r)
			} else {
				pevForget(m)
			}
		}
		p.stopped = true
	}
	for _, m := range s.crowd.mem {
		if m.mine() {
			pevScare(g, m.gu, r)
		} else {
			pevForget(m)
		}
	}
	s.crowd.mem = nil
	s.phase = cpOver
}

func couplesEnd(g *Game, s *couplesState) {
	if s.phase == cpOver {
		return
	}
	r := partyDice()
	for _, p := range s.pairs {
		if p.widow != nil && !p.stopped && p.widow.mine() { // the one left alone still runs off
			pevScare(g, p.widow.gu, r)
		} else {
			pevReleaseAll([]*pevMem{p.a, p.b}, r)
		}
		p.stopped = true
	}
	s.crowd.release(g, r)
	s.phase = cpOver
}

var (
	heartCol   = color.RGBA{0xff, 0x5a, 0x8a, 255}
	heartColDk = color.RGBA{0xb8, 0x20, 0x50, 255}
)

// drawHeartPix: a 5x4 heart (px 1) with its top-left corner at (x, y).
func drawHeartPix(plot plotFn, x, y, px int, col color.RGBA) {
	plot(x, y, 2*px, px, col)
	plot(x+3*px, y, 2*px, px, col)
	plot(x, y+px, 5*px, px, col)
	plot(x+px, y+2*px, 3*px, px, col)
	plot(x+2*px, y+3*px, px, px, col)
}

// couplesItems: hearts rising over each dancing couple; a broken heart over the one left alone.
func couplesItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*couplesState)
	if s.phase != cpDance && s.phase != cpWalk {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for i, p := range s.pairs {
		if p.stopped {
			continue
		}
		if p.widow != nil {
			w := p.widow.gu
			items = append(items, drawItem{w.pos.Y + 0.4, func() {
				cx, cy := g.pevXY(w.pos)
				x, y := cx-2*px, cy-k(44)
				drawHeartPix(plot, x, y, px, heartColDk)
				plot(x+2*px, y, px, 4*px, color.RGBA{0x20, 0x10, 0x18, 255}) // the crack
			}})
			continue
		}
		if !p.on {
			continue
		}
		c, ph := p.c, float64(i)*0.37
		items = append(items, drawItem{c.Y + 0.4, func() {
			cx, cy := g.pevXY(c)
			for j := 0; j < 2; j++ {
				u := math.Mod(g.time*0.45+ph+float64(j)*0.5, 1)
				x := cx - 2*px + int(math.Round(math.Sin(u*6+float64(j)*2)*4))*px
				y := cy - k(40) - int(math.Round(u*16))*px
				if u > 0.85 {
					continue
				}
				drawHeartPix(plot, x, y, px, heartCol)
			}
		}})
	}
	return items
}
