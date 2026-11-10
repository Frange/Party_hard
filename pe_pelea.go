package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "pelea": two ordinary guests have words and come to blows. They walk up to each other, shout, and trade
// punches (the punch of the character sheet); every hit shoves the other one back a little. A few guests stop to watch
// ("¡Pelea, pelea!"). It ends by itself: most times they calm down, sometimes one of them falls (stars, then up), and
// now and then one of them dies. That death is NOT the player's: the body is not his (byPlayer false), nobody reports
// it as his, it raises no stars and no suspicion of him; the watchers run off. The player can break the fight up with E
// next to them: both calm down and he gets points.
//
// WHEN: it is a MINOR event (partyevents.go): it does not take one of the three places of the party and is not kept out
// by the memory. In almost every level (not at the kids' birthday), every ~2-2.5 minutes: the first one at 35-120 s,
// then again 90-140 s after the previous one ends, up to 3 per party. Not in the RETO DE HOY, never in STEALTH or Jurassic.
// ---------------------------------------------------------------------------

var (
	peleaShout = pevTexts("¿Qué miras?", "¡Repite eso!", "¡Esa era MI copa!", "¡Me has pisado!", "¡Tú a mí no me hablas así!",
		"¡Fuera de mi pista!", "¿Tienes algún problema?", "¡Has empezado tú!")
	peleaHit   = pevTexts("¡Toma!", "¡Ay!", "¡Pum!", "¡Au!", "¡Zas!")
	peleaCrowd = pevTexts("¡Pelea, pelea!", "¡Uuuh!", "¡Separadlos!", "¡Dale!", "¡Que alguien haga algo!")
	peleaCalm  = pevTexts("Vale, vale... ya está", "Bueno, perdona", "¿Amigos?", "Ya me había cansado")
	peleaDown  = pevTexts("¡K.O.!", "Veo pajaritos...")
	peleaDead  = pevTexts("¡Se le ha ido la mano!", "Una pelea ha acabado muy mal")
	peleaStop  = pevTexts("¡Paz!", "Gracias por separarnos")
	pevRunAway = pevTexts("¡AAAH!", "¡CORRE!", "¡Al suelo!")
)

const (
	peleaChance   = 0.90
	peleaBonus    = 60   // points for breaking a fight up
	peleaDeathP   = 0.15 // one of them dies
	peleaKOP      = 0.30 // one of them falls and gets up
	peleaGap      = 14.0 // distance between the two while they fight
	peleaMaxRuns  = 3
	peleaCooldown = 115.0
)

const (
	peleaWalk = iota
	peleaArgue
	peleaBrawl
	peleaKO
	peleaOver
)

type peleaState struct {
	a, b    *pevMem
	crowd   pevCrowd
	phase   int
	t0      float64
	center  Vec
	axis    Vec // unit vector from a to b
	nextHit float64
	hitter  int // 0 = a hits, 1 = b hits
	brawl   float64
	ko      *pevMem
	stopped bool
}

func (s *peleaState) pevHas(gu *Guest) bool {
	return s.a != nil && s.a.gu == gu || s.b != nil && s.b.gu == gu || s.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "pelea", Name: "Pelea a puñetazos", Group: "fight", Minor: true,
		Prob: peleaChance, AtMin: 35, AtMax: 120, Dur: 25, Cooldown: peleaCooldown, MaxRuns: peleaMaxRuns,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && t.guests >= 15 && t.icon != "cake" },
		Start:    peleaStart, Items: peleaItems,
		Update: pevGuard(peleaUpdate, func(g *Game, a *partyAct) { peleaEnd(g, a.D.(*peleaState)) })})
}

// peleaCanBe: an ordinary guest (no star, no name, no act of his own) who is free.
func peleaCanBe(g *Game, gu *Guest) bool {
	return g.pevFreeGuest(gu) && gu.name == "" && pevFxs[gu] == nil && gu.state != gLounge
}

func peleaStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if peleaCanBe(g, gu) && g.pevPlayerFar(gu.pos, 50) {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 6 {
		return false
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	for _, ga := range cand[:minI(len(cand), 8)] {
		// the other one: the nearest free guest within 120 px
		var gb *Guest
		for _, o := range cand {
			if o != ga && dist(o.pos, ga.pos) < 120 && (gb == nil || dist(o.pos, ga.pos) < dist(gb.pos, ga.pos)) {
				gb = o
			}
		}
		if gb == nil {
			continue
		}
		c := g.lvl.NearestFree(Vec{(ga.pos.X + gb.pos.X) / 2, (ga.pos.Y + gb.pos.Y) / 2})
		if !g.pevClear(c, peleaGap) || !g.pevPlayerFar(c, 40) {
			continue
		}
		ax := Vec{1, 0}
		if ga.pos.X > gb.pos.X {
			ax = Vec{-1, 0}
		}
		pa := Vec{c.X - ax.X*peleaGap/2, c.Y}
		pb := Vec{c.X + ax.X*peleaGap/2, c.Y}
		s := &peleaState{a: &pevMem{gu: ga, pt: pa}, b: &pevMem{gu: gb, pt: pb}, center: c, axis: ax, t0: g.time, hitter: r.Intn(2)}
		if !g.pevSendTo(s.a, pa) || !g.pevSendTo(s.b, pb) {
			pevReleaseAll([]*pevMem{s.a, s.b}, r)
			continue
		}
		s.brawl = 7 + r.Float64()*5
		a.D = s
		return true
	}
	return false
}

// peleaArrived: he reached his spot and stands there held.
func peleaArrived(m *pevMem) bool {
	return m.gu.state == gIdle && m.gu.timer > 1000 && dist(m.gu.pos, m.pt) < 5
}

func (s *peleaState) both() bool { return s.a.mine() && s.b.mine() }

// peleaFace: they look at each other.
func (s *peleaState) face() {
	s.a.gu.dir = faceTo(s.a.gu.pos, s.b.gu.pos)
	s.b.gu.dir = faceTo(s.b.gu.pos, s.a.gu.pos)
}

func peleaUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*peleaState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase != peleaKO && s.phase != peleaOver && !s.both() { // something else took one of them (a fright, a murder)
		peleaEnd(g, s)
		return true
	}
	switch s.phase {
	case peleaWalk:
		if peleaArrived(s.a) && peleaArrived(s.b) {
			s.phase, s.t0 = peleaArgue, g.time
			s.a.want, s.b.want = gIdle, gIdle
			s.face()
			s.a.gu.icon, s.a.gu.iconT = pevPick(r, peleaShout), 1.8
			s.crowd.gather(g, r, s.center, 3+r.Intn(3), 30, 46, func(gu *Guest) bool { return gu == s.a.gu || gu == s.b.gu })
		} else if age > 14 {
			peleaEnd(g, s)
			return true
		}
	case peleaArgue:
		s.a.keep()
		s.b.keep()
		s.face()
		s.crowd.update(g, r, nil, 0, nil)
		if age > 1.2 && s.b.gu.icon == "" {
			s.b.gu.icon, s.b.gu.iconT = pevPick(r, peleaShout), 1.6
		}
		if age > 2.6 {
			s.phase, s.t0, s.nextHit = peleaBrawl, g.time, g.time+0.2
		}
	case peleaBrawl:
		s.face()
		s.crowd.update(g, r, nil, 0, nil)
		for _, m := range s.crowd.mem {
			if m.gu.icon == "" && r.Float64() < dt*0.35 {
				m.gu.icon, m.gu.iconT = pevPick(r, peleaCrowd), 1.5
			}
		}
		hit, got := s.a, s.b
		if s.hitter == 1 {
			hit, got = s.b, s.a
		}
		if g.time >= s.nextHit {
			s.nextHit = g.time + 0.55 + r.Float64()*0.5
			hit.pose(AnimPunch)
			got.pose(AnimIdle)
			peleaShove(g, s, got, hit)
			if r.Float64() < 0.35 && got.gu.icon == "" {
				got.gu.icon, got.gu.iconT = pevPick(r, peleaHit), 0.9
			}
			if r.Float64() < 0.5 {
				g.snd.Play("thud")
			}
			if r.Float64() < 0.6 { // the other one hits back
				s.hitter = 1 - s.hitter
			}
		}
		hit.keep()
		got.keep()
		if age > s.brawl {
			peleaOutcome(g, s, r, got)
		}
	case peleaKO:
		if s.ko != nil && s.ko.mine() {
			s.ko.keep()
		}
		s.crowd.update(g, r, nil, 0, nil)
		if age > 2.8 {
			peleaEnd(g, s)
			return true
		}
	case peleaOver:
		return true
	}
	return false
}

// peleaShove: the one who got hit goes back a little along the axis (only where he can stand), never far from his spot.
func peleaShove(g *Game, s *peleaState, got, hit *pevMem) {
	dir := faceVec(hit.gu.pos, got.gu.pos)
	p := Vec{got.gu.pos.X + dir.X*2.5, got.gu.pos.Y + dir.Y*2.5}
	if dist(p, s.center) < peleaGap && g.lvl.Free(p) && !g.lvl.SolidP(p.X, p.Y) {
		got.gu.pos = p
	}
	// the hitter steps in after him (keeps the gap)
	q := Vec{hit.gu.pos.X + dir.X*2, hit.gu.pos.Y + dir.Y*2}
	if dist(q, got.gu.pos) > peleaGap*0.7 && dist(q, s.center) < peleaGap && g.lvl.Free(q) && !g.lvl.SolidP(q.X, q.Y) {
		hit.gu.pos = q
	}
}

// peleaOutcome: they calm down, one falls (and gets up), or one dies.
func peleaOutcome(g *Game, s *peleaState, r *rand.Rand, loser *pevMem) {
	roll := r.Float64()
	if dbgPeleaDie { // -peleadie (debug): the fight always ends with a death
		roll = 0
	}
	switch {
	case roll < peleaDeathP:
		winner := s.a
		if loser == s.a {
			winner = s.b
		}
		s.crowdFlee(g, r)
		peleaKill(g, loser.gu)
		pevRelease(winner, r)
		winner.gu.icon, winner.gu.iconT = "...", 1.5
		g.say(pevPick(r, peleaDead))
		g.peleaNote("pelea_muerto")
		s.phase = peleaOver
	case roll < peleaDeathP+peleaKOP:
		loser.pose(AnimLying)
		loser.gu.icon, loser.gu.iconT = pevPick(r, peleaDown), 1.8
		g.snd.Play("thud")
		winner := s.a
		if loser == s.a {
			winner = s.b
		}
		pevRelease(winner, r)
		s.ko, s.phase, s.t0 = loser, peleaKO, g.time
	default:
		s.a.gu.icon, s.a.gu.iconT = pevPick(r, peleaCalm), 1.8
		peleaEnd(g, s)
		s.phase = peleaOver
	}
}

// peleaKill: a guest dies in a fight between guests. Not the player's kill: the body is nobody's (it is not reported as
// his, it gives him no stars nor suspicion) and it gives him no points; it counts like any other body for the party.
// dbgPeleaDie: -peleadie (debug) makes every fight deadly, to see the police come for a body that is not the player's.
var dbgPeleaDie bool

func peleaKill(g *Game, gu *Guest) {
	if gu == nil || gu.out() {
		return
	}
	cr, kb, nh := g.credit, g.killByPlayer, g.noHeat
	g.credit, g.killByPlayer, g.noHeat = false, false, true
	g.killGuest(gu, false)
	g.credit, g.killByPlayer, g.noHeat = cr, kb, nh
	if n := len(g.bodies); n > 0 {
		g.bodies[n-1].byPlayer = false
	}
}

// peleaNote: a fact of this party (the newspaper may tell it).
func (g *Game) peleaNote(id string) {
	if g.pev.ran == nil {
		g.pev.ran = map[string]bool{}
	}
	g.pev.ran[id] = true
}

// crowdFlee: the watchers run away (their own fright, with the dice of the frame).
func (s *peleaState) crowdFlee(g *Game, r *rand.Rand) {
	for _, m := range s.crowd.mem {
		if m.mine() {
			pevScare(g, m.gu, r)
		} else {
			pevForget(m)
		}
	}
	s.crowd.mem = nil
}

// pevScare: the guest runs off for a few seconds (like scare(), but with the frame's dice: g.rng is never moved).
func pevScare(g *Game, gu *Guest, r *rand.Rand) {
	if gu == nil || gu.out() || !lifeFree(gu) {
		return
	}
	gu.state, gu.timer, gu.path, gu.dur = gScared, 2.5+r.Float64()*2.5, nil, 0
	if gu.icon == "" && r.Float64() < 0.4 {
		gu.icon, gu.iconT = pevPick(r, pevRunAway), 1.3
	}
}

func peleaEnd(g *Game, s *peleaState) {
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
}

// pevUse: E next to the fighters breaks it up (once): they calm down and the player gets points.
func (s *peleaState) pevUse(g *Game, p *Player) bool {
	if s.stopped || (s.phase != peleaArgue && s.phase != peleaBrawl) || dist(p.pos, s.center) > 34 {
		return false
	}
	r := partyDice()
	s.stopped = true
	s.a.gu.icon, s.a.gu.iconT = pevPick(r, peleaStop), 1.8
	s.b.gu.icon, s.b.gu.iconT = pevPick(r, peleaCalm), 1.8
	g.award(s.center, peleaBonus, peleaStop[0])
	g.snd.Play("spark")
	peleaEnd(g, s)
	s.phase = peleaOver
	return true
}

// peleaItems: little impact stars between them while they trade blows, stars over the head of the one on the floor.
func peleaItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*peleaState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	switch s.phase {
	case peleaBrawl:
		if int(g.time*8)%3 == 0 {
			c := Vec{(s.a.gu.pos.X + s.b.gu.pos.X) / 2, (s.a.gu.pos.Y + s.b.gu.pos.Y) / 2}
			items = append(items, drawItem{c.Y + 0.4, func() {
				cx, cy := g.pevXY(c)
				y := cy - k(22)
				plot(cx-px, y-2*px, px*2, px*5, pevWhite)
				plot(cx-2*px-px, y, px*7-px, px, pevWhite)
				plot(cx, y-px, px, px*3, pevGold)
			}})
		}
	case peleaKO:
		if s.ko != nil && !s.ko.gu.out() {
			gu := s.ko.gu
			items = append(items, drawItem{gu.pos.Y + 0.3, func() {
				cx, cy := g.pevXY(gu.pos)
				drawStarsPix(plot, cx, cy-k(11), px, g.time)
			}})
		}
	}
	return items
}
