package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "murga" (idea 313): A STREET BAND. In the parties with a street (streetOK: the lanes of the police cars with
// ordinary traffic), now and then three or four guests pick up improvised instruments (a bucket for a bass drum, a pot
// and a spoon, a plastic flute, a toy trumpet, a kazoo) and march along the pavement by the street for 30-60 s, one
// behind the other on the leader's steps (pevTrail, like the conga), with notes floating over them and, when a player
// is near, a quiet synthetic "bom, bom, clink" every couple of bars (no music of its own: it never fights the party's
// song). Guests near them come over and follow dancing for a while (8-16 s each) and then go back to their business.
//
// They are plain guests (pevMem): they can be killed (the others react as with any murder and drop out; the band goes on
// while two of them play, the next one leads), witnesses, every cap and goal. Every phase has a time limit (gathering
// 14 s, the march at most 60 s, the frame's guard). 12 % of the parties with a street, once (MaxRuns 1), between 60 and
// 260 s; group "street" (never with the broken-down car or the taxi) and Big (never with the conga or the contest).
// Dice: partyDice(); g.rng is never moved. -pevent murga forces it (progress frozen).
// ---------------------------------------------------------------------------

const (
	murgaGap     = 14.0
	murgaSpeed   = 18.0
	murgaPull    = 40.0
	murgaGather  = 14.0
	murgaFolMax  = 5
	murgaMinLen  = 200.0
	murgaStreetD = 120.0 // the route keeps within this distance of the street
)

// The instruments (and the pose of the one who plays it).
const (
	instBucket = iota
	instPot
	instFlute
	instTrumpet
	instKazoo
	instN
)

var murgaLines = pevTexts("¡Que suene la murga!", "¡Pom, pom, chis!", "¡Todos detrás de la murga!", "¡Esa cacerola!", "¡Más ritmo!")
var murgaFollow = pevTexts("¡Yo me apunto!", "¡Qué escándalo!", "¡Esto es una fiesta!")
var murgaBye = pevTexts("¡Bravo, murga!", "¡Hasta la próxima esquina!", "¡Me duelen las orejas!")

func init() {
	registerPartyEvent(&partyEvent{ID: "murga", Name: "Murga callejera", Group: "street", Big: true,
		Prob: 0.12, AtMin: 60, AtMax: 260, Dur: 50, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return streetOK(partyDefOf(g, t)) && t.roam && t.guests >= 24 },
		Start:    murgaStart, Items: murgaItems,
		Update: pevGuard(murgaUpdate, func(g *Game, a *partyAct) { murgaEnd(a.D.(*murgaState), false) })})
}

type murgaState struct {
	band   []*pevMem // band[0] leads; m.idx = his instrument
	fol    []*pevMem // the ones who follow for a while; m.next = when they leave
	trail  pevTrail
	route  []Vec
	march  bool
	t0     float64
	stop   float64 // game time the march ends at the latest
	nextF  float64
	said   float64
	beat   float64
	joined int
}

func (s *murgaState) pevHas(gu *Guest) bool { return pevHasIn(s.band, gu) || pevHasIn(s.fol, gu) }

// murgaStreetDist: how far a point is from the street (the nearest lane), 1e9 if the level has none.
func (g *Game) murgaStreetDist(p Vec) float64 {
	if !streetOK(g.lvl.Def) {
		return 1e9
	}
	ys, _ := g.streetLanes()
	best := 1e9
	for _, y := range ys {
		best = math.Min(best, math.Abs(p.Y-y))
	}
	return best
}

func (g *Game) murgaPointOK(p Vec, street bool) bool {
	l := g.lvl
	if !l.Free(p) || l.SolidP(p.X, p.Y) || l.badGround(p) || l.InPool(p) || !l.Outdoor(p) {
		return false
	}
	return !street || g.murgaStreetDist(p) < murgaStreetD
}

// murgaRoute: a walk of at least murgaMinLen px along the street (or anywhere outdoors when the level has none).
func (g *Game) murgaRoute(r *rand.Rand) []Vec {
	street := streetOK(g.lvl.Def)
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < 50; i++ {
			a := g.lvl.NearestFree(g.lvl.RandomRoam(r))
			if !g.murgaPointOK(a, street) || !g.pevPlayerFar(a, 60) {
				continue
			}
			for j := 0; j < 12; j++ {
				b := g.lvl.NearestFree(g.lvl.RandomRoam(r))
				if math.Abs(b.X-a.X) < murgaMinLen || !g.murgaPointOK(b, street) {
					continue
				}
				path := g.lvl.PathG(a, b)
				if len(path) == 0 {
					continue
				}
				ok, total, at := true, 0.0, a
				for _, p := range path {
					total += dist(at, p)
					at = p
					if street && g.murgaStreetDist(p) > murgaStreetD+20 {
						ok = false
						break
					}
				}
				if ok && total < 2.2*dist(a, b) {
					return append([]Vec{a}, path...)
				}
			}
		}
		street = false // no way along the street now: anywhere outdoors (forced with -pevent in a level without one)
		if streetOK(g.lvl.Def) && dbgPartyEvent != "murga" {
			return nil
		}
	}
	return nil
}

func murgaStart(g *Game, a *partyAct) bool {
	r := partyDice()
	route := g.murgaRoute(r)
	if len(route) < 2 {
		return false
	}
	start := route[0]
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && dist(gu.pos, start) < 240 {
			cand = append(cand, gu)
		}
	}
	for i := 1; i < len(cand); i++ {
		for j := i; j > 0 && dist(cand[j].pos, start) < dist(cand[j-1].pos, start); j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	n := 3 + r.Intn(2)
	insts := r.Perm(instN)
	fwd := faceVec(route[0], route[1])
	s := &murgaState{route: route[1:], t0: g.time}
	for _, gu := range cand {
		if len(s.band) >= n {
			break
		}
		k := float64(len(s.band))
		pt := g.lvl.NearestFree(Vec{start.X - fwd.X*murgaGap*k, start.Y - fwd.Y*murgaGap*k})
		m := &pevMem{gu: gu, pt: pt, idx: insts[len(s.band)]}
		if g.pevSendTo(m, pt) {
			s.band = append(s.band, m)
		}
	}
	if len(s.band) < 3 {
		pevReleaseAll(s.band, r)
		return false
	}
	s.band[0].gu.icon, s.band[0].gu.iconT = murgaLines[0], 2.4
	a.D = s
	return true
}

func murgaEnd(s *murgaState, cheer bool) {
	r := partyDice()
	if cheer {
		k := 0
		for _, m := range s.fol {
			if m.mine() && k < 2 && m.gu.icon == "" {
				m.gu.icon, m.gu.iconT = pevPick(r, murgaBye), 2
				k++
			}
		}
	}
	pevReleaseAll(s.band, r)
	pevReleaseAll(s.fol, r)
	s.band, s.fol = nil, nil
}

func murgaAnim(inst int) int {
	if inst == instBucket || inst == instPot {
		return AnimDrum
	}
	return AnimSing
}

func murgaUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*murgaState)
	r := partyDice()
	if !s.march {
		// gathering: they walk to their places; whoever is not there in time is let go
		kept := s.band[:0]
		ready := 0
		for _, m := range s.band {
			if !m.mine() {
				pevForget(m)
				continue
			}
			kept = append(kept, m)
			if m.gu.state == gIdle && m.gu.timer > 1000 && dist(m.gu.pos, m.pt) < 6 {
				ready++
			}
		}
		s.band = kept
		over := g.time-s.t0 > murgaGather
		if len(s.band) < 3 || over && ready < 3 { // a band is three or four
			murgaEnd(s, false)
			return true
		}
		if ready < len(s.band) && !over {
			return false
		}
		kept = s.band[:0]
		for _, m := range s.band {
			if m.gu.state == gIdle && m.gu.timer > 1000 && dist(m.gu.pos, m.pt) < 6 {
				m.pose(murgaAnim(m.idx))
				kept = append(kept, m)
			} else {
				pevRelease(m, r)
			}
		}
		s.band = kept
		lead := s.band[0].gu
		s.trail.add(lead.pos)
		s.trail.prefix(lead.pos, faceVec(lead.pos, s.route[0]), func(p Vec) bool { return g.lvl.Free(p) && !g.lvl.SolidP(p.X, p.Y) })
		s.march, s.stop, s.nextF, s.beat = true, g.time+30+r.Float64()*30, g.time+2, g.time
		lead.icon, lead.iconT = pevPick(r, murgaLines[1:]), 2
		return false
	}
	// the band: the ones who are not ours any more drop out; the next one leads
	kept := s.band[:0]
	for _, m := range s.band {
		if m.mine() {
			kept = append(kept, m)
		} else {
			pevForget(m)
		}
	}
	lostLead := len(kept) > 0 && len(s.band) > 0 && kept[0] != s.band[0]
	s.band = kept
	if len(s.band) < 2 {
		murgaEnd(s, false)
		return true
	}
	lead := s.band[0].gu
	if lostLead {
		if len(s.route) == 0 {
			murgaEnd(s, true)
			return true
		}
		s.route = g.lvl.PathG(lead.pos, s.route[len(s.route)-1])
		if len(s.route) == 0 {
			murgaEnd(s, false)
			return true
		}
	}
	if _, arr := follow(&lead.pos, &lead.dir, &s.route, murgaSpeed, dt); arr || g.time >= s.stop {
		murgaEnd(s, true)
		return true
	}
	s.trail.add(lead.pos)
	for i, m := range s.band {
		m.keep()
		m.gu.dAnim = murgaAnim(m.idx)
		if i == 0 {
			continue
		}
		slot := s.trail.at(float64(i) * murgaGap)
		gu := m.gu
		if d := dist(gu.pos, slot); d > 0.01 {
			gu.dir = faceTo(gu.pos, slot)
			if v := murgaPull * dt; d > v && d > 3 {
				gu.pos = Vec{gu.pos.X + (slot.X-gu.pos.X)/d*v, gu.pos.Y + (slot.Y-gu.pos.Y)/d*v}
				continue
			}
		}
		gu.pos = slot
	}
	// followers: they come dancing behind for a while and go
	keptF := s.fol[:0]
	for _, m := range s.fol {
		switch {
		case !m.mine():
			pevForget(m)
		case g.time >= m.next:
			if m.gu.icon == "" && r.Float64() < 0.4 {
				m.gu.icon, m.gu.iconT = pevPick(r, murgaBye), 1.8
			}
			pevRelease(m, r)
		default:
			keptF = append(keptF, m)
		}
	}
	s.fol = keptF
	for j, m := range s.fol {
		m.keep()
		slot := s.trail.at(float64(len(s.band)+j) * murgaGap)
		gu := m.gu
		d := dist(gu.pos, slot)
		if d > 0.01 {
			gu.dir = faceTo(gu.pos, slot)
		}
		if v := murgaPull * dt; d > v {
			gu.pos = Vec{gu.pos.X + (slot.X-gu.pos.X)/d*v, gu.pos.Y + (slot.Y-gu.pos.Y)/d*v}
		} else {
			gu.pos = slot
		}
	}
	if g.time >= s.nextF {
		s.nextF = g.time + 1.5 + r.Float64()*1.5
		if len(s.fol) < murgaFolMax && s.trail.length() > float64(len(s.band)+len(s.fol)+1)*murgaGap {
			for _, gu := range g.guests {
				if !g.pevFreeGuest(gu) || s.pevHas(gu) || r.Float64() > 0.5 {
					continue
				}
				near := false
				for _, b := range s.band {
					if dist(b.gu.pos, gu.pos) < 90 {
						near = true
					}
				}
				if !near || !g.lvl.clearLine(gu.pos, s.trail.at(float64(len(s.band)+len(s.fol))*murgaGap)) {
					continue
				}
				m := &pevMem{gu: gu, next: g.time + 8 + r.Float64()*8}
				m.hold(gDance)
				s.fol = append(s.fol, m)
				s.joined++
				if gu.icon == "" {
					gu.icon, gu.iconT = pevPick(r, murgaFollow), 1.6
				}
				break
			}
		}
	}
	// the leader shouts now and then; the drum is heard when a player is near (quiet, every two bars)
	if g.time-s.said > 6 && lead.icon == "" {
		s.said = g.time
		lead.icon, lead.iconT = pevPick(r, murgaLines[1:]), 2
	}
	if g.time >= s.beat {
		s.beat = g.time + 2.4
		if !g.pevPlayerFar(lead.pos, 220) {
			g.snd.toroPlay("murga")
		}
	}
	return false
}

// ---- drawing ------------------------------------------------------------------------------------------------

var (
	murgaBucket  = color.RGBA{0x3a, 0x7a, 0xd8, 255}
	murgaBucketD = color.RGBA{0x24, 0x4c, 0x8c, 255}
	murgaPink    = color.RGBA{0xf0, 0x7a, 0xb8, 255}
	murgaYellow  = color.RGBA{0xf4, 0xc8, 0x30, 255}
	murgaRed     = color.RGBA{0xd8, 0x34, 0x34, 255}
)

// drawInstrumentPix: the improvised instrument of a player whose feet are at (cx, cy), facing side (+1 right, -1 left);
// t in seconds makes the sticks and the spoon hit.
func drawInstrumentPix(plot plotFn, inst, cx, cy, px, side int, k func(float64) int, t float64) {
	hit := int(t*4)%2 == 0
	sx := func(dx int) int { // a horizontal offset turned to the side he faces
		if side < 0 {
			return cx - dx*px - px
		}
		return cx + dx*px
	}
	switch inst {
	case instBucket: // a bucket upside down at the belly, a stick that beats it
		y := cy - k(19)
		plot(cx-4*px, y, 8*px, 6*px, murgaBucket)
		plot(cx-4*px, y, 8*px, px, murgaBucketD)
		plot(cx-3*px, y+5*px, 6*px, px, murgaBucketD)
		sy := y - 3*px
		if hit {
			sy = y - px
		}
		plot(sx(3), sy, px, 3*px, pevBrown)
	case instPot: // a steel pot with its handle, and the spoon
		y := cy - k(21)
		plot(sx(-2), y, 6*px, 3*px, pevSteel)
		plot(sx(-2), y+2*px, 6*px, px, pevSteelD)
		plot(sx(-4), y+px, 2*px, px, pevInk)
		sy := y - 3*px
		if hit {
			sy = y - 2*px
		}
		plot(sx(4), sy, px, 3*px, pevSteelD)
		plot(sx(4), sy-px, 2*px, px, pevSteel)
	case instFlute: // a pink plastic flute from the mouth down and out
		y := cy - k(29)
		for i := 0; i < 6; i++ {
			plot(sx(2+i), y+i*px/2, px, px, murgaPink)
		}
	case instTrumpet: // a yellow toy trumpet with its bell
		y := cy - k(29)
		plot(sx(2), y, 5*px, px, murgaYellow)
		plot(sx(7), y-px, 2*px, 3*px, murgaYellow)
	case instKazoo: // a little red kazoo at the mouth
		plot(sx(2), cy-k(29), 3*px, px, murgaRed)
	}
}

func murgaItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*murgaState)
	if !ok || !s.march {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for i, m := range s.band {
		m := m
		if !m.mine() {
			continue
		}
		gu := m.gu
		items = append(items, drawItem{gu.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(gu.pos)
			side := 1
			if gu.dir == DirLeft {
				side = -1
			}
			drawInstrumentPix(plot, m.idx, cx, cy, px, side, k, g.time)
		}})
		if i == 0 {
			items = append(items, drawItem{gu.pos.Y + 0.3, func() {
				cx, cy := g.pevXY(gu.pos)
				drawNotesPix(plot, cx, cy-k(46), px, g.time, 4)
			}})
		}
	}
	return items
}
