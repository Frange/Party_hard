package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "waiters": one or two waiters (white shirt, black waistcoat, red bow tie) walk the party with a silver tray of
// glasses and canapes (3-5 things on it, visible). They stop beside a guest, offer, and up to two free guests walk up,
// take something and drink it with the usual drinking gesture (game's own state, so the drunk counter goes up as with
// any drink). When the tray is empty the waiter goes back to the bar to restock. Sometimes (35 % of the waiters) he
// trips on the way: everything falls, glass breaks on the floor, the ones close by get a fright and run a moment (nobody
// is hurt, no body). The PLAYER can take something too with E when he is beside a waiter: a small bonus, once.
// The waiters are actors of their own (not guests): they never count for the level and cannot be killed.
//
// WHEN: parties with a bar or a lounge feel (waitersIcons), 25 % of them, between 30 and 200 s, once.
// ---------------------------------------------------------------------------

var waitersIcons = map[string]bool{"house": true, "casino": true, "cake": true, "neon": true, "disco": true, "disco70": true}

const (
	wtrBonus     = 25   // points for the player's first glass
	wtrTripP     = 0.35 // chance that a waiter trips on this run
	wtrReach     = 28.0
	wtrOffer     = 4.0
	wtrTakeRange = 26.0
)

const (
	wLoad = iota
	wGo
	wOffer
	wBack
	wFall
	wGetUp
	wLeave
	wDone
)

var waitersLines = pevTexts("¿Una copita?", "¡Canapés recién hechos!", "¡Copas frescas!", "Con permiso...", "¡Cuidado, que quema!",
	"¡Uy, uy, uy!", "Se me ha caído todo...", "¡Mi bandeja!", "¡Qué desastre!", "¡Salud!", "¡Una para mí!", "Gracias, majo",
	"Esto es lo mejor de la fiesta", "¡Ñam!", "¡AAH! ¡Cristales!", "¡Cuidado con el suelo!")

var _ = pevTexts("¡Uy!", "¡Madre mía!")

type waiter struct {
	pf        *pevPerf
	phase     int
	t0        float64
	tray      []int
	base      Vec
	stops     int
	tripAfter float64 // px of walking before he trips (0 = he does not)
	walked    float64
	last      Vec
	hintAt    float64
	retry     float64
}

type wShard struct {
	pos  Vec
	n    int
	seed int
	t0   float64
}

type waitersState struct {
	w      []*waiter
	take   []*pevMem
	shards []wShard
	given  bool
	t0     float64
	late   float64 // when the second waiter comes in (0 = none)
}

func (s *waitersState) pevHas(gu *Guest) bool { return pevHasIn(s.take, gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "waiters", Name: "Camareros con bandeja", Group: "service",
		Prob: 0.25, AtMin: 30, AtMax: 200, Dur: 80, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool {
			return t.roam && t.guests >= 20 && (t.bar || waitersIcons[t.icon])
		},
		Start: waitersStart, Items: waitersItems,
		Update: pevGuard(waitersUpdate, func(g *Game, a *partyAct) { waitersEnd(g, a.D.(*waitersState)) })})
}

func waiterSpec() CharSpec {
	return CharSpec{
		Skin: hx(0xe8b890), Hair: hx(0x2a1c14), Shirt: hx(0xf4f4f8), Shirt2: hx(0xf4f4f8), Pants: hx(0x14141a), Shoes: hx(0x14141a),
		Top: TopLong, Beard: BeardMustache, Accent: hx(0x14141a), noOverride: true,
	}
}

func waiterFill(r interface{ Intn(int) int }) []int {
	n := 3 + r.Intn(3)
	out := make([]int, n)
	for i := range out {
		out[i] = trayGlass + r.Intn(4)
	}
	return out
}

// waitersBase: where the waiters come from: a place in front of the bar (or a quiet roaming spot).
func waitersBase(g *Game, r interface{ Intn(int) int }) (Vec, bool) {
	pois := g.lvl.POIs["drink"]
	for i := 0; i < 12 && len(pois) > 0; i++ {
		p := g.lvl.NearestFree(pois[r.Intn(len(pois))])
		if g.lvl.Free(p) && !g.lvl.InPool(p) && !g.lvl.badGround(p) {
			return p, true
		}
	}
	rr := partyDice()
	p, ok := g.pevSpot(rr, 14, false)
	return p, ok
}

func waitersStart(g *Game, a *partyAct) bool {
	r := partyDice()
	base, ok := waitersBase(g, r)
	if !ok {
		return false
	}
	pevFxPurge(g)
	st := &waitersState{t0: g.time}
	st.w = append(st.w, newWaiter(g, base, r.Float64() < wtrTripP, r.Float64()))
	if r.Float64() < 0.4 {
		st.late = g.time + 7
	}
	a.D = st
	return true
}

func newWaiter(g *Game, base Vec, trips bool, r float64) *waiter {
	w := &waiter{pf: newPevPerf(waiterSpec(), base, DirDown), phase: wLoad, t0: g.time, base: base, last: base}
	if trips {
		w.tripAfter = 70 + r*240
	}
	return w
}

func (w *waiter) pop() int {
	n := len(w.tray)
	if n == 0 {
		return trayEmpty
	}
	it := w.tray[n-1]
	w.tray = w.tray[:n-1]
	return it
}

// waiterPickStop: a guest who is free and a place beside him where the waiter can stand.
func waiterPickStop(g *Game, w *waiter) ([]Vec, bool) {
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if d := dist(gu.pos, w.pf.pos); d > 36 && d < 280 && g.pevFreeGuest(gu) {
			cand = append(cand, gu)
		}
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	for i, gu := range cand {
		if i >= 8 {
			break
		}
		to := faceVec(gu.pos, w.pf.pos)
		stop := g.lvl.NearestFree(Vec{gu.pos.X + to.X*20, gu.pos.Y + to.Y*14})
		if dist(stop, gu.pos) < 10 {
			continue
		}
		if path := g.lvl.PathG(w.pf.pos, stop); len(path) > 0 {
			return path, true
		}
	}
	return nil, false
}

func waiterStartFall(g *Game, st *waitersState, w *waiter) {
	r := partyDice()
	w.phase, w.t0, w.tripAfter = wFall, g.time, 0
	w.tray = nil
	w.pf.path, w.pf.moving = nil, false
	for i := 0; i < 3; i++ {
		st.shards = append(st.shards, wShard{pos: Vec{w.pf.pos.X + (r.Float64()-0.5)*26, w.pf.pos.Y + (r.Float64()-0.2)*10}, n: 3 + r.Intn(3), seed: r.Intn(97), t0: g.time})
	}
	g.snd.Play("thud")
	g.bubble(w.pf, bubGuest, w.pf.pos, pevPick(r, waitersLines[5:8]), false, 0)
	n := 0
	for _, gu := range g.guests {
		if n >= 4 {
			break
		}
		if dist(gu.pos, w.pf.pos) < 70 && g.pevFreeGuest(gu) {
			gu.state, gu.timer, gu.path, gu.moving = gScared, 1.6+r.Float64(), nil, false
			gu.icon, gu.iconT = pevPick(r, []string{waitersLines[14], "¡Uy!", "¡Madre mía!"}), 1.8
			n++
		}
	}
}

// waiterServe: a guest who reached the waiter takes his glass and drinks.
func waiterServe(g *Game, w *waiter, m *pevMem) {
	r := partyDice()
	gu := m.gu
	if w.pop() == trayEmpty {
		pevRelease(m, r)
		return
	}
	gu.state, gu.timer, gu.dur, gu.path, gu.moving = gDrink, 1.9, 1.9, nil, false
	gu.dir = faceTo(gu.pos, w.pf.pos)
	if r.Float64() < 0.5 {
		gu.icon, gu.iconT = pevPick(r, waitersLines[9:14]), 1.6
	}
}

func waiterInvite(g *Game, st *waitersState, w *waiter) {
	r := partyDice()
	free := 2 - len(st.take)
	for _, gu := range g.guests {
		if free <= 0 || len(w.tray) <= len(st.take) {
			break
		}
		if dist(gu.pos, w.pf.pos) > 100 || !g.pevFreeGuest(gu) {
			continue
		}
		side := 1.0
		if len(st.take)%2 == 1 {
			side = -1
		}
		pt := g.lvl.NearestFree(Vec{w.pf.pos.X + side*(12+r.Float64()*4), w.pf.pos.Y + 5})
		m := &pevMem{gu: gu, pt: pt, next: g.time + 8}
		if g.pevSendTo(m, pt) {
			st.take = append(st.take, m)
			free--
		}
	}
}

func waitersUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*waitersState)
	r := partyDice()
	age := g.time - st.t0
	closing := age > a.ev.Dur-14
	if st.late > 0 && g.time >= st.late && !closing {
		st.late = 0
		if base, ok := waitersBase(g, r); ok {
			st.w = append(st.w, newWaiter(g, base, r.Float64() < wtrTripP, r.Float64()))
		}
	}
	// the guests who walk up to a waiter
	st.take = pevPruneMems(st.take)
	for i := 0; i < len(st.take); i++ {
		m := st.take[i]
		var near *waiter
		for _, w := range st.w {
			if (w.phase == wOffer || w.phase == wGo) && dist(m.gu.pos, w.pf.pos) < wtrTakeRange {
				near = w
			}
		}
		if m.gu.state == gIdle && m.gu.timer > 1000 && near != nil { // arrived beside him
			waiterServe(g, near, m)
			st.take = append(st.take[:i], st.take[i+1:]...)
			i--
		} else if g.time > m.next { // never got there
			pevRelease(m, r)
			st.take = append(st.take[:i], st.take[i+1:]...)
			i--
		}
	}
	for _, m := range st.take {
		m.keep()
	}
	// shards fade after a while
	keep := st.shards[:0]
	for _, s := range st.shards {
		if g.time-s.t0 < 9 {
			keep = append(keep, s)
		}
	}
	st.shards = keep

	alive := 0
	for _, w := range st.w {
		if w.phase == wDone {
			continue
		}
		alive++
		waiterStep(g, st, w, dt, closing)
	}
	if st.late == 0 && alive == 0 {
		waitersEnd(g, st)
		return true
	}
	return false
}

func waiterStep(g *Game, st *waitersState, w *waiter, dt float64, closing bool) {
	r := partyDice()
	age := g.time - w.t0
	pf := w.pf
	moveTracked := func() bool {
		arr := pf.step(dt)
		w.walked += dist(w.last, pf.pos)
		w.last = pf.pos
		return arr
	}
	if closing && (w.phase == wLoad || w.phase == wGo || w.phase == wOffer) {
		w.phase, w.t0 = wLeave, g.time
		pf.path = g.lvl.PathG(pf.pos, w.base)
		if pf.path == nil {
			w.phase = wDone
		}
		return
	}
	switch w.phase {
	case wLoad:
		pf.moving = false
		if age >= 2.2 {
			w.tray, w.stops = waiterFill(r), 0
			w.phase, w.t0, w.retry = wGo, g.time, 0
		}
	case wGo:
		if len(pf.path) == 0 && !pf.moving {
			if g.time < w.retry {
				return
			}
			p, ok := waiterPickStop(g, w)
			if !ok {
				w.retry = g.time + 3
				return
			}
			pf.path = p
		}
		if w.tripAfter > 0 && w.walked >= w.tripAfter {
			waiterStartFall(g, st, w)
			return
		}
		if moveTracked() {
			w.phase, w.t0 = wOffer, g.time
			pf.moving = false
			if gu := nearestFreeGuest(g, pf.pos, 120); gu != nil {
				pf.dir = faceTo(pf.pos, gu.pos)
			}
			g.bubble(pf, bubGuest, pf.pos, pevPick(r, waitersLines[1:5]), false, 0)
			waiterInvite(g, st, w)
		}
	case wOffer:
		pf.moving = false
		if age > 1.4 && len(st.take) < 2 && age < wtrOffer-1 && len(w.tray) > len(st.take) && r.Float64() < dt*0.8 {
			waiterInvite(g, st, w)
		}
		if age >= wtrOffer || (len(w.tray) == 0 && len(st.take) == 0) {
			w.stops++
			if len(w.tray) == 0 || w.stops >= 4 {
				w.phase, w.t0 = wBack, g.time
				pf.path = g.lvl.PathG(pf.pos, w.base)
				if pf.path == nil {
					w.phase, w.t0 = wLoad, g.time
				}
			} else {
				w.phase, w.t0 = wGo, g.time
				pf.path = nil
			}
		}
	case wBack:
		if moveTracked() || age > 30 {
			w.phase, w.t0 = wLoad, g.time
			pf.moving = false
			pf.dir = DirDown
		}
	case wFall:
		if age >= 2.8 {
			w.phase, w.t0 = wGetUp, g.time
			g.bubble(pf, bubGuest, pf.pos, waitersLines[6], false, 0)
		}
	case wGetUp:
		if age >= 1.2 {
			w.phase, w.t0 = wBack, g.time
			pf.path = g.lvl.PathG(pf.pos, w.base)
			if pf.path == nil {
				w.phase = wLoad
			}
		}
	case wLeave:
		if moveTracked() || age > 30 {
			w.phase = wDone
		}
	}
	// a player close to a waiter with something on the tray is offered a glass
	if (w.phase == wOffer || w.phase == wGo) && len(w.tray) > 0 && g.time >= w.hintAt {
		for _, pl := range g.players() {
			if pl != nil && !pl.dead && dist(pl.pos, pf.pos) < 44 {
				w.hintAt = g.time + 6
				pf.dir = faceTo(pf.pos, pl.pos)
				g.bubble(pf, bubGuest, pf.pos, waitersLines[0], false, 0)
				break
			}
		}
	}
}

func nearestFreeGuest(g *Game, p Vec, maxD float64) *Guest {
	var best *Guest
	bd := maxD
	for _, gu := range g.guests {
		if d := dist(gu.pos, p); d < bd && !gu.out() {
			best, bd = gu, d
		}
	}
	return best
}

// pevUse: E beside a waiter with something on the tray: the player takes a glass (small bonus, once per party).
func (s *waitersState) pevUse(g *Game, p *Player) bool {
	for _, w := range s.w {
		if (w.phase != wOffer && w.phase != wGo && w.phase != wBack) || len(w.tray) == 0 || dist(p.pos, w.pf.pos) > wtrReach {
			continue
		}
		w.pop()
		w.pf.dir = faceTo(w.pf.pos, p.pos)
		g.snd.Play("glug")
		p.msg, p.msgT = waitersLines[9], 1.3
		if !s.given {
			s.given = true
			g.award(p.pos, wtrBonus, "")
		}
		return true
	}
	return false
}

func waitersEnd(g *Game, st *waitersState) {
	pevReleaseAll(st.take, partyDice())
	st.take = nil
}

func waitersItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*waitersState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, s := range st.shards {
		s := s
		items = append(items, drawItem{s.pos.Y - 3, func() {
			x, y := g.pevXY(s.pos)
			drawShardsPix(plot, x, y, px, s.n, s.seed)
		}})
	}
	for _, w := range st.w {
		w := w
		if w.phase == wDone {
			continue
		}
		pf := w.pf
		items = append(items, drawItem{pf.pos.Y, func() {
			cx, cy := g.pevXY(pf.pos)
			if w.phase == wFall || w.phase == wGetUp {
				if w.phase == wFall {
					g.drawLying(dst, pf.cs, true, pf.pos)
					drawStarsPix(plot, cx, cy-k(10), px, g.time)
				} else {
					pf.draw(g, dst, AnimTired, g.time*2)
				}
				return
			}
			pf.draw(g, dst, AnimIdle, 0)
			if pf.dir != DirUp {
				drawVestPix(plot, cx, cy, px, k)
				drawBowTiePix(plot, cx, cy-k(26), px)
			}
			if w.phase != wLoad && w.phase != wDone {
				side := 1
				if pf.dir == DirLeft {
					side = -1
				}
				drawTrayPix(plot, cx+side*k(7), cy-k(15), px, w.tray)
			}
		}})
	}
	return items
}
