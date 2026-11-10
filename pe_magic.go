package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "magician": a magician (cape, top hat, wand) does four little tricks in front of an audience: a RABBIT out of
// the hat, a chain of coloured SCARVES, CARDS that fly all over the place... and now and then a trick FAILS (a puff
// of smoke, the hat falls off) and the guests boo; when it works they applaud.
// Sometimes he picks the player who walks by: "¡Tú! ¡Sígueme el truco!". If the player uses (E) near him within a few
// seconds he gets a small tip (25 points, ONCE per party); if not, "¡Qué mal público!" and a few boos.
//
// BIRTHDAY PARTY: it already has a magician as a decoration (prop "entertainer", variant 1). Here the show happens AT
// that prop (he stays where he is and does the tricks; nobody is duplicated). Everywhere else the magician walks in.
// He can be stabbed like the pizza boy (not the decorative one). No real names.
//
// WHEN: house, club, neon, casino, cruise, festival, fair and birthday parties, 20 % of them, between 60 and 220 s, once.
// Group "street": never together with the mime, the juggler or the buskers.
// ---------------------------------------------------------------------------

var magicIcons = map[string]bool{"house": true, "neon": true, "club": true, "casino": true, "boat": true, "festival": true, "funfair": true, "cake": true}

const (
	magicActLen   = 27.0
	magicTrickLen = 6.6
	magicTip      = 25
	magicReach    = 56.0
	magicPickWin  = 5.0
)

var magicLines = pevTexts("¡Abracadabra, pata de cabra!", "¡Nada por aquí, nada por allá!", "¡Y ahora, el gran final!", "¡Presto!", "¡Ta-chán!",
	"¡Esto no estaba en el ensayo!", "¿Dónde está el conejo?", "¡Se me cae el sombrero!", "¡Elige una carta! ¡Esa!")
var magicCheers = pevTexts("¡Bravo!", "¡Magia!", "¡Otra, otra!", "¡Cómo lo hace!")
var magicBoos = pevTexts("¡Buuu!", "¡Menudo timo!", "¡Devuelve el dinero!", "¡Qué mal truco!")
var magicPick = pevTexts("¡Tú! ¡Sígueme el truco!", "¡Gran público! Toma, tu propina", "¡Qué mal público!")

// magicGiven: the party (level) in which the player already got the tip: once per party.
var magicGiven *Level

const (
	mgRabbit = iota
	mgScarves
	mgCards
	mgTricks
)

type magicState struct {
	show          pevShow
	prop          *Prop // birthday: the decorative magician does the tricks
	csHat, csBare *CharSet
	order         []int
	fail          []bool
	trick         int
	tt0           float64
	reacted       bool
	pickEnd       float64
	pickDone      bool
	pickTried     bool
	pickWho       *Player
	tipped        bool
	dead          bool
	hatDown       bool
	cheeredTrick  int
}

func (m *magicState) pevHas(gu *Guest) bool { return m.show.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "magician", Name: "Mago", Group: "street",
		Prob: 0.20, AtMin: 60, AtMax: 220, Dur: 45,
		Eligible: func(g *Game, t partyTraits) bool { return magicIcons[t.icon] && t.roam && t.guests >= 20 },
		Start:    magicStart, Items: magicItems,
		Update: pevGuard(magicUpdate, func(g *Game, a *partyAct) { a.D.(*magicState).show.end(g) })})
}

func magicSpec(hat bool) CharSpec {
	sp := CharSpec{Skin: hx(0xf0c8a0), Hair: hx(0x201810), Shirt: hx(0x14101c), Shirt2: hx(0xf4f4f8), Pants: hx(0x14101c), Shoes: hx(0x0c0a10),
		Accent: hx(0xc83a50), Cape: hx(0x2a1a4a), Top: TopSuit, Beard: 1, noOverride: true}
	if hat {
		sp.Hat, sp.HatCol = HatTopHat, hx(0x14101c)
	}
	return sp
}

func magicStart(g *Game, a *partyAct) bool {
	r := partyDice()
	st := &magicState{}
	var prop *Prop
	if g.lvl.Def.Icon == "cake" {
		for _, p := range g.lvl.Props {
			if p.Name == "entertainer" && p.Var == 1 && !p.gone {
				prop = p
			}
		}
	}
	if prop != nil { // the decorative magician performs where he stands
		pos := Vec{prop.X, prop.Y}
		st.prop = prop
		st.show = pevShow{pf: newPevPerf(magicSpec(true), pos, DirDown), spot: pos, tip: Vec{pos.X + 12, pos.Y + 4}, actLen: magicActLen, t0: g.time, phase: showAct}
		st.show.crowd.gather(g, r, pos, 4+r.Intn(2), 30, 46, nil)
		if len(st.show.crowd.mem) < 2 {
			return false
		}
	} else {
		s, ok := g.pevShowStart(magicSpec(true), magicActLen, false)
		if !ok {
			return false
		}
		st.show = *s
	}
	sb := magicSpec(false)
	st.csHat, st.csBare = st.show.pf.cs, newCharSet(&sb)
	st.order = []int{0, 1, 2, r.Intn(mgTricks)}
	r.Shuffle(len(st.order), func(i, j int) { st.order[i], st.order[j] = st.order[j], st.order[i] })
	st.fail = make([]bool, len(st.order))
	for i := range st.fail {
		st.fail[i] = r.Float64() < 0.3
	}
	st.fail[0] = false // the first one always works
	a.D = st
	return true
}

func magicUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*magicState)
	r := partyDice()
	s := &st.show
	prev := s.phase
	if st.dead { // stabbed: the audience is free and the event is over
		s.end(g)
		return true
	}
	if done := g.pevShowStep(s, dt, 4+r.Intn(2), 5, nil); done {
		s.end(g)
		return true
	}
	if prev == showEnter && s.phase == showAct {
		st.tt0 = g.time
		s.pf.dir = DirDown
	}
	if s.phase != showAct {
		return false
	}
	pf := s.pf
	age := g.time - s.t0
	st.trick = int(age / magicTrickLen)
	if st.trick >= len(st.order) {
		st.trick = len(st.order) - 1
	}
	u := age - float64(st.trick)*magicTrickLen
	if st.trick != st.cheeredTrick-1 && u < 0.2 {
		st.hatDown, st.reacted = false, false
		pf.cs = st.csHat
		st.cheeredTrick = st.trick + 1
		if st.trick > 0 {
			bubbleOnce(g, pf, magicLines[r.Intn(5)])
		}
	}
	// the failure: smoke, the hat falls; then the audience boos. The success: applause
	if st.fail[st.trick] {
		if u >= 3.0 && !st.hatDown {
			st.hatDown = true
			pf.cs = st.csBare
			bubbleOnce(g, pf, magicLines[5+r.Intn(3)])
			g.snd.Play("spark")
		}
	}
	if u >= 4.0 && !st.reacted {
		st.reacted = true
		pool := magicCheers
		if st.fail[st.trick] {
			pool = magicBoos
		}
		for _, m := range s.crowd.mem {
			if m.gu.icon == "" && r.Float64() < 0.6 {
				m.gu.icon, m.gu.iconT = pevPick(r, pool), 1.8
			}
		}
	}
	// the volunteer
	if !st.pickTried && st.trick >= 1 && u >= 1.2 && st.pickEnd == 0 {
		for _, pl := range g.players() {
			if pl != nil && !pl.dead && dist(pl.pos, pf.pos) < 90 {
				st.pickTried, st.pickEnd, st.pickWho = true, g.time+magicPickWin, pl
				pf.dir = faceTo(pf.pos, pl.pos)
				bubbleOnce(g, pf, magicPick[0])
				break
			}
		}
	}
	if st.pickEnd > 0 && !st.pickDone && g.time > st.pickEnd {
		st.pickDone = true
		bubbleOnce(g, pf, magicPick[2])
		for i, m := range s.crowd.mem {
			if i < 3 && m.gu.icon == "" {
				m.gu.icon, m.gu.iconT = pevPick(r, magicBoos), 1.8
			}
		}
	}
	return false
}

// pevUse: E near the magician while he has picked someone: the player follows the trick (tip once per party).
func (m *magicState) pevUse(g *Game, p *Player) bool {
	if m.show.phase != showAct || m.pickEnd == 0 || m.pickDone || g.time > m.pickEnd || dist(p.pos, m.show.pf.pos) > magicReach {
		return false
	}
	m.pickDone = true
	m.show.pf.dir = faceTo(m.show.pf.pos, p.pos)
	if magicGiven != g.lvl {
		magicGiven, m.tipped = g.lvl, true
		g.award(m.show.pf.pos, magicTip, "")
	}
	bubbleOnce(g, m.show.pf, magicPick[1])
	r := partyDice()
	for _, mm := range m.show.crowd.mem {
		if mm.gu.icon == "" && r.Float64() < 0.6 {
			mm.gu.icon, mm.gu.iconT = pevPick(r, magicCheers), 1.8
		}
	}
	g.snd.Play("spark")
	return true
}

// pevStab: the walking magician can be stabbed (not the decoration).
func (m *magicState) pevStab(g *Game, p *Player, maxD float64) bool {
	if m.prop != nil || m.dead || m.show.phase == showEnter || dist(p.pos, m.show.pf.pos) > math.Min(maxD, tvReach) {
		return false
	}
	m.dead = true
	g.pevKillPerf(m.show.pf, "Mago")
	return true
}

// ---- pixel pieces (through plotFn) -------------------------------------------------------------------------------

var magicScarfCols = []color.RGBA{{0xe8, 0x30, 0x40, 255}, {0xf8, 0xd8, 0x30, 255}, {0x30, 0x90, 0xf0, 255}, {0x40, 0xc8, 0x60, 255},
	{0xe0, 0x60, 0xd0, 255}, {0xf0, 0x90, 0x30, 255}}

// drawRabbitPix: a white rabbit of about 6x9 sprite pixels sitting at (x, y) (its feet); ears wiggle with t.
func drawRabbitPix(plot plotFn, x, y, px int, t float64) {
	wh := pevWhite
	sh := color.RGBA{0xc8, 0xc4, 0xbc, 255}
	pk := color.RGBA{0xf0, 0x90, 0xa0, 255}
	w := int(math.Round(math.Sin(t * 9)))
	plot(x-3*px, y-4*px, 6*px, 4*px, wh) // body
	plot(x-3*px, y-px, 6*px, px, sh)
	plot(x-2*px, y-7*px, 4*px, 3*px, wh) // head
	plot(x-2*px+w*px, y-11*px, px, 4*px, wh)
	plot(x+px-w*px, y-11*px, px, 4*px, wh)
	plot(x-2*px+w*px, y-10*px, px, 2*px, pk)
	plot(x-px, y-6*px, px, px, pevInk) // eye
	plot(x+px, y-6*px, px, px, pevInk)
	plot(x+3*px, y-3*px, px, 2*px, wh) // tail
}

// drawCardPix: a playing card of 3x4 sprite pixels with its top-left corner at (x, y); i picks the pip.
func drawCardPix(plot plotFn, x, y, px, i int) {
	plot(x, y, 3*px, 4*px, pevInk)
	plot(x+px, y+px, px, 2*px, pevWhite)
	plot(x, y+px, px, 2*px, pevWhite)
	plot(x+2*px, y+px, px, 2*px, pevWhite)
	if i%2 == 0 {
		plot(x+px, y+px, px, px, color.RGBA{0xe0, 0x30, 0x40, 255})
	}
}

// drawScarvesPix: a chain of n coloured scarves waving away from the hand at (x, y).
func drawScarvesPix(plot plotFn, x, y, px, n int, t float64) {
	for j := 0; j < n; j++ {
		wy := int(math.Round(math.Sin(float64(j)*0.8+t*6) * 2))
		plot(x+j*3*px, y+wy*px-j*px/2, 3*px, 3*px, magicScarfCols[j%len(magicScarfCols)])
	}
}

// drawTopHatPix: a top hat on the ground (8x6 sprite pixels) with its base centred on (x, y).
func drawTopHatPix(plot plotFn, x, y, px int) {
	k := color.RGBA{0x14, 0x10, 0x1c, 255}
	plot(x-4*px, y-px, 8*px, px, k)
	plot(x-3*px, y-6*px, 6*px, 5*px, k)
	plot(x-3*px, y-3*px, 6*px, px, color.RGBA{0xc8, 0x3a, 0x50, 255})
}

// magicTrick paints the effect of trick `kind` at local time u (0..magicTrickLen) around a magician whose feet are at (cx, cy).
func magicTrick(plot plotFn, cx, cy, px int, k func(float64) int, kind int, fail bool, u, t float64) {
	hatY := cy - k(41)
	if fail && u >= 3.0 {
		if u < 4.8 {
			drawSmokePix(plot, cx, hatY, px, (u-3.0)*0.9, 1)
		}
		if u < 5.8 {
			drawTopHatPix(plot, cx+k(14), cy+k(4), px)
		}
		return
	}
	switch kind {
	case mgRabbit:
		if u < 1.2 || u > 6.0 {
			return
		}
		v := u - 1.2
		switch {
		case v < 0.6: // up out of the hat
			drawRabbitPix(plot, cx, hatY-int(math.Round(math.Sin(v/0.6*math.Pi/2)*float64(k(9)))), px, t)
		case v < 1.2: // down to the floor
			f := (v - 0.6) / 0.6
			drawRabbitPix(plot, cx+int(f*float64(k(12))), hatY-k(9)+int(f*float64(cy-hatY+k(9))), px, t)
		default: // hops on the ground
			hop := int(math.Abs(math.Sin((v-1.2)*5)) * float64(k(5)))
			drawRabbitPix(plot, cx+k(12)+int((v-1.2)*float64(k(3))), cy-hop, px, t)
		}
	case mgScarves:
		if u < 1.0 || u > 6.0 {
			return
		}
		n := int((u - 1.0) * 6)
		if u > 4.6 {
			n = int((6.0 - u) * 9)
		}
		if n > 14 {
			n = 14
		}
		drawScarvesPix(plot, cx+k(9), cy-k(26), px, n, t)
	case mgCards:
		for i := 0; i < 5; i++ {
			v := u - 1.0 - 0.12*float64(i)
			if v < 0 || u > 6.0 {
				continue
			}
			f := math.Min(1, v/1.1)
			x := cx + int((float64(i)-2)*float64(k(9))*f)
			y := cy - k(26) + int(f*float64(k(26)+k(2)*(i%3))) - int(math.Sin(f*math.Pi)*float64(k(26)))
			drawCardPix(plot, x-1*px, y, px, i)
		}
	}
}

func magicItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*magicState)
	s := &st.show
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	pf := s.pf
	acting := s.phase == showAct
	if st.prop == nil && !st.dead {
		items = append(items, drawItem{pf.pos.Y, func() {
			anim, ph := AnimIdle, g.time*1.3
			if acting {
				u := math.Mod(g.time-s.t0, magicTrickLen)
				switch {
				case u < 1.2:
					anim, ph = AnimWave, g.time*3
				case u < 4.0:
					anim, ph = AnimPanic, g.time*2.5
				}
			}
			pf.draw(g, dst, anim, ph)
		}})
	}
	if acting {
		items = append(items, drawItem{pf.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(pf.pos)
			age := g.time - s.t0
			ti := int(age / magicTrickLen)
			if ti >= len(st.order) {
				ti = len(st.order) - 1
			}
			u := age - float64(ti)*magicTrickLen
			hk := k
			if st.prop != nil { // the decoration is a little shorter than a person
				hk = func(v float64) int { return int(math.Round(v * 0.9 * g.lifeScale())) }
			}
			magicTrick(plot, cx, cy, px, hk, st.order[ti], st.fail[ti], u, g.time)
			if st.pickEnd > 0 && !st.pickDone && st.pickWho != nil && g.time < st.pickEnd { // a "!" over the chosen one
				x, y := g.pevXY(st.pickWho.pos)
				by := y - k(46) - int(math.Abs(math.Sin(g.time*6))*float64(k(3)))
				plot(x-px, by, 2*px, 5*px, pevGold)
				plot(x-px, by+6*px, 2*px, 2*px, pevGold)
			}
		}})
	}
	items = showCoinItems(g, dst, s, items)
	items = showClapItems(g, dst, s, acting && !st.fail[minI(int(math.Max(0, (g.time-s.t0)/magicTrickLen)), len(st.order)-1)] && math.Mod(g.time-s.t0, magicTrickLen) > 4.0, items)
	return items
}
