package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "chairs": musical chairs on the dance floor. 4-6 guests walk around a ring of chairs (one chair fewer than
// players) while the music plays (notes over the ring); at the "¡STOP!" they run to sit down and one is left
// standing and complains. Two or three rounds (a chair is taken away every round) and the ones left sitting
// celebrate. The chairs are drawn by code (back behind the guest, seat in front, so he looks seated) and vanish at
// the end. Nobody is hurt, nothing counts for points.
//
// WHEN: parties with a dance floor of the kind a game fits (chairsIcons: home, club, birthday, ranch, beach, 70s disco),
// 18 % of them, between 80 and 240 s, once. Group "dance" and Big (never together with the conga or the contest).
// ---------------------------------------------------------------------------

// chairsIcons: a game for a party at home, a fair, a ranch or the beach.
var chairsIcons = map[string]bool{"house": true, "neon": true, "cake": true, "ranch": true, "beach": true, "disco70": true}

const (
	chRingSpeed = 0.85 // rad/s around the ring
	chRush      = 54.0 // px/s to the chair
	chSquash    = 0.78 // the ring is an ellipse seen from above
)

const (
	chGather = iota
	chMusic
	chRushP
	chPause
	chWin
)

var chairsLines = pevTexts("¡Juego de las sillas en la pista!", "¡STOP!", "¡Había una silla menos!", "¡Eso es trampa!", "¡Me han empujado!",
	"¡Esa silla era mía!", "¡Yo llegué primero!", "¡No vale, me ataba el zapato!", "¡Mía!", "¡Aquí!", "¡Uf, por poco!", "¡Soy el rey de las sillas!")

var _ = pevTexts("¡HE GANADO!")

type chairsState struct {
	center     Vec
	R, Rc      float64
	mem        []*pevMem
	slots      []Vec
	angs       []float64
	alive      []bool
	phase      int
	t0         float64
	musicLen   float64
	round      int
	rounds     int
	loser      *pevMem
	stopped    bool
	flashUntil float64
}

func (c *chairsState) pevHas(gu *Guest) bool { return pevHasIn(c.mem, gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "chairs", Name: "Juego de las sillas", Group: "dance", Big: true,
		Prob: 0.18, AtMin: 80, AtMax: 240, Dur: 50, MaxRuns: 1, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool {
			return t.dance && t.roam && t.guests >= 20 && chairsIcons[t.icon]
		},
		Start: chairsStart, Items: chairsItems,
		Update: pevGuard(chairsUpdate, func(g *Game, a *partyAct) { chairsFinish(g, a.D.(*chairsState)) })})
}

func (c *chairsState) ring(a, rad float64) Vec {
	return Vec{c.center.X + math.Cos(a)*rad, c.center.Y + math.Sin(a)*rad*chSquash}
}

func (c *chairsState) aliveN() int {
	n := 0
	for _, a := range c.alive {
		if a {
			n++
		}
	}
	return n
}

func chairsStart(g *Game, a *partyAct) bool {
	r := partyDice()
	center := congaFloor(g.lvl.Def)
	var cand []*Guest
	for _, gu := range g.guests {
		if congaCanJoin(g, gu) && dist(gu.pos, center) < 150 && g.lvl.PathG(gu.pos, center) != nil {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 5 {
		return false
	}
	st := &chairsState{center: center}
	// the biggest ring that fits (every point of the ring, the chairs and the middle must be walkable)
	ok := false
	for _, rad := range []float64{30, 26, 22} {
		st.R, st.Rc = rad, rad-13
		fine := g.pevClear(center, 0.1) && g.pevClear(center, st.Rc*0.5)
		for i := 0; i < 24 && fine; i++ {
			an := float64(i) / 24 * 2 * math.Pi
			for _, rr := range []float64{st.R, st.Rc} {
				p := st.ring(an, rr)
				if !g.lvl.Free(p) || g.lvl.SolidP(p.X, p.Y) || g.lvl.badGround(p) {
					fine = false
				}
			}
		}
		if fine {
			ok = true
			break
		}
	}
	if !ok {
		return false
	}
	n := 4 + r.Intn(3)
	if n > len(cand) {
		n = len(cand)
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	for _, gu := range cand {
		if len(st.mem) >= n {
			break
		}
		an := float64(len(st.mem)) / float64(n) * 2 * math.Pi
		m := &pevMem{gu: gu, ang: an}
		if g.pevSendTo(m, st.ring(an, st.R)) {
			st.mem = append(st.mem, m)
		}
	}
	if len(st.mem) < 4 {
		pevReleaseAll(st.mem, r)
		return false
	}
	// one chair fewer than players, evenly spread; every round one more is taken away
	k := len(st.mem) - 1
	off := r.Float64() * 2 * math.Pi
	for i := 0; i < k; i++ {
		an := off + float64(i)/float64(k)*2*math.Pi
		st.angs = append(st.angs, an)
		st.slots = append(st.slots, st.ring(an, st.Rc))
		st.alive = append(st.alive, true)
	}
	st.rounds = minI(3, len(st.mem)-2)
	st.t0 = g.time
	st.phase = chGather
	g.say(chairsLines[0])
	a.D = st
	return true
}

func chairsFinish(g *Game, st *chairsState) {
	r := partyDice()
	pevReleaseAll(st.mem, r)
	st.mem = nil
	st.alive = nil
}

func chairsUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*chairsState)
	r := partyDice()
	age := g.time - st.t0
	// those who are not ours any more (a fright, a murder) leave the game
	st.mem = pevPruneMems(st.mem)
	if st.loser != nil && !st.loser.mine() {
		st.loser = nil
	}
	if len(st.mem) < 2 {
		chairsFinish(g, st)
		return true
	}
	for _, m := range st.mem {
		m.keep()
	}
	switch st.phase {
	case chGather:
		all := true
		for _, m := range st.mem {
			gu := m.gu
			if gu.state == gIdle && gu.timer > 1000 {
				m.want = gIdle
				gu.dir = faceTo(gu.pos, st.center)
			} else {
				all = false
			}
		}
		if all || age > 22 {
			// the late ones are out of the game
			keep := st.mem[:0]
			for _, m := range st.mem {
				if m.gu.state == gIdle && m.gu.timer > 1000 {
					keep = append(keep, m)
				} else {
					pevRelease(m, r)
				}
			}
			st.mem = keep
			if len(st.mem) < 4 {
				chairsFinish(g, st)
				return true
			}
			k := len(st.mem) - 1
			for len(st.alive) > k { // fewer players: fewer chairs
				st.alive[len(st.alive)-1] = false
				st.alive = st.alive[:len(st.alive)-1]
				st.slots, st.angs = st.slots[:len(st.slots)-1], st.angs[:len(st.angs)-1]
			}
			st.rounds = minI(3, len(st.mem)-2)
			chairsMusic(g, st, r)
		}
	case chMusic:
		for _, m := range st.mem {
			old := m.gu.pos
			// a guest that was sitting glides back to the ring
			m.ang += chRingSpeed * dt
			tgt := st.ring(m.ang, st.R)
			if k := math.Min(1, age/0.9); k < 1 {
				m.gu.pos = Vec{m.pt.X + (tgt.X-m.pt.X)*k, m.pt.Y + (tgt.Y-m.pt.Y)*k}
			} else {
				m.gu.pos = tgt
			}
			m.gu.dir = faceTo(old, m.gu.pos)
			if m.want != gDance {
				m.hold(gDance)
			}
		}
		if age >= st.musicLen {
			chairsStop(g, st, r)
		}
	case chRushP:
		seated := 0
		for _, m := range st.mem {
			gu := m.gu
			if m.idx >= 0 && m.idx < 100 { // has a chair: runs to it
				if m.want == gFun {
					seated++
					continue
				}
			}
			d := dist(gu.pos, m.pt)
			v := chRush * dt
			if d <= v {
				gu.pos = m.pt
				if m == st.loser {
					gu.dir = faceTo(gu.pos, st.center)
					continue
				}
				m.hold(gFun)
				gu.fun, gu.dir = 5, DirDown
				gu.pos.Y += 2
				seated++
				if r.Float64() < 0.6 {
					gu.icon, gu.iconT = pevPick(r, chairsLines[8:11]), 1.4
				}
				continue
			}
			gu.pos = Vec{gu.pos.X + (m.pt.X-gu.pos.X)/d*v, gu.pos.Y + (m.pt.Y-gu.pos.Y)/d*v}
			gu.dir = faceTo(gu.pos, m.pt)
		}
		if seated >= len(st.mem)-1 || age > 5 {
			st.phase, st.t0 = chPause, g.time
			if st.loser != nil {
				l := st.loser.gu
				l.icon, l.iconT = pevPick(r, chairsLines[2:8]), 2.6
				l.dir = DirDown
			}
		}
	case chPause:
		for _, m := range st.mem {
			if m.want == gFun {
				m.gu.fun = 5
			}
		}
		if age >= 2.6 {
			if st.loser != nil { // out of the game: he walks away grumbling
				pevRelease(st.loser, r)
				for i, m := range st.mem {
					if m == st.loser {
						st.mem = append(st.mem[:i], st.mem[i+1:]...)
						break
					}
				}
				st.loser = nil
			}
			st.round++
			if st.round >= st.rounds || len(st.mem) < 2 {
				st.phase, st.t0 = chWin, g.time
				for i, m := range st.mem {
					if i < 2 {
						m.gu.icon, m.gu.iconT = pevPick(r, []string{"¡HE GANADO!", chairsLines[11]}), 3
					}
				}
				g.snd.Play("clap")
			} else {
				// a chair less
				var idx []int
				for i, al := range st.alive {
					if al {
						idx = append(idx, i)
					}
				}
				if len(idx) > len(st.mem)-1 {
					st.alive[idx[r.Intn(len(idx))]] = false
				}
				for _, m := range st.mem { // each one starts from where he sits
					m.pt = m.gu.pos
					m.ang = math.Atan2((m.gu.pos.Y-st.center.Y)/chSquash, m.gu.pos.X-st.center.X)
					m.idx = -1
				}
				chairsMusic(g, st, r)
			}
		}
	case chWin:
		for _, m := range st.mem {
			if m.want == gFun {
				m.gu.fun = 5
			}
		}
		if age >= 3.4 {
			chairsFinish(g, st)
			return true
		}
	}
	return false
}

func chairsMusic(g *Game, st *chairsState, r interface{ Float64() float64 }) {
	st.phase, st.t0, st.stopped = chMusic, g.time, false
	st.musicLen = 5 + r.Float64()*2.5
	for _, m := range st.mem {
		m.idx = -1
		if m.want != gDance {
			m.pt = m.gu.pos
			m.hold(gDance)
		}
	}
}

// chairsStop: the music stops; everybody runs to a chair, one is left without.
func chairsStop(g *Game, st *chairsState, r interface{ Intn(int) int }) {
	st.phase, st.t0 = chRushP, g.time
	g.floats = append(g.floats, floatText{pos: Vec{st.center.X, st.center.Y - 18}, txt: chairsLines[1], t: 1.6, col: rgb(0xff5a4a)})
	g.snd.Play("alert")
	var free []int
	for i, al := range st.alive {
		if al {
			free = append(free, i)
		}
	}
	li := r.Intn(len(st.mem))
	st.loser = st.mem[li]
	taken := map[int]bool{}
	// each chair goes to the nearest player still without one (the loser is left out)
	got := map[*pevMem]bool{}
	for _, ci := range free {
		var best *pevMem
		for _, m := range st.mem {
			if m == st.loser || got[m] {
				continue
			}
			if best == nil || dist(m.gu.pos, st.slots[ci]) < dist(best.gu.pos, st.slots[ci]) {
				best = m
			}
		}
		if best != nil {
			got[best] = true
			taken[ci] = true
			best.pt, best.idx = st.slots[ci], ci
		}
	}
	// the loser runs to the nearest chair too and finds it taken
	l := st.loser
	best := free[0]
	for _, ci := range free {
		if dist(l.gu.pos, st.slots[ci]) < dist(l.gu.pos, st.slots[best]) {
			best = ci
		}
	}
	l.pt, l.idx = Vec{st.slots[best].X, st.slots[best].Y + 9}, 100
}

// drawChairPix: part 0 = the back and the rear legs (behind the sitter), part 1 = the seat and the front legs (in front,
// they hide his thighs). (cx, cy) are the feet of the chair.
func drawChairPix(plot plotFn, cx, cy, px, part int) {
	wood, dark, lite := color.RGBA{0xa8, 0x72, 0x3a, 255}, color.RGBA{0x4e, 0x30, 0x16, 255}, color.RGBA{0xd0, 0x94, 0x52, 255}
	if part == 0 {
		plot(cx-5*px, cy-17*px, 11*px, 11*px, dark)
		plot(cx-4*px, cy-16*px, 9*px, 9*px, wood)
		plot(cx-3*px, cy-15*px, 7*px, px, lite)
		plot(cx-4*px, cy-5*px, px, 5*px, dark)
		plot(cx+4*px, cy-5*px, px, 5*px, dark)
		return
	}
	plot(cx-6*px, cy-9*px, 13*px, 5*px, dark)
	plot(cx-5*px, cy-8*px, 11*px, 3*px, lite)
	plot(cx-5*px, cy-5*px, 11*px, px, wood)
	plot(cx-5*px, cy-4*px, px, 4*px, dark)
	plot(cx+5*px, cy-4*px, px, 4*px, dark)
}

func chairsItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*chairsState)
	plot := g.plotter(dst)
	px := g.pevPx()
	for i, al := range st.alive {
		if !al {
			continue
		}
		s := st.slots[i]
		items = append(items,
			drawItem{s.Y - 0.2, func() { cx, cy := g.pevXY(s); drawChairPix(plot, cx, cy, px, 0) }},
			drawItem{s.Y + 4, func() { cx, cy := g.pevXY(s); drawChairPix(plot, cx, cy, px, 1) }})
	}
	if st.phase == chMusic {
		items = append(items, drawItem{st.center.Y + 60, func() {
			cx, cy := g.pevXY(st.center)
			drawNotesPix(plot, cx, cy-g.pevK()(30), px, g.time, 4)
		}})
	}
	if st.phase == chWin {
		for i, m := range st.mem {
			if i >= 2 || !m.mine() {
				continue
			}
			gu := m.gu
			items = append(items, drawItem{gu.pos.Y + 4.1, func() {
				cx, cy := g.pevXY(gu.pos)
				bob := int(math.Floor(math.Mod(g.time*5, 2)))
				drawCrownPix(plot, cx, cy-g.pevK()(47)-bob)
			}})
		}
	}
	return items
}
