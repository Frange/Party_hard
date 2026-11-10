package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "drunks": for a while three to six ordinary guests (more where there is a bar) are tipsy: they use the staggering
// walk that the game already has (gait "stagger", only a drawing) and their drawn x weaves softly from side to side (an S)
// while they walk. Every now and then one of them trips: he lies on the floor for a moment with stars over his head, says
// something silly and gets up. The weave and the fall are only DRAWING: positions, speed, collisions and the guests'
// own state machine are the game's (a trip holds him still for a second and a half, like a short idle).
//
// WHEN: parties with a bar or a lounge feel (waitersIcons), 25 % of them, between 25 and 150 s, once. Nobody is hurt.
// ---------------------------------------------------------------------------

var drunksLines = pevTexts("¡Hic!", "¡Ups!", "¡Estoy bien, estoy bien!", "Yo controlo... creo", "¡El suelo se ha movido!",
	"¿Quién ha puesto ese escalón?", "No he bebido nada... mucho", "¡Otra ronda!", "Todo da vueltas, ¡qué bien!", "Me he tropezado con el aire")

type drunkOne struct {
	m        *pevMem
	gait     string
	nextTrip float64
	tripEnd  float64
	nextHic  float64
}

type drunksState struct {
	d  []*drunkOne
	t0 float64
}

func (s *drunksState) pevHas(gu *Guest) bool {
	for _, d := range s.d {
		if d.m.gu == gu {
			return true
		}
	}
	return false
}

func init() {
	registerPartyEvent(&partyEvent{ID: "drunks", Name: "Borrachos con eses", Group: "drunk",
		Prob: 0.25, AtMin: 25, AtMax: 150, Dur: 55, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && t.guests >= 20 && (t.bar || waitersIcons[t.icon]) },
		Start:    drunksStart, Items: drunksItems,
		Update: pevGuard(drunksUpdate, func(g *Game, a *partyAct) { drunksEnd(g, a.D.(*drunksState)) })})
}

// drunksCanBe: an ordinary guest (no star, no act of his own, no style already) who is not busy.
func drunksCanBe(g *Game, gu *Guest) bool {
	if !lifeFree(gu) || gu.quirk != "" || gu.leader != nil || gu.armed || gu.partner != nil || gu.name != "" || g.pevTaken(gu) {
		return false
	}
	if starSlugs[guestSlug(gu)] || pevFxs[gu] != nil || g.inSelfie(gu) || g.inContest(gu) {
		return false
	}
	g.gaitInit(gu)
	return gu.gait == "" || gu.gait == "stagger"
}

func drunksStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if drunksCanBe(g, gu) && (gu.state == gGoto || gu.state == gIdle || gu.state == gDance) && gu.timer < 1000 {
			cand = append(cand, gu)
		}
	}
	n := 3 + r.Intn(2)
	if g.pev.traits.bar {
		n += 1 + r.Intn(2)
	}
	if len(cand) < 3 {
		return false
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	if n > len(cand) {
		n = len(cand)
	}
	pevFxPurge(g)
	st := &drunksState{t0: g.time}
	for _, gu := range cand[:n] {
		d := &drunkOne{m: &pevMem{gu: gu, want: gIdle}, gait: gu.gait, nextTrip: g.time + 5 + r.Float64()*16, nextHic: g.time + 2 + r.Float64()*5}
		pevFxOf(gu).sway = 1.8 + r.Float64()*1.4
		gu.gait = "stagger"
		st.d = append(st.d, d)
	}
	a.D = st
	return true
}

func drunkRestore(d *drunkOne) {
	gu := d.m.gu
	if gu.gait == "stagger" {
		gu.gait = d.gait
	}
	if f := pevFxs[gu]; f != nil {
		f.sway, f.trip = 0, 0
	}
	pevFxDrop(gu)
}

func drunksUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*drunksState)
	r := partyDice()
	age := g.time - st.t0
	ending := age > a.ev.Dur-3
	live := st.d[:0]
	for _, d := range st.d {
		gu := d.m.gu
		f := pevFxs[gu]
		tripping := f != nil && g.time < f.trip
		if gu.out() || f == nil {
			if f != nil {
				drunkRestore(d)
			}
			continue
		}
		if tripping {
			if !d.m.mine() { // something else took him (a fright, a murder): he is up and the weave stops
				f.trip = 0
				drunkRestore(d)
				continue
			}
			d.m.keep()
			live = append(live, d)
			continue
		}
		if d.tripEnd > 0 { // just got up
			d.tripEnd = 0
			if d.m.mine() {
				pevRelease(d.m, r)
			}
		}
		if ending {
			drunkRestore(d)
			continue
		}
		live = append(live, d)
		// a trip: only on the move, walking by his own will, and not too close to the others' fun
		if g.time >= d.nextTrip && gu.moving && gu.state == gGoto && !gu.hurry && lifeFree(gu) {
			d.nextTrip = g.time + 12 + r.Float64()*18
			f.trip = g.time + 1.6
			d.tripEnd = f.trip
			d.m.want = gIdle
			d.m.hold(gIdle)
			gu.icon, gu.iconT = pevPick(r, []string{drunksLines[1], drunksLines[5], drunksLines[9]}), 1.5
			g.snd.Play("thud")
			continue
		}
		if g.time >= d.nextHic && gu.icon == "" {
			d.nextHic = g.time + 7 + r.Float64()*9
			gu.icon, gu.iconT = pevPick(r, drunksLines[:9]), 1.8
		}
	}
	st.d = live
	return len(st.d) == 0
}

func drunksEnd(g *Game, st *drunksState) {
	r := partyDice()
	for _, d := range st.d {
		if d.m.mine() && d.m.gu.state == gIdle && d.m.gu.timer > 1000 {
			pevRelease(d.m, r)
		}
		drunkRestore(d)
	}
	st.d = nil
}

func drunksItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*drunksState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, d := range st.d {
		gu := d.m.gu
		f := pevFxs[gu]
		if f == nil || g.time >= f.trip {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(gu.pos)
			drawStarsPix(plot, cx, cy-k(11), px, g.time)
		}})
	}
	return items
}
