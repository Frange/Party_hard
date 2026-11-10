package main

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "tiroteo": two guests who did not like each other pull out pistols in the middle of the party. They stop where
// they are, shout, and shoot at each other for a few seconds (a flash at the barrel and a thin trace, no extra blood);
// everybody around runs off. Sometimes one of them falls: that body is NOT the player's (byPlayer false: no stars, no
// suspicion, no points for him). The noise makes one witness phone the police by the usual rules (a caller who goes to
// the phone; nobody calls if the police are already called or somebody is already calling). Then the one left runs away.
//
// WHEN: rare (about one party in three or four where it fits): only in rough venues (casino, disco, neon, rooftop,
// motorway, train station, boat, circuit), 40 % of the parties (and the memory of the last two parties), between
// 90 and 260 s, once. Big. Never in the RETO DE HOY (DailyOK false), never in STEALTH or Jurassic (the frame).
// ---------------------------------------------------------------------------

var (
	tiroShout = pevTexts("¡Tú otra vez!", "¡Esta fiesta es pequeña para los dos!", "¡Quieto ahí!", "¡Me debes dinero!")
	tiroCrowd = pevTexts("¡Que lleva pistola!", "¡Todos al suelo!", "¡Llamad a la policía!")
	tiroEnd   = pevTexts("¡Tiroteo en la fiesta!", "Alguien ha caído en el tiroteo")
)

// tiroIcons: the venues where a shooting fits.
var tiroIcons = map[string]bool{"casino": true, "disco": true, "neon": true, "roof": true, "highway": true, "train": true,
	"boat": true, "circuit": true}

const (
	tiroChance = 0.40
	tiroHitP   = 0.08 // each shot that hits the other one
	tiroFlee   = 170.0
	tiroFlash  = 0.09
)

const (
	tiroStand = iota
	tiroShoot
	tiroOver
)

type tiroShot struct {
	from, to Vec
	t0       float64
}

type tiroState struct {
	a, b    *pevMem
	phase   int
	t0      float64
	next    float64
	who     int
	shots   []tiroShot
	length  float64
	called  bool
	dead    bool
	center  Vec
	running []*Guest
}

func (s *tiroState) pevHas(gu *Guest) bool {
	return s.a != nil && s.a.gu == gu || s.b != nil && s.b.gu == gu
}

func init() {
	registerPartyEvent(&partyEvent{ID: "tiroteo", Name: "Tiroteo entre invitados", Group: "fight", Big: true,
		Prob: tiroChance, AtMin: 90, AtMax: 260, Dur: 16, MaxRuns: 1, Prio: 2,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && t.guests >= 20 && tiroIcons[t.icon] },
		Start:    tiroStart, Items: tiroItems,
		Update: pevGuard(tiroUpdate, func(g *Game, a *partyAct) { tiroEnd2(g, a.D.(*tiroState)) })})
}

func tiroStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if peleaCanBe(g, gu) && g.pevPlayerFar(gu.pos, 70) && gu.state != gDance {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 6 {
		return false
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	for _, ga := range cand[:minI(len(cand), 10)] {
		for _, gb := range cand {
			d := dist(ga.pos, gb.pos)
			if gb == ga || d < 55 || d > 150 || !g.lvl.LOS(ga.pos, gb.pos) || !g.lvl.clearLine(ga.pos, gb.pos) {
				continue
			}
			s := &tiroState{a: &pevMem{gu: ga, pt: ga.pos}, b: &pevMem{gu: gb, pt: gb.pos}, t0: g.time, who: r.Intn(2),
				length: 6 + r.Float64()*4, center: Vec{(ga.pos.X + gb.pos.X) / 2, (ga.pos.Y + gb.pos.Y) / 2}}
			s.a.pose(AnimPoint)
			s.b.pose(AnimPoint)
			s.face()
			ga.icon, ga.iconT = pevPick(r, tiroShout), 1.8
			a.D = s
			return true
		}
	}
	return false
}

func (s *tiroState) face() {
	s.a.gu.dir = faceTo(s.a.gu.pos, s.b.gu.pos)
	s.b.gu.dir = faceTo(s.b.gu.pos, s.a.gu.pos)
}

// tiroScatter: everybody free near the shooters runs off (the frame's dice; guests of other events are let be).
func tiroScatter(g *Game, s *tiroState, r *rand.Rand) {
	shouts := 0
	for _, gu := range g.guests {
		if gu.out() || s.pevHas(gu) || g.pevTaken(gu) || gu.state == gScared || gu.state == gSwim {
			continue
		}
		if dist(gu.pos, s.center) > tiroFlee || !lifeFree(gu) || gu.perf != "" || gu.state == gKiller || gu.state == gZombie {
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gDance, gLounge, gDrink, gChat, gChatGo, gWaitP, gKiss, gDanceWith:
		default:
			continue
		}
		tiroUnpair(gu)
		pevScare(g, gu, r)
		if shouts < 2 && gu.icon == "" {
			gu.icon, gu.iconT = pevPick(r, tiroCrowd), 1.5
			shouts++
		}
	}
}

// tiroUnpair: like alarm(): a guest who runs off leaves his partner (chat, kiss) free.
func tiroUnpair(gu *Guest) {
	if p := gu.partner; p != nil && p.partner == gu {
		p.partner = nil
		if p.state == gChat || p.state == gKiss || p.state == gWaitP {
			p.state, p.timer = gIdle, 1
		}
	}
	gu.partner = nil
}

// tiroCall: the noise: one witness goes to phone the police by the usual rules (nobody if they are called already).
func tiroCall(g *Game, s *tiroState, r *rand.Rand) {
	s.called = true
	if !g.callerAvailable() {
		return
	}
	var best *Guest
	for _, gu := range g.guests {
		if gu.out() || s.pevHas(gu) || g.pevTaken(gu) || !lifeFree(gu) || !g.canWatch(gu) || gu.state == gKiller || gu.state == gZombie {
			continue
		}
		if d := dist(gu.pos, s.center); d < 260 && (best == nil || d < dist(best.pos, s.center)) {
			best = gu
		}
	}
	if best == nil {
		return
	}
	tiroUnpair(best)
	g.callerGuest = best
	g.claimBodies(best)
	best.state, best.timer, best.path = gAlarm, 1+r.Float64()*1.3, nil
	best.icon, best.iconT = tiroCrowd[2], 1.6
	if g.alertPos == (Vec{}) {
		g.alertPos = s.center
	}
}

func tiroUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*tiroState)
	r := partyDice()
	age := g.time - s.t0
	alive := func(m *pevMem) bool { return m.mine() }
	switch s.phase {
	case tiroStand:
		if !alive(s.a) || !alive(s.b) {
			tiroEnd2(g, s)
			return true
		}
		s.a.keep()
		s.b.keep()
		s.face()
		if age > 1.0 && s.b.gu.icon == "" {
			s.b.gu.icon, s.b.gu.iconT = pevPick(r, tiroShout), 1.5
		}
		if age > 1.8 {
			s.phase, s.t0, s.next = tiroShoot, g.time, g.time
			tiroScatter(g, s, r)
			g.say(tiroEnd[0])
		}
	case tiroShoot:
		if !alive(s.a) || !alive(s.b) { // one fell, or something else took one of them
			tiroEnd2(g, s)
			return true
		}
		s.a.keep()
		s.b.keep()
		s.face()
		if g.time >= s.next {
			s.next = g.time + 0.45 + r.Float64()*0.45
			sh, tg := s.a, s.b
			if s.who == 1 {
				sh, tg = s.b, s.a
			}
			s.who = 1 - s.who
			from, to := tiroBarrel(sh.gu), Vec{tg.gu.pos.X + (r.Float64()-0.5)*14, tg.gu.pos.Y - 14 + (r.Float64()-0.5)*10}
			s.shots = append(s.shots, tiroShot{from: from, to: to, t0: g.time})
			g.snd.Play("clac")
			if !s.called {
				tiroCall(g, s, r)
			}
			if r.Float64() < tiroHitP {
				peleaKill(g, tg.gu)
				s.dead = true
				g.say(tiroEnd[1])
				g.peleaNote("tiroteo_muerto")
				tiroEnd2(g, s)
				return true
			}
		}
		if len(s.shots) > 0 && g.time-s.shots[0].t0 > 1 {
			s.shots = s.shots[1:]
		}
		if age > s.length {
			tiroEnd2(g, s)
			return true
		}
	case tiroOver:
		return true
	}
	return false
}

// tiroEnd2: the ones still standing put the pistol away and run off.
func tiroEnd2(g *Game, s *tiroState) {
	r := partyDice()
	for _, m := range []*pevMem{s.a, s.b} {
		if m == nil || m.gu.out() {
			continue
		}
		if m.mine() {
			pevRelease(m, r)
			if s.phase == tiroShoot {
				pevScare(g, m.gu, r)
			}
		} else {
			pevForget(m)
		}
	}
	s.phase, s.shots = tiroOver, nil
}

// tiroBarrel: the end of the pistol (world): at chest height, to the side he faces.
func tiroBarrel(gu *Guest) Vec {
	v := dirVec(gu.dir)
	return Vec{gu.pos.X + v.X*9, gu.pos.Y - 17 + v.Y*3}
}

var (
	tiroGun   = color.RGBA{0x2a, 0x2a, 0x30, 255}
	tiroFlare = color.RGBA{0xff, 0xe8, 0x80, 255}
	tiroTrace = color.RGBA{0xff, 0xf4, 0xc0, 180}
)

// tiroItems: the pistols, the flash of each shot and its thin trace.
func tiroItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*tiroState)
	if s.phase == tiroOver {
		return items
	}
	plot := g.plotter(dst)
	px := g.pevPx()
	k := g.pevK()
	for _, m := range []*pevMem{s.a, s.b} {
		gu := m.gu
		if gu.out() || !m.mine() {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.35, func() {
			b := tiroBarrel(gu)
			x, y := g.pevXY(b)
			v := dirVec(gu.dir)
			w := k(4)
			if v.X < 0 {
				x -= w
			}
			plot(x, y, maxI(w, 2*px), 2*px, tiroGun)
			plot(x+int(v.X)*px, y+2*px, px, 2*px, tiroGun) // the grip
		}})
	}
	for _, sh := range s.shots {
		if g.time-sh.t0 > tiroFlash {
			continue
		}
		sh := sh
		items = append(items, drawItem{sh.from.Y + 0.4, func() {
			x0, y0 := g.pevXY(sh.from)
			x1, y1 := g.pevXY(sh.to)
			drawLinePix(plot, x0, y0, x1, y1, px, tiroTrace)
			plot(x0-px, y0-px, 3*px, 3*px, tiroFlare)
		}})
	}
	return items
}
