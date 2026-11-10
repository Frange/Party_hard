package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "toiletq": THE TOILET QUEUE. Three to five guests are bursting and there is only one toilet: they stand in a
// LINE in front of its door, one behind the other, facing it, with an urgent face (speech bubbles, the odd one hops
// and crosses his legs, small yellow shake marks; NOT the blue sweat drop), and the line moves one at a time: the one
// in front goes in (hidden behind the door, the toilet shakes), comes out relieved, and everybody steps forward. Once,
// somebody jumps the queue and the one he skips shouts "¡Oye!".
//
// HOW THE GUESTS DO IT TODAY (game.go, toilets.go): a guest who needs it walks to a random POI "toilet" and
// updateToilet puts him inside (state gToilet, Guest.inside = the toilet prop, hidden by the drawing); on a portable
// toilet that is busy the next one waits where he is (Guest.queue) a few seconds, and if nobody comes out he has an
// accident. This event does NOT change that code: the occupant it sends in uses exactly the same state (gToilet +
// inside) so the normal visitors and the portable-toilet queue keep working around him; the line is only a scene
// of held guests (pevMem) in front of the door, and they are let go when it ends.
//
// WHEN: parties with a real toilet in the plan (or the fair / festival / beach, where the portable ones are the
// toilet), 20 %, between 60 and 240 s, once. Group "wc".
// ---------------------------------------------------------------------------

var toiletqIcons = map[string]bool{"festival": true, "funfair": true, "beach": true, "ranch": true}

const (
	tqSlot    = 13.0 // pixels between two people in the line
	tqMax     = 5
	tqMin     = 3
	tqStayMin = 3.4
	tqStayMax = 5.0
)

var tqUrgent = pevTexts("¡No aguanto más!", "¡Date prisa, por favor!", "¡Esto es una emergencia!", "Cruzo las piernas y rezo",
	"¡Tardas un mundo!", "¿Quién hace tanta cola?", "¡Me hago pis!", "¡Qué mal momento!")
var tqCut = pevTexts("¡Oye!", "¡Eh, hay cola!", "¡Al final de la cola!")
var tqCutter = pevTexts("¡Solo será un momento!", "¡Es que me urge mucho!")
var tqRelief = pevTexts("¡Gracias, qué alivio!", "Uf, ahora sí")

type tqState struct {
	door, dir Vec
	prop      *Prop
	line      []*pevMem
	occ       *Guest
	cutter    *pevMem
	cutDone   bool
	served    int
	next      float64
	say       float64
}

func (s *tqState) pevHas(gu *Guest) bool {
	return pevHasIn(s.line, gu) || s.cutter != nil && s.cutter.gu == gu || s.occ == gu
}

func init() {
	registerPartyEvent(&partyEvent{ID: "toiletq", Name: "Cola del baño", Group: "wc",
		Prob: 0.20, AtMin: 60, AtMax: 240, Dur: 40,
		Eligible: func(g *Game, t partyTraits) bool {
			return (t.toilet || toiletqIcons[t.icon]) && t.icon != "station" && t.roam && t.guests >= 28
		},
		Start: tqStart, Items: tqItems,
		Update: pevGuard(tqUpdate, func(g *Game, a *partyAct) { tqEnd(a.D.(*tqState)) })})
}

// tqToiletProp: the prop that hides the guest who goes in (the portable toilet at the door, or the toilet itself).
func (g *Game) tqToiletProp(door Vec) *Prop {
	if pr := g.pottyAt(door); pr != nil {
		return pr
	}
	var best *Prop
	bd := 34.0
	for _, p := range g.lvl.Props {
		if p.Name != "toilet" || p.gone {
			continue
		}
		if d := dist(door, Vec{p.X, p.Y + 10}); d < bd {
			best, bd = p, d
		}
	}
	return best
}

func (g *Game) tqSlotOK(p Vec) bool {
	l := g.lvl
	return l.Free(p) && !l.SolidP(p.X, p.Y) && !l.badGround(p) && !l.InPool(p)
}

// tqPlace: the direction in which the line forms (the first one that leaves n slots free and in sight of each other).
func (g *Game) tqPlace(door Vec, n int) (Vec, bool) {
	dirs := []Vec{{0, 1}, {-1, 0}, {1, 0}, {-0.7, 0.7}, {0.7, 0.7}, {0, -1}, {-0.7, -0.7}, {0.7, -0.7}}
	for _, d := range dirs {
		ok := true
		prev := door
		for i := 0; i < n && ok; i++ {
			p := Vec{door.X + d.X*(tqSlot*float64(i+1)+4), door.Y + d.Y*(tqSlot*float64(i+1)+4)}
			ok = g.tqSlotOK(p) && g.lvl.clearLine(prev, p)
			prev = p
		}
		if ok {
			return d, true
		}
	}
	return Vec{}, false
}

func (s *tqState) slot(i int) Vec {
	return Vec{s.door.X + s.dir.X*(tqSlot*float64(i+1)+4), s.door.Y + s.dir.Y*(tqSlot*float64(i+1)+4)}
}

func tqStart(g *Game, a *partyAct) bool {
	pois := g.lvl.POIs["toilet"]
	if len(pois) == 0 {
		return false
	}
	r := partyDice()
	n := tqMin + r.Intn(tqMax-tqMin+1)
	// a door with a prop to hide behind and room for the line; the first of the POIs in a random order
	start := r.Intn(len(pois))
	for k := 0; k < len(pois); k++ {
		door := pois[(start+k)%len(pois)]
		pr := g.tqToiletProp(door)
		if pr == nil || !g.pevPlayerFar(door, 50) {
			continue
		}
		d, ok := g.tqPlace(door, n)
		if !ok {
			continue
		}
		st := &tqState{door: door, dir: d, prop: pr}
		// the guests: the n nearest free ones with a way to their place
		var cand []*Guest
		for _, gu := range g.guests {
			if g.pevFreeGuest(gu) && dist(gu.pos, door) < 230 {
				cand = append(cand, gu)
			}
		}
		for i := 1; i < len(cand); i++ {
			for j := i; j > 0 && dist(cand[j].pos, door) < dist(cand[j-1].pos, door); j-- {
				cand[j], cand[j-1] = cand[j-1], cand[j]
			}
		}
		for _, gu := range cand {
			if len(st.line) >= n {
				break
			}
			m := &pevMem{gu: gu, next: g.time + 1 + r.Float64()*2.5}
			if g.pevSendTo(m, g.lvl.NearestFree(st.slot(len(st.line)))) {
				st.line = append(st.line, m)
			}
		}
		if len(st.line) < tqMin {
			tqEnd(st)
			continue
		}
		st.next = g.time + 9 // the first one goes in when they have settled
		a.D = st
		return true
	}
	return false
}

func tqEnd(s *tqState) {
	r := partyDice()
	pevReleaseAll(s.line, r)
	s.line = nil
	if s.cutter != nil {
		pevRelease(s.cutter, r)
		s.cutter = nil
	}
}

func (s *tqState) occupied() bool {
	return s.occ != nil && !s.occ.out() && s.occ.state == gToilet && s.occ.inside == s.prop
}

func (s *tqState) arrived(m *pevMem) bool {
	gu := m.gu
	return (gu.state == gIdle && gu.timer > 1000 || gu.state == gFun && gu.fun == 18) && dist(gu.pos, m.pt) < 4
}

// tqEnter: the guest steps into the toilet (same state as a normal visit: it hides him and the toilet shakes).
func (g *Game) tqEnter(s *tqState, gu *Guest) {
	r := partyDice()
	gu.state, gu.next, gu.path, gu.moving = gToilet, gIdle, nil, false
	gu.wcInit, gu.inside, gu.queue = true, s.prop, nil
	gu.timer = tqStayMin + r.Float64()*(tqStayMax-tqStayMin)
	gu.pos = s.door
	gu.dir = DirUp
	gu.icon = ""
	s.occ = gu
	s.served++
}

func tqUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*tqState)
	r := partyDice()
	// the ones who are not ours any more (frightened, killed) leave the line; the line closes up
	kept := s.line[:0]
	for _, m := range s.line {
		if m.mine() {
			kept = append(kept, m)
		}
	}
	s.line = kept
	if s.cutter != nil && !s.cutter.mine() {
		s.cutter = nil
	}
	if s.occ != nil && !s.occupied() {
		if s.occ != nil && !s.occ.out() && s.occ.state != gToilet && s.occ.iconT <= 0 {
			s.occ.icon, s.occ.iconT = pevPick(r, tqRelief), 1.8
		}
		s.occ = nil
	}
	if len(s.line) == 0 && s.cutter == nil && s.occ == nil {
		return true
	}
	// the line is where it should be: the first at the door, and every frame the held ones stand and shake
	for i, m := range s.line {
		m.pt = g.lvl.NearestFree(s.slot(i))
		gu := m.gu
		if gu.state == gGoto && dist(gu.pos, m.pt) > 3 {
			continue
		}
		if m.want == gGoto && gu.state == gIdle {
			m.want = gIdle
		}
		if dist(gu.pos, m.pt) > 4 && gu.state == gIdle && gu.timer > 1000 {
			g.pevSendTo(m, m.pt) // one slot forward
			continue
		}
		// urgency: cross the legs (the "pee dance" pose) in turns, else stand tapping
		if g.time >= m.next {
			m.next = g.time + 1.1 + r.Float64()*1.6
			if m.want == gFun {
				m.hold(gIdle)
			} else if r.Float64() < 0.75 {
				m.pose(AnimPee)
			}
		}
		m.keep()
		gu.dir = faceTo(gu.pos, s.door)
		if gu.icon == "" && g.time >= s.say && r.Float64() < dt*0.25 {
			s.say = g.time + 2.5
			gu.icon, gu.iconT = pevPick(r, tqUrgent), 2.2
		}
	}
	// the cutter walks to the door
	if s.cutter != nil {
		s.cutter.pt = g.lvl.NearestFree(Vec{s.door.X + s.dir.Y*9, s.door.Y - s.dir.X*9 + s.dir.Y*10})
		if s.cutter.mine() && s.cutter.gu.state == gIdle && s.cutter.gu.timer > 1000 {
			s.cutter.gu.dir = faceTo(s.cutter.gu.pos, s.door)
			s.cutter.keep()
		}
	}
	// the door is free: the cutter first (if he is there), else the head of the line
	if !s.occupied() && g.time >= s.next {
		switch {
		case s.cutter != nil && s.cutter.gu.state == gIdle && s.cutter.gu.timer > 1000 && dist(s.cutter.gu.pos, s.door) < 22:
			gu := s.cutter.gu
			s.cutter = nil
			if len(s.line) > 0 {
				first := s.line[0].gu
				first.icon, first.iconT = pevPick(r, tqCut), 2
			}
			g.tqEnter(s, gu)
			s.next = g.time + 0.8
		case len(s.line) > 0 && s.arrived(s.line[0]):
			m := s.line[0]
			s.line = s.line[1:]
			m.gu.state, m.gu.timer = gIdle, 0
			g.tqEnter(s, m.gu)
			// everybody steps forward (the next frame notices the slots changed)
			for _, o := range s.line {
				o.next = g.time + 0.2
			}
			for i, o := range s.line {
				o.pt = g.lvl.NearestFree(s.slot(i))
				g.pevSendTo(o, o.pt)
			}
			s.next = g.time + 0.8
		}
	}
	// somebody jumps the queue once, when the line has two or more and the first has gone in
	if !s.cutDone && s.served >= 1 && len(s.line) >= 2 && s.cutter == nil && g.time > s.next {
		for _, gu := range g.guests {
			if g.pevFreeGuest(gu) && dist(gu.pos, s.door) < 150 && dist(gu.pos, s.door) > 40 && !s.pevHas(gu) {
				m := &pevMem{gu: gu, pt: g.lvl.NearestFree(Vec{s.door.X + s.dir.Y*9, s.door.Y + s.dir.Y*10})}
				if g.pevSendTo(m, m.pt) {
					s.cutter, s.cutDone = m, true
					gu.icon, gu.iconT = pevPick(r, tqCutter), 2
					break
				}
			}
		}
	}
	return false
}

// drawShakePix: two short yellow marks at the hips of a figure (feet at cx, cy) that jiggle: "I have to go".
func drawShakePix(plot plotFn, cx, cy, px int, k func(float64) int, t float64) {
	y := color.RGBA{0xf4, 0xc4, 0x30, 255}
	o := int(math.Round(math.Sin(t*14))) * px
	plot(cx-k(9)+o, cy-k(14), px, 3*px, y)
	plot(cx-k(11)-o, cy-k(11), px, 2*px, y)
	plot(cx+k(8)-o, cy-k(14), px, 3*px, y)
	plot(cx+k(10)+o, cy-k(11), px, 2*px, y)
}

func tqItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*tqState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, m := range s.line {
		gu := m.gu
		if gu.state != gFun || gu.fun != 18 {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.1, func() {
			cx, cy := g.pevXY(gu.pos)
			drawShakePix(plot, cx, cy, px, k, g.time)
		}})
	}
	return items
}
