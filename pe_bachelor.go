package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "bachelor": a hen/stag party. A group of 4-6 guests with sashes and silly party gear (the honoree wears a veil
// and a tiara, or a crown and star glasses) joins up and goes through the party singing (notes and speech
// bubbles), the honoree in front, the others walking his steps. They stop at a bar (or an open spot) and raise
// their glasses with a "¡Chin chin!". With E on the honoree the player gets a silly dare in a speech bubble.
// No points, nothing is gained or lost: it is only company.
//
// WHEN: only in parties that fit it (house, neon club, disco, 70s disco, club, beach, cruise, Paris, fair, festival,
// casino): 18 % of them, between 60 and 240 s, up to twice. Big (so it is never together with the conga or the
// chairs). The honoree walks with the normal guest walking, so ice, props and doors behave as always.
// ---------------------------------------------------------------------------

var bachelorIcons = map[string]bool{"house": true, "neon": true, "disco": true, "disco70": true, "club": true, "beach": true,
	"boat": true, "eiffel": true, "funfair": true, "festival": true, "casino": true}

const (
	baGatherMax = 10.0
	baWalkMax   = 34.0
	baToastLen  = 7.0
	baGap       = 15.0
)

const (
	baGather = iota
	baWalk
	baToast
)

var bachelorSongs = pevTexts("¡Que se besen, que se besen!", "¡Hoy se casa el amigo!", "¡Que viva el novio!", "¡Que viva la novia!", "¡Yo soy la que más baila!",
	"¡Esta noche no hay vuelta atrás!", "¡Cantad conmigo!", "¡Despedida, despedida!")
var bachelorToasts = pevTexts("¡CHIN CHIN!", "¡Por la despedida!", "¡Por los nervios!", "¡Que sea un gran sí!")
var bachelorDares = pevTexts("¡Reto: baila como un pato!", "¡Reto: cuenta un chiste malo!", "¡Reto: habla en rima un minuto!",
	"¡Reto: imita a un camarero!", "¡Reto: un cumplido al primero que veas!", "¡Reto: abraza a un desconocido!", "¡Reto: grita \"patata\" tres veces!")
var bachelorCheers = pevTexts("¡Que lo haga!", "¡Reto aceptado!", "¡Cobarde!", "¡Eso, eso!")
var bachelorStart = pevTexts("¡Despedida de soltero en marcha!", "¡Despedida de soltera en marcha!")

type bachelorState struct {
	honor, gang []*pevMem // gang does not include the honoree
	trail       pevTrail
	stops       []Vec
	leg         int
	phase       int
	t0          float64
	nextSong    float64
	nextRepath  float64
	dareAt      float64
	toastSent   bool
	clinkAt     float64
	female      bool
	last        Vec
}

func (b *bachelorState) all() []*pevMem { return append(append([]*pevMem{}, b.honor...), b.gang...) }
func (b *bachelorState) pevHas(gu *Guest) bool {
	return pevHasIn(b.honor, gu) || pevHasIn(b.gang, gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "bachelor", Name: "Despedida de soltero/a", Group: "party", Big: true,
		Prob: 0.18, AtMin: 60, AtMax: 240, Dur: 36, Cooldown: 170, MaxRuns: 2, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return bachelorIcons[t.icon] && t.roam && t.guests >= 24 },
		Start:    bachelorStart_, Items: bachelorItems,
		Update: pevGuard(bachelorUpdate, func(g *Game, a *partyAct) { bachelorEnd(g, a.D.(*bachelorState)) })})
}

func bachelorStart_(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 7 {
		return false
	}
	h := cand[r.Intn(len(cand))]
	// the stops: a point on the way, then the bar (or any open spot) where they toast
	st := &bachelorState{t0: g.time, female: h.cs.Spec.Female}
	for i := 0; i < 40 && len(st.stops) < 1; i++ {
		p := g.lvl.RandomRoam(r)
		if d := dist(p, h.pos); d > 70 && d < 190 && g.pevClear(p, 14) && g.lvl.PathG(h.pos, p) != nil {
			st.stops = append(st.stops, p)
		}
	}
	last := Vec{}
	ok := false
	for _, pr := range g.lvl.Props {
		if !ok && !pr.gone && (pr.Name == "bar" || pr.Name == "minibar") {
			for _, dy := range []float64{22, 32} {
				p := Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y + pr.Foot.H + dy}
				if g.pevClear(p, 16) && g.lvl.PathG(h.pos, p) != nil {
					last, ok = p, true
					break
				}
			}
		}
	}
	for i := 0; i < 60 && !ok; i++ {
		p := g.lvl.RandomRoam(r)
		if d := dist(p, h.pos); d > 70 && d < 220 && g.pevClear(p, 16) && g.lvl.PathG(h.pos, p) != nil {
			last, ok = p, true
		}
	}
	if !ok || len(st.stops) == 0 {
		return false
	}
	st.stops = append(st.stops, last)
	hm := &pevMem{gu: h}
	hm.hold(gIdle)
	st.honor = []*pevMem{hm}
	// the group: the nearest free guests who can reach him
	n := 3 + r.Intn(3)
	for len(st.gang) < n {
		var best *Guest
		for _, gu := range cand {
			if gu == h || pevHasIn(st.gang, gu) || g.lvl.PathG(gu.pos, h.pos) == nil {
				continue
			}
			if best == nil || dist(gu.pos, h.pos) < dist(best.pos, h.pos) {
				best = gu
			}
		}
		if best == nil || dist(best.pos, h.pos) > 150 {
			break
		}
		m := &pevMem{gu: best, idx: len(st.gang) + 1}
		an := float64(len(st.gang)) / float64(n) * 2 * math.Pi
		pt := g.lvl.NearestFree(Vec{h.pos.X + math.Cos(an)*17, h.pos.Y + math.Sin(an)*13})
		if !g.pevSendTo(m, pt) {
			cand = removeGuest(cand, best)
			continue
		}
		st.gang = append(st.gang, m)
		cand = removeGuest(cand, best)
	}
	if len(st.gang) < 3 {
		pevRelease(hm, r)
		pevReleaseAll(st.gang, r)
		return false
	}
	st.trail.add(h.pos)
	st.nextSong = g.time + 3
	if st.female {
		g.say(bachelorStart[1])
	} else {
		g.say(bachelorStart[0])
	}
	a.D = st
	return true
}

func removeGuest(l []*Guest, gu *Guest) []*Guest {
	for i, o := range l {
		if o == gu {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

func bachelorEnd(g *Game, st *bachelorState) {
	r := partyDice()
	pevReleaseAll(st.all(), r)
}

func bachelorUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*bachelorState)
	r := partyDice()
	age := g.time - st.t0
	if !st.honor[0].mine() { // the honoree is hurt, scared away or gone: the party breaks up
		bachelorEnd(g, st)
		return true
	}
	st.gang = pevPruneMems(st.gang)
	if len(st.gang) < 2 {
		bachelorEnd(g, st)
		return true
	}
	h := st.honor[0].gu
	idle := func(m *pevMem) bool { return m.gu.state == gIdle && m.gu.timer > 1000 }
	switch st.phase {
	case baGather:
		all := true
		st.honor[0].keep()
		for _, m := range st.gang {
			if idle(m) {
				m.want = gIdle
				m.keep()
				m.gu.dir = faceTo(m.gu.pos, h.pos)
			} else {
				all = false
			}
		}
		if all || age > baGatherMax {
			st.phase, st.t0 = baWalk, g.time
			if !g.pevStartLeg(st) {
				bachelorEnd(g, st)
				return true
			}
		}
	case baWalk:
		st.honor[0].want = gGoto
		if idle(st.honor[0]) { // arrived at a stop
			st.honor[0].want = gIdle
			st.honor[0].keep()
			st.leg++
			if st.leg >= len(st.stops) || age > baWalkMax {
				bachelorToast(g, st, r)
				return false
			}
			if !g.pevStartLeg(st) {
				bachelorToast(g, st, r)
				return false
			}
		} else if age > baWalkMax+8 {
			bachelorToast(g, st, r)
			return false
		}
		if moved := dist(h.pos, st.last) > 0.01; moved {
			st.trail.add(h.pos)
		}
		st.last = h.pos
		// the others keep their place in the line behind him
		if g.time >= st.nextRepath {
			st.nextRepath = g.time + 0.45
			for i, m := range st.gang {
				slot := st.trail.at(float64(i+1) * baGap)
				if dist(m.gu.pos, slot) > 9 {
					g.pevSendTo(m, slot)
				}
			}
		}
		for _, m := range st.gang {
			if idle(m) {
				m.want = gIdle
				m.keep()
				m.gu.dir = h.dir
			}
		}
		if g.time >= st.nextSong {
			st.nextSong = g.time + 2.2 + r.Float64()*1.2
			if all := st.all(); len(all) > 0 {
				c := all[r.Intn(len(all))].gu
				if c.icon == "" {
					c.icon, c.iconT = pevPick(r, bachelorSongs), 2.4
				}
			}
		}
	case baToast:
		for _, m := range st.gang {
			if idle(m) {
				m.want = gIdle
				m.keep()
				m.gu.dir = faceTo(m.gu.pos, h.pos)
			}
		}
		st.honor[0].keep()
		if st.clinkAt == 0 {
			ready := true
			for _, m := range st.gang {
				if !idle(m) {
					ready = false
				}
			}
			if ready || age > baGatherMax {
				st.clinkAt = g.time + 1.6
				st.t0 = g.time
			}
		} else {
			if g.time >= st.clinkAt && !st.toastSent {
				st.toastSent = true
				h.icon, h.iconT = bachelorToasts[0], 2.6
				for _, m := range st.gang {
					if r.Float64() < 0.5 && m.gu.icon == "" {
						m.gu.icon, m.gu.iconT = pevPick(r, bachelorToasts[1:]), 2.4
					}
				}
				g.snd.Play("spark")
			}
			if g.time-st.clinkAt > baToastLen*0.6 {
				bachelorEnd(g, st)
				return true
			}
		}
	}
	return false
}

// pevStartLeg sends the honoree to the next stop.
func (g *Game) pevStartLeg(st *bachelorState) bool {
	if st.leg >= len(st.stops) {
		return false
	}
	return g.pevSendTo(st.honor[0], st.stops[st.leg])
}

func bachelorToast(g *Game, st *bachelorState, r interface{ Float64() float64 }) {
	st.phase, st.t0 = baToast, g.time
	st.honor[0].want = gIdle
	st.honor[0].gu.timer = 9999
	st.honor[0].gu.path = nil
	h := st.honor[0].gu
	for i, m := range st.gang {
		an := float64(i)/float64(len(st.gang))*2*math.Pi + 0.4
		pt := g.lvl.NearestFree(Vec{h.pos.X + math.Cos(an)*17, h.pos.Y + math.Sin(an)*13})
		g.pevSendTo(m, pt)
	}
}

// pevUse: E on the honoree: a silly dare (a speech bubble, nothing else).
func (b *bachelorState) pevUse(g *Game, p *Player) bool {
	if !b.honor[0].mine() {
		return false
	}
	h := b.honor[0].gu
	if dist(p.pos, h.pos) > 30 {
		return false
	}
	if g.time < b.dareAt {
		return true
	}
	r := partyDice()
	b.dareAt = g.time + 5
	h.icon, h.iconT = pevPick(r, bachelorDares), 3.4
	for _, m := range b.gang {
		if r.Float64() < 0.5 && m.gu.icon == "" {
			m.gu.icon, m.gu.iconT = pevPick(r, bachelorCheers), 2.4
			break
		}
	}
	g.snd.Play("spark")
	return true
}

var (
	baPink = color.RGBA{0xf0, 0x5a, 0xa0, 255}
	baTeal = color.RGBA{0x40, 0xd0, 0xc8, 255}
	baVeil = rgba(0xf0f0ff, 105)
)

// drawBachelorKitPix: the gear of one member: kind 0 = honoree (veil + tiara or crown + glasses), 1 = glasses,
// 2 = party cone, 3 = bunny ears; every one wears a sash. Feet at (cx, cy).
func drawBachelorKitPix(plot plotFn, cx, cy, px int, k func(float64) int, kind int, female bool, sash color.RGBA) {
	// the sash: from the shoulder to the opposite hip
	drawLinePix(plot, cx-k(4), cy-k(27), cx+k(4), cy-k(15), maxI(1, k(2)), sash)
	drawLinePix(plot, cx-k(4), cy-k(27)+px, cx+k(4), cy-k(15)+px, px, pevGold)
	top := cy - k(31)
	switch kind {
	case 0:
		if female {
			plot(cx-5*px, top-px, 10*px, 3*px, baVeil) // the veil falls from the head
			plot(cx-6*px, top+2*px, 12*px, k(14), baVeil)
			plot(cx-3*px, top-3*px, 7*px, px, color.RGBA{0xd8, 0xe8, 0xf8, 255}) // tiara
			plot(cx, top-4*px, px, px, color.RGBA{0xff, 0x60, 0x90, 255})
		} else {
			drawCrownPix(plot, cx, top-3*px)
			drawStarGlassesPix(plot, cx, cy-k(28), px, pevGold)
		}
	case 1:
		drawStarGlassesPix(plot, cx, cy-k(28), px, baTeal)
	case 2:
		for i := 0; i < 6; i++ { // a cone with stripes
			c := baPink
			if i%2 == 1 {
				c = pevWhite
			}
			w := 7 - i
			plot(cx-w*px/2, top-(i+1)*px, w*px, px, c)
		}
		plot(cx-px, top-8*px, 3*px, px, pevGold)
	case 3:
		plot(cx-4*px, top-7*px, 2*px, 7*px, pevWhite)
		plot(cx+2*px, top-7*px, 2*px, 7*px, pevWhite)
		plot(cx-4*px, top-5*px, px, 4*px, baPink)
		plot(cx+3*px, top-5*px, px, 4*px, baPink)
	}
}

func bachelorItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*bachelorState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for i, m := range st.all() {
		i, m := i, m
		if !m.mine() {
			continue
		}
		gu := m.gu
		items = append(items, drawItem{gu.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(gu.pos)
			kind := 0
			sash := baPink
			if i > 0 {
				kind = 1 + (i-1)%3
				if i%2 == 0 {
					sash = baTeal
				}
			}
			drawBachelorKitPix(plot, cx, cy, px, k, kind, st.female, sash)
			if st.phase == baToast && st.clinkAt > 0 && g.time >= st.clinkAt-0.5 { // the raised glass
				up := 0
				if int(g.time*4)%2 == 0 {
					up = px
				}
				drawGlassPix(plot, cx+k(7), cy-k(40)-up, px)
			}
		}})
	}
	if st.phase == baWalk && len(st.honor) > 0 && st.honor[0].mine() {
		h := st.honor[0].gu
		items = append(items, drawItem{h.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(h.pos)
			drawNotesPix(plot, cx, cy-k(47), px, g.time, 3)
		}})
	}
	if st.phase == baToast && st.clinkAt > 0 && g.time >= st.clinkAt && g.time < st.clinkAt+0.5 {
		h := st.honor[0].gu
		items = append(items, drawItem{h.pos.Y + 0.4, func() { // the clink: a little burst over the middle
			cx, cy := g.pevXY(h.pos)
			u := (g.time - st.clinkAt) / 0.5
			rr := int(math.Round(4 + u*7))
			for j := 0; j < 8; j++ {
				an := float64(j) * math.Pi / 4
				plot(cx+int(math.Round(math.Cos(an)*float64(rr)))*px, cy-k(46)+int(math.Round(math.Sin(an)*float64(rr)))*px, px, px, pevGold)
			}
		}})
	}
	return items
}
