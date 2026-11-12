package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "pogo" (idea 316): a MOSH PIT. With the fast music of a club, a festival or the stadium, a group of 4 to 8
// ordinary guests forms a circle on the dance floor and spins in a whirl, jumping and shoving each other (the push of
// the character sheet), bouncing off one another; now and then one of them goes down and gets up again. It is pure
// atmosphere: nobody is hurt. A few guests watch from outside (a fan, open at the back: the player is never closed in).
//
// If the player kills one of them, the circle OPENS (the others step back and stare at the body) and then, half the
// times, it goes on with the ones left; the other half (or when the police are called, or a body falls near) all of
// them run off. Every phase has its time limit (walking 14 s, the open circle 2.5 s, the frame's pevGuard at 2*Dur),
// so nobody is left stuck; the movement only goes to walkable points (never into a wall or the water).
//
// WHEN: a party event of the "dance" group (never together with the conga, the dance battle, the couples or the dance
// contest), Big, chance 11 % only in the fast-music parties (pogoIcons) with 20+ guests, at most once a party.
// Never in STEALTH, the RETO DE HOY or the island (the frame). Capture: -pevent pogo (or pogo:fall, pogo:kill).
// ---------------------------------------------------------------------------

// pogoIcons: the fast-music parties (club, neon, festival, stadium, the 70s disco, the skate park).
var pogoIcons = map[string]bool{"disco": true, "neon": true, "festival": true, "stadium": true, "disco70": true, "skate": true}

var (
	pogoStart = pevTexts("¡POGO, POGO!", "¡Que empiece el pogo!", "¡A saltar todos!")
	pogoShout = pevTexts("¡Oi, oi, oi!", "¡Más fuerte!", "¡Hey, hey!", "¡Arriba esas manos!", "¡Pogo!")
	pogoFall  = pevTexts("¡Ay, mi espalda!", "¡Hombre al suelo!", "¡Arriba, colega!")
	pogoCrowd = pevTexts("¡Qué locos!", "Yo no me meto ahí", "¡Dale!", "¡Uooo!")
	pogoOpen  = pevTexts("¡Eh! ¿Qué ha pasado?", "¡Abrid el círculo!", "¡Para, para!")
)

const (
	pogoChance  = 0.11
	pogoMinN    = 4
	pogoMaxN    = 8
	pogoRad     = 17.0 // radius of the circle
	pogoOpenRad = 30.0 // radius when it opens around a body
	pogoMinLen  = 20.0
	pogoMaxLen  = 40.0
	pogoWalkMax = 14.0
	pogoOpenT   = 2.5
	pogoSpeed   = 42.0 // px/s at most when they shuffle round the circle
	pogoFallP   = 0.10 // a shove knocks the other one down
	pogoFallT   = 1.8
)

const (
	pgWalk = iota
	pgMosh
	pgOpen
	pgOver
)

type pogoState struct {
	mem      []*pevMem
	crowd    pevCrowd
	phase    int
	t0       float64
	center   Vec
	length   float64
	moshT    float64 // seconds of mosh done
	rot      float64
	spin     float64
	rad      float64
	bump     map[*Guest]float64 // radial push (px), decays
	shoveT   map[*Guest]float64 // game time until which he shows the shove
	fall     map[*Guest]float64 // game time until which he lies on the floor
	nextBump float64
	police   bool
	bodies   int
	killed   int  // members killed while it ran
	reopened bool // it went on after a kill
	fled     bool
	maxN     int // most members at once (tests)
	falls    int // falls so far (tests)
}

func (s *pogoState) pevHas(gu *Guest) bool { return pevHasIn(s.mem, gu) || s.crowd.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "pogo", Name: "Pogo", Group: "dance", Big: true,
		Prob: pogoChance, AtMin: 80, AtMax: 280, Dur: 34, MaxRuns: 1, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return pogoIcons[t.icon] && (t.dance || t.roam) && t.guests >= 20 },
		Start:    pogoStartFn, Items: pogoItems,
		Update: pevGuard(pogoUpdate, func(g *Game, a *partyAct) { pogoEnd(g, a.D.(*pogoState)) })})
}

// danceFloorSpot: a point of a dance floor with room `rad` around it (else an open spot of the party).
func (g *Game) danceFloorSpot(r *rand.Rand, rad float64) (Vec, bool) {
	for _, i := range r.Perm(len(g.lvl.Def.Dance)) {
		d := g.lvl.Def.Dance[i]
		for k := 0; k < 10; k++ {
			p := Vec{d.X + d.W*(0.3+0.4*r.Float64()), d.Y + d.H*(0.3+0.4*r.Float64())}
			if g.pevClear(p, rad) && g.pevPlayerFar(p, 60) {
				return p, true
			}
		}
	}
	return g.pevSpot(r, rad, false)
}

// danceFree: a free guest an event may take for a dance (named guests too, unlike the fight: with the asset folder
// most guests have a name; the stars with acts of their own are kept out by pevFreeGuest).
func danceFree(g *Game, gu *Guest) bool {
	return g.pevFreeGuest(gu) && pevFxs[gu] == nil && gu.state != gLounge
}

// danceCands: ordinary free guests within `within` px of c, nearest first.
func (g *Game) danceCands(c Vec, within float64) []*Guest {
	var cand []*Guest
	for _, gu := range g.guests {
		if danceFree(g, gu) && dist(gu.pos, c) < within && !fursuitSlow(gu) {
			cand = append(cand, gu)
		}
	}
	for i := 1; i < len(cand); i++ {
		for j := i; j > 0 && dist(cand[j].pos, c) < dist(cand[j-1].pos, c); j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	return cand
}

func pogoSlot(c Vec, a, rad float64) Vec {
	return Vec{c.X + math.Cos(a)*rad, c.Y + math.Sin(a)*rad*0.7}
}

func pogoStartFn(g *Game, a *partyAct) bool {
	r := partyDice()
	c, ok := g.danceFloorSpot(r, pogoOpenRad)
	if !ok {
		return false
	}
	cand := g.danceCands(c, 260)
	if len(cand) < pogoMinN+2 {
		return false
	}
	n := minI(pogoMinN+r.Intn(pogoMaxN-pogoMinN+1), len(cand)-2)
	s := &pogoState{center: c, t0: g.time, rad: pogoRad, spin: 1, police: g.policeCalled, bodies: len(g.bodies),
		length: pogoMinLen + r.Float64()*(pogoMaxLen-pogoMinLen),
		bump:   map[*Guest]float64{}, shoveT: map[*Guest]float64{}, fall: map[*Guest]float64{}}
	if r.Intn(2) == 0 {
		s.spin = -1
	}
	a0 := r.Float64() * 2 * math.Pi
	for i := 0; i < n; i++ {
		pt := pogoSlot(c, a0+2*math.Pi*float64(i)/float64(n), pogoRad)
		if !g.lvl.Free(pt) || g.lvl.SolidP(pt.X, pt.Y) {
			pt = c
		}
		m := &pevMem{gu: cand[i], pt: pt, idx: i}
		if g.pevSendTo(m, pt) {
			s.mem = append(s.mem, m)
		}
	}
	if len(s.mem) < pogoMinN {
		pevReleaseAll(s.mem, r)
		return false
	}
	s.maxN = len(s.mem)
	a.D = s
	if pevDemo("pogo") != "" {
		pevDemoLog("pogo", c)
	}
	return true
}

// pogoAlarm: the police were called, or a body (not of the pit) fell near.
func (g *Game) pogoAlarm(s *pogoState) bool {
	if g.policeCalled && !s.police {
		return true
	}
	for i := s.bodies; i < len(g.bodies); i++ {
		if dist(g.bodies[i].pos, s.center) < 140 {
			return true
		}
	}
	return false
}

// pogoPrune drops the members that are not ours any more; returns how many of them died.
func (s *pogoState) prune() int {
	dead := 0
	out := s.mem[:0]
	for _, m := range s.mem {
		if m.mine() {
			out = append(out, m)
			continue
		}
		if m.gu.dead {
			dead++
		}
		pevForget(m)
		delete(s.fall, m.gu)
	}
	s.mem = out
	return dead
}

func pogoUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*pogoState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase == pgOver {
		return true
	}
	dead := s.prune()
	if dead > 0 {
		s.killed += dead
		s.bodies += dead // the body of a member is handled here (the circle opens), not as an alarm
		if s.phase == pgWalk || len(s.mem) < 3 {
			pogoFlee(g, s, r)
			return true
		}
		s.phase, s.t0 = pgOpen, g.time
		for _, m := range s.mem {
			delete(s.fall, m.gu)
			m.pose(AnimPanic)
			if m.gu.icon == "" && r.Float64() < 0.5 {
				m.gu.icon, m.gu.iconT = pevPick(r, pogoOpen), 1.6
			}
		}
		return false
	}
	if g.pogoAlarm(s) {
		pogoFlee(g, s, r)
		return true
	}
	if len(s.mem) < 3 { // the others were taken by something else (a fright, a star's act)
		pogoEnd(g, s)
		return true
	}
	if s.phase != pgWalk {
		s.crowd.update(g, r, nil, 0, nil)
	}
	switch s.phase {
	case pgWalk:
		arrived := 0
		for _, m := range s.mem {
			if peleaArrived(m) {
				arrived++
			}
		}
		if arrived == len(s.mem) || age > pogoWalkMax && arrived >= pogoMinN {
			var in []*pevMem
			for _, m := range s.mem {
				if peleaArrived(m) {
					in = append(in, m)
				} else {
					pevRelease(m, r)
				}
			}
			s.mem = in
			s.phase, s.t0, s.nextBump = pgMosh, g.time, g.time+0.6
			for i, m := range s.mem {
				m.pose(pogoDanceAnim(i))
			}
			s.mem[0].gu.icon, s.mem[0].gu.iconT = pevPick(r, pogoStart), 2
			s.crowd.gather(g, r, s.center, 3+r.Intn(3), 42, 56, func(gu *Guest) bool { return pevHasIn(s.mem, gu) })
		} else if age > pogoWalkMax {
			pogoEnd(g, s)
			return true
		}
	case pgMosh:
		s.moshT += dt
		g.pogoMosh(s, r, dt)
		if pevDemo("pogo") == "kill" && s.killed == 0 && s.moshT > 6 && len(s.mem) > 0 {
			peleaKill(g, s.mem[0].gu) // capture: one of them dies (not by the player: no stars)
		}
		if s.moshT > s.length {
			pogoEnd(g, s)
			return true
		}
	case pgOpen:
		// the circle opens: they step back from the body and stare at it
		for _, m := range s.mem {
			m.keep()
			out := faceVec(s.center, m.gu.pos)
			g.pogoMoveTo(m.gu, Vec{s.center.X + out.X*pogoOpenRad, s.center.Y + out.Y*pogoOpenRad*0.7}, dt)
			m.gu.dir = faceTo(m.gu.pos, s.center)
		}
		if age > pogoOpenT {
			if len(s.mem) >= pogoMinN-1 && r.Float64() < 0.5 { // it goes on with the ones left
				s.phase, s.reopened = pgMosh, true
				s.nextBump = g.time + 0.8
				for i, m := range s.mem {
					m.pose(pogoDanceAnim(i))
				}
			} else {
				pogoFlee(g, s, r)
				return true
			}
		}
	}
	return false
}

func pogoDanceAnim(i int) int {
	if i%2 == 0 {
		return AnimDance3 // headbang / bounce
	}
	return AnimDance0 // fist pump
}

// pogoMoveTo: a short step towards p, only onto walkable ground.
func (g *Game) pogoMoveTo(gu *Guest, p Vec, dt float64) {
	d := dist(gu.pos, p)
	if d < 0.3 {
		return
	}
	step := math.Min(d, pogoSpeed*dt)
	q := Vec{gu.pos.X + (p.X-gu.pos.X)/d*step, gu.pos.Y + (p.Y-gu.pos.Y)/d*step}
	l := g.lvl
	if l.Free(q) && !l.SolidP(q.X, q.Y) && !l.InPool(q) && !l.badGround(q) {
		gu.pos = q
	}
}

// pogoMosh: the whirl, the shoves and the falls.
func (g *Game) pogoMosh(s *pogoState, r *rand.Rand, dt float64) {
	n := len(s.mem)
	s.maxN = maxI(s.maxN, n)
	s.rot += s.spin * dt * 0.9
	if r.Float64() < dt*0.08 {
		s.spin = -s.spin // the whirl turns the other way now and then
	}
	if s.rad < pogoRad {
		s.rad = math.Min(pogoRad, s.rad+dt*8)
	}
	if g.time >= s.nextBump && n >= 2 {
		s.nextBump = g.time + 0.35 + r.Float64()*0.45
		i := r.Intn(n)
		j := (i + 1) % n
		a, b := s.mem[i].gu, s.mem[j].gu
		if g.time >= s.fall[a] && g.time >= s.fall[b] {
			s.mem[i].pose(AnimPunch)
			s.shoveT[a] = g.time + 0.35
			s.bump[a], s.bump[b] = -5, 6
			a.dir = faceTo(a.pos, b.pos)
			if r.Float64() < 0.2 {
				g.snd.Play("thud")
			}
			if r.Float64() < pogoFallP || pevDemo("pogo") == "fall" && s.falls == 0 && s.moshT > 3 {
				if !pogoAnyDown(s, g.time) {
					s.fall[b] = g.time + pogoFallT
					s.falls++
					b.icon, b.iconT = pevPick(r, pogoFall), 1.6
				}
			} else if a.icon == "" && r.Float64() < 0.15 {
				a.icon, a.iconT = pevPick(r, pogoShout), 1.3
			}
		}
	}
	for i, m := range s.mem {
		gu := m.gu
		m.keep()
		if t, ok := s.shoveT[gu]; ok && g.time >= t {
			delete(s.shoveT, gu)
			m.pose(pogoDanceAnim(i))
		}
		if t, ok := s.fall[gu]; ok && g.time >= t {
			delete(s.fall, gu)
		}
		bp := s.bump[gu]
		bp -= math.Copysign(math.Min(math.Abs(bp), dt*14), bp)
		s.bump[gu] = bp
		if g.time < s.fall[gu] {
			continue // lying: he stays where he fell
		}
		ang := s.rot + 2*math.Pi*float64(i)/float64(n)
		g.pogoMoveTo(gu, pogoSlot(s.center, ang, s.rad+bp), dt)
		if _, sh := s.shoveT[gu]; !sh {
			gu.dir = faceTo(gu.pos, s.center)
		}
	}
	for _, m := range s.crowd.mem {
		if m.gu.icon == "" && r.Float64() < dt*0.15 {
			m.gu.icon, m.gu.iconT = pevPick(r, pogoCrowd), 1.4
		}
	}
}

func pogoAnyDown(s *pogoState, now float64) bool {
	for _, t := range s.fall {
		if now < t {
			return true
		}
	}
	return false
}

// pogoFlee: the pit and the watchers run off.
func pogoFlee(g *Game, s *pogoState, r *rand.Rand) {
	s.fled = true
	for _, m := range append(append([]*pevMem(nil), s.mem...), s.crowd.mem...) {
		if m.mine() {
			pevScare(g, m.gu, r)
		} else {
			pevForget(m)
		}
	}
	s.mem, s.crowd.mem = nil, nil
	s.fall = map[*Guest]float64{}
	s.phase = pgOver
}

func pogoEnd(g *Game, s *pogoState) {
	if s.phase == pgOver {
		return
	}
	r := partyDice()
	pevReleaseAll(s.mem, r)
	s.crowd.release(g, r)
	s.fall = map[*Guest]float64{}
	s.phase = pgOver
}

// drawGuest: the pit draws its own members (a jump on the beat, or lying on the floor after a fall); pe_fx.go asks.
func (s *pogoState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if s.phase != pgMosh || !pevHasIn(s.mem, gu) {
		return false
	}
	if g.time < s.fall[gu] {
		g.drawLying(dst, gu.cs, false, pos)
		return true
	}
	anim, phase := g.guestPose(gu)
	yoff := -math.Round(math.Abs(math.Sin(g.time*7+gu.off*5)) * 3)
	g.drawActor(dst, gu.cs, anim, phase, pos, gu.dir, yoff, false)
	return true
}

var pogoBumpCol = color.RGBA{0xff, 0xf0, 0x80, 255}

// pogoItems: a little burst where a shove lands, stars over the one on the floor, and notes over the pit.
func pogoItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*pogoState)
	if s.phase != pgMosh {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, m := range s.mem {
		gu := m.gu
		if g.time < s.fall[gu] {
			items = append(items, drawItem{gu.pos.Y + 0.4, func() {
				cx, cy := g.pevXY(gu.pos)
				drawStarsPix(plot, cx, cy-k(8), px, g.time)
			}})
			continue
		}
		if t, ok := s.shoveT[gu]; ok && t-g.time > 0.15 {
			items = append(items, drawItem{gu.pos.Y + 0.4, func() {
				cx, cy := g.pevXY(gu.pos)
				v := dirVec(gu.dir)
				x, y := cx+int(math.Round(v.X*8))*px, cy-k(18)
				plot(x-px, y, 3*px, px, pogoBumpCol)
				plot(x, y-px, px, 3*px, pogoBumpCol)
			}})
		}
	}
	c := s.center
	items = append(items, drawItem{c.Y + 0.2, func() {
		cx, cy := g.pevXY(c)
		drawNotesPix(plot, cx, cy-k(46), px, g.time*1.6, 3)
	}})
	return items
}
