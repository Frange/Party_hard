package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "philosopher": a tipsy guest leans on the bar (back to the room, elbow on the counter), sways a little and every few
// seconds comes out with a deep, absurd thought in a speech bubble, with one finger raised. The ones beside him either
// ignore him or nod ("Totally"). He is held (state idle) only while he speaks and is let go at the end; if anything else
// takes him (a fright, a murder) the event just ends.
//
// WHEN: indoor parties with a bar, 20 % of them, between 40 and 220 s, once. Group "bar" (never with the drinking contest).
// ---------------------------------------------------------------------------

var philoLines = pevTexts(
	"¿Y si el DJ somos todos?", "La barra no existe; la sed sí", "Si bailo solo, ¿bailo o me caigo?",
	"El hielo es agua que se ha puesto seria", "Somos polvo de estrellas con resaca", "El vaso no está vacío: está pensando",
	"¿Quién fue primero, la fiesta o el ruido?", "Si nadie escucha, ¿suena la música?", "El tiempo es un círculo... como el hielo",
	"Los espejos de la disco saben demasiado", "La vida es un canapé: se acaba rápido", "¿Y si la luna se ha ido de fiesta sin mí?",
	"El silencio es música que no se invitó", "Todos somos el último invitado de alguien", "La resaca es el universo cobrando",
	"El camarero es el verdadero filósofo", "Nadie baila dos veces en el mismo suelo", "Beber es pensar con más ganas",
	"Hay un universo en cada burbuja", "¿Y si el baño me está esperando a mí?", "Cada confeti fue un sueño en rosa",
	"La noche es joven; yo, un poco menos", "Nunca sabrás cuántos bailamos de verdad", "La cuenta es lo único que nunca baila")

var philoNods = pevTexts("Ajá...", "Qué profundo", "Hmm, totalmente", "Ya, ya.", "Tiene razón", "...")

type philoState struct {
	m       *pevMem
	gait    string
	spot    Vec
	order   []int
	n       int
	t0      float64
	talkAt  float64
	sayEnd  float64 // the finger stays up until this time
	arrived bool
}

func (s *philoState) pevHas(gu *Guest) bool { return s.m != nil && s.m.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "philosopher", Name: "Borracho filósofo", Group: "bar",
		Prob: 0.20, AtMin: 40, AtMax: 220, Dur: 60, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.indoor && t.bar && t.roam && t.guests >= 20 },
		Start:    philoStart, Items: philoItems,
		Update: pevGuard(philoUpdate, func(g *Game, a *partyAct) { philoEnd(g, a.D.(*philoState)) })})
}

func philoStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var spots []Vec
	for _, p := range g.lvl.POIs["drink"] {
		q := Vec{p.X, p.Y}
		if g.lvl.Free(q) && !g.lvl.badGround(q) && !g.lvl.InPool(q) && g.pevPlayerFar(q, 40) {
			spots = append(spots, q)
		}
	}
	if len(spots) == 0 {
		return false
	}
	r.Shuffle(len(spots), func(i, j int) { spots[i], spots[j] = spots[j], spots[i] })
	for _, spot := range spots {
		var best *Guest
		for _, gu := range g.guests {
			if !g.pevFreeGuest(gu) || gu.name != "" || pevFxs[gu] != nil {
				continue
			}
			g.gaitInit(gu)
			if gu.gait != "" && gu.gait != "stagger" {
				continue
			}
			if d := dist(gu.pos, spot); d < 300 && (best == nil || d < dist(best.pos, spot)) {
				best = gu
			}
		}
		if best == nil {
			continue
		}
		m := &pevMem{gu: best}
		if !g.pevSendTo(m, spot) {
			continue
		}
		pevFxPurge(g)
		st := &philoState{m: m, gait: best.gait, spot: spot, t0: g.time}
		st.order = r.Perm(len(philoLines))
		best.gait = "stagger"
		st.talkAt = g.time + 2
		a.D = st
		return true
	}
	return false
}

func philoUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*philoState)
	r := partyDice()
	gu := st.m.gu
	if !st.m.mine() {
		philoEnd(g, st)
		return true
	}
	age := g.time - st.t0
	if !st.arrived {
		if gu.state == gIdle && gu.timer > 1000 {
			st.arrived = true
			st.m.want = gIdle
			gu.dir = DirUp // elbow on the bar, back to the room
		} else if age > 24 {
			philoEnd(g, st)
			return true
		}
		return false
	}
	st.m.keep()
	if age > a.ev.Dur-6 {
		philoEnd(g, st)
		return true
	}
	if g.time >= st.talkAt {
		st.talkAt = g.time + 5.2 + r.Float64()*1.6
		st.sayEnd = g.time + 3.6
		gu.icon, gu.iconT = philoLines[st.order[st.n%len(st.order)]], 3.6
		st.n++
		// the ones beside him: some nod, most ignore him
		k := 0
		for _, o := range g.guests {
			if k >= 2 {
				break
			}
			if o != gu && dist(o.pos, gu.pos) < 56 && lifeFree(o) && o.icon == "" && !g.pevTaken(o) && o.state != gDrink {
				if r.Float64() < 0.4 {
					o.icon, o.iconT = pevPick(r, philoNods), 1.6
					o.dir = faceTo(o.pos, gu.pos)
				}
				k++
			}
		}
	}
	return false
}

func philoEnd(g *Game, st *philoState) {
	gu := st.m.gu
	if gu.gait == "stagger" {
		gu.gait = st.gait
	}
	pevLetGo(st.m)
}

func philoItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*philoState)
	if !st.arrived || g.time >= st.sayEnd || !st.m.mine() {
		return items
	}
	gu := st.m.gu
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	return append(items, drawItem{gu.pos.Y + 0.3, func() {
		cx, cy := g.pevXY(gu.pos)
		drawFingerPix(plot, cx, cy, px, 1, k)
	}})
}
