package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "granny": a nosy old lady (grey hair in a bun, round glasses, cardigan, skirt, handbag) comes up to the player and
// stays about two tiles away watching what he does, a magnifying glass in her hand, whispering (speech bubbles). If the
// player looks straight at her for a moment she is scandalised and walks away; later she comes back.
//
// She is an ordinary guest of the party (one of the free ones is dressed as her, and stays a granny afterwards): she can be
// killed like anybody and the witness rules are the usual ones, nothing about her accuses the player by herself. If she
// dies the fact is counted (stat "abuela_dead", pevGuestKilled in pe_fx.go) for the newspaper.
//
// WHEN: the parties of grannyIcons with room for her (roaming area, 25 guests or more), 20 % of them, between 40 and 200 s, once.
// ---------------------------------------------------------------------------

// grannyIcons: parties where an old lady is not out of place (a home, a square, a casino, a birthday...).
var grannyIcons = map[string]bool{"house": true, "beach": true, "eiffel": true, "festival": true, "cinema": true, "casino": true,
	"cake": true, "ranch": true, "train": true, "bull": true}

const (
	grannyDist   = 36.0 // two tiles away
	grannyLookT  = 1.1  // seconds of being stared at before she leaves
	grannyAwayT  = 9.0
	grannyBackAt = 4.0
)

var grannyLines = pevTexts("Psst... ¿has visto qué hace ese?", "En mis tiempos esto no se hacía", "Yo lo cuento todo, ¿eh?",
	"Mmm... muy sospechoso", "Este chico no me gusta nada", "¡Y sin abrigo! ¡Se va a constipar!", "¿Dónde vas con tanta prisa, hijo?",
	"Eso no se lo digo ni a mi comadre", "Qué mirada más rara tiene", "Apunto en mi libretita...", "¡Ay, qué modernos son!",
	"Y luego dicen que no pasa nada", "Esto mañana sale en el barrio", "¡Si lo ve mi hermana...!", "Algo trama, yo lo huelo")

var grannyScold = pevTexts("¡Qué descaro, mirar a una señora!", "¡Ay, qué vergüenza!", "¡Uy, qué ojos!", "¡Yo no miraba!")

const (
	grApproach = iota
	grWatch
	grAway
	grRest
)

type grannyState struct {
	m       *pevMem
	gait    string
	phase   int
	t0      float64 // start of the party event
	pt0     float64 // start of the phase
	goal    Vec
	talkAt  float64
	lookT   float64
	repath  float64
	scolds  int
	watched bool // the lens is out
}

func (s *grannyState) pevHas(gu *Guest) bool { return s.m != nil && s.m.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "granny", Name: "Abuela cotilla", Group: "gossip",
		Prob: 0.20, AtMin: 40, AtMax: 200, Dur: 75, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && t.guests >= 25 && grannyIcons[t.icon] },
		Start:    grannyStart, Items: grannyItems,
		Update: pevGuard(grannyUpdate, func(g *Game, a *partyAct) { grannyEnd(g, a.D.(*grannyState)) })})
}

func grannySpec() CharSpec {
	return CharSpec{
		Skin: hx(0xf0cfb0), Hair: hx(0xc9c9d2), Shirt: hx(0x9a6aa8), Shirt2: hx(0x9a6aa8), Pants: hx(0x5c4868), Shoes: hx(0x3a2a2a),
		Top: TopLong, Bottom: BotSkirt, Hair2: HairBun, Glasses: GlassRound, Female: true, Body: BodySlim, Accent: hx(0xe8e0d0), noOverride: true,
	}
}

func grannyStart(g *Game, a *partyAct) bool {
	var best *Guest
	var pl *Player
	for _, p := range g.players() {
		if p != nil && !p.dead && !p.hidden {
			pl = p
			break
		}
	}
	if pl == nil {
		return false
	}
	for _, gu := range g.guests {
		if !g.pevFreeGuest(gu) || gu.name != "" || gu.quirk != "" || pevFxs[gu] != nil || gu.cs == nil || gu.cs.Spec == nil {
			continue
		}
		g.gaitInit(gu)
		if gu.gait != "" || dist(gu.pos, pl.pos) < 90 { // she comes from afar
			continue
		}
		if d := dist(gu.pos, pl.pos); best == nil || d < dist(best.pos, pl.pos) {
			best = gu
		}
	}
	if best == nil {
		return false
	}
	pevFxPurge(g)
	spec := grannySpec()
	st := &grannyState{m: &pevMem{gu: best, want: gIdle}, gait: best.gait, t0: g.time, pt0: g.time, talkAt: g.time + 3}
	best.cs = newCharSet(&spec)
	best.origCS = nil
	best.gait = "glide"
	pevFxOf(best).granny = true
	st.m.want = gIdle
	a.D = st
	// she starts walking at once
	if !grannyGo(g, st, pl) {
		st.phase, st.m.gu.state, st.m.gu.timer = grRest, gIdle, 0.5
	}
	return true
}

// grannyNearest: the player she is nosing at.
func grannyNearest(g *Game, from Vec) *Player {
	var best *Player
	for _, p := range g.players() {
		if p != nil && !p.dead && !p.hidden && (best == nil || dist(p.pos, from) < dist(best.pos, from)) {
			best = p
		}
	}
	return best
}

// grannyGo: walk to the place two tiles from the player, on her side of him.
func grannyGo(g *Game, st *grannyState, pl *Player) bool {
	gu := st.m.gu
	to := faceVec(pl.pos, gu.pos)
	goal := g.lvl.NearestFree(Vec{pl.pos.X + to.X*grannyDist, pl.pos.Y + to.Y*grannyDist*0.8})
	if dist(goal, pl.pos) < 14 {
		return false
	}
	if !g.pevSendTo(st.m, goal) {
		return false
	}
	st.goal, st.phase, st.pt0 = goal, grApproach, g.time
	return true
}

func grannyUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*grannyState)
	r := partyDice()
	gu := st.m.gu
	if !st.m.mine() {
		grannyEnd(g, st) // dead, frightened or taken: the event is over
		return true
	}
	if g.time-st.t0 > a.ev.Dur-8 && st.phase != grAway {
		grannyEnd(g, st)
		return true
	}
	pl := grannyNearest(g, gu.pos)
	if pl == nil {
		grannyEnd(g, st)
		return true
	}
	st.watched = false
	age := g.time - st.pt0
	switch st.phase {
	case grApproach:
		st.repath -= dt
		if gu.state == gIdle && gu.timer > 1000 { // arrived
			st.m.want = gIdle
			if dist(gu.pos, pl.pos) < grannyDist+22 {
				st.phase, st.pt0, st.lookT = grWatch, g.time, 0
				break
			}
			grannyGo(g, st, pl) // the player went away meanwhile
		} else if st.repath <= 0 {
			st.repath = 1.2
			if dist(st.goal, pl.pos) > grannyDist+26 {
				grannyGo(g, st, pl)
			}
		}
		if age > 40 {
			grannyEnd(g, st)
			return true
		}
	case grWatch:
		st.m.keep()
		gu.dir = faceTo(gu.pos, pl.pos)
		st.watched = true
		if dist(gu.pos, pl.pos) > grannyDist+34 { // he walked off: follow
			grannyGo(g, st, pl)
			break
		}
		// being stared at?
		v := dirVec(pl.dir)
		u := faceVec(pl.pos, gu.pos)
		if dist(gu.pos, pl.pos) < 72 && v.X*u.X+v.Y*u.Y > 0.7 {
			st.lookT += dt
		} else {
			st.lookT = math.Max(0, st.lookT-dt*0.5)
		}
		if st.lookT >= grannyLookT {
			st.scolds++
			gu.icon, gu.iconT = pevPick(r, grannyScold), 2.2
			// away from the player, to a corner of the roaming area
			var far Vec
			for i := 0; i < 24; i++ {
				far = g.lvl.NearestFree(g.lvl.RandomRoam(r))
				if dist(far, pl.pos) > 170 {
					break
				}
			}
			m := st.m
			if g.pevSendTo(m, far) {
				st.phase, st.pt0, st.lookT = grAway, g.time, 0
			} else {
				st.lookT = 0
			}
			break
		}
		if g.time >= st.talkAt {
			st.talkAt = g.time + 3.2 + r.Float64()*2
			if gu.icon == "" {
				gu.icon, gu.iconT = pevPick(r, grannyLines), 2.6
			}
		}
	case grAway:
		st.watched = false
		if (gu.state == gIdle && gu.timer > 1000) || age > grannyAwayT {
			st.m.hold(gIdle)
			st.phase, st.pt0 = grRest, g.time
		}
	case grRest:
		st.m.keep()
		gu.moving = false
		if age > grannyBackAt {
			if !grannyGo(g, st, pl) {
				st.pt0 = g.time
			}
		}
	}
	return false
}

func grannyEnd(g *Game, st *grannyState) {
	gu := st.m.gu
	pevLetGo(st.m)
	if !gu.out() && gu.gait == "glide" {
		gu.gait = "" // she walks like any guest, still dressed as a granny
	}
	if f := pevFxs[gu]; f != nil && !f.granny {
		pevFxDrop(gu)
	}
}

func grannyItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*grannyState)
	if !st.m.mine() {
		return items
	}
	gu := st.m.gu
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	return append(items, drawItem{gu.pos.Y + 0.3, func() {
		cx, cy := g.pevXY(gu.pos)
		if gu.dir != DirUp {
			plot(cx-6*px, cy-k(14), 3*px, 3*px, pevBrown) // the handbag
			plot(cx-6*px, cy-k(14), 3*px, px, pevGoldDk)
		}
		if st.watched {
			side := 1
			if gu.dir == DirLeft {
				side = -1
			}
			drawLensPix(plot, cx+side*k(9), cy-k(22), px)
		}
	}})
}
