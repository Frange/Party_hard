package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "busker": one or two street musicians (a guitarist, a singer with a microphone, or both) set up in an
// open spot of the street or square, with a hat on the ground. Notes float up from them, they play with their own
// rhythm (the sway of the animation, no new audio), some guests stop to listen and toss coins into the hat, and
// they clap at the end. They leave when they are done. No points, nothing to pick up.
//
// WHEN: open-air parties with a street, square or beach feel (Paris, fair, festival, beach) and the ski resort
// terrace; 25 % of them, between 40 and 200 s, up to twice. Group "street".
// ---------------------------------------------------------------------------

var buskerIcons = map[string]bool{"eiffel": true, "funfair": true, "festival": true, "beach": true, "snow": true}

const buskActLen = 26.0

var buskerLines = pevTexts("¡Qué bien toca!", "¡Toca otra!", "¡Bravo!", "¡Eso es música!", "¡Qué voz!", "Una moneda para el artista", "¡Otra, otra!", "Me ha puesto la piel de gallina")

type buskerState struct {
	show    pevShow
	second  *pevPerf // the singer when there are two
	solo    int      // 0 = guitar, 1 = singer (only when alone)
	duo     bool
	prev    int
	cheered int
}

func (b *buskerState) pevHas(gu *Guest) bool { return b.show.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "busker", Name: "Músicos callejeros", Group: "street",
		Prob: 0.25, AtMin: 40, AtMax: 200, Dur: 40, Cooldown: 140, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool { return t.outdoor && buskerIcons[t.icon] && t.roam && t.guests >= 16 },
		Start:    buskerStart, Items: buskerItems,
		Update: pevGuard(buskerUpdate, func(g *Game, a *partyAct) { a.D.(*buskerState).show.end(g) })})
}

func guitaristSpec() CharSpec {
	return CharSpec{
		Skin: hx(0xd8a070), Hair: hx(0x3a2418), Shirt: hx(0xb83a2a), Shirt2: hx(0x6a1c14), Pants: hx(0x3a4a6a), Shoes: hx(0x2a1c14),
		Accent: hx(0xe8d8b0), HatCol: hx(0x4a3020), Hat: HatFedora, Top: TopTee, Pattern: 3, Hair2: HairLong, noOverride: true,
	}
}

func singerSpec() CharSpec {
	return CharSpec{
		Skin: hx(0xf0c8a0), Hair: hx(0xe0a830), Shirt: hx(0x8a3ad0), Shirt2: hx(0xd8a0f0), Pants: hx(0x16121c), Shoes: hx(0xf0f0f0),
		Accent: hx(0xf0e0a0), HatCol: hx(0x8a3ad0), Hat: HatNone, Top: TopLong, Pattern: 0, Hair2: HairBob, Female: true, Makeup: true, noOverride: true,
	}
}

func buskerStart(g *Game, a *partyAct) bool {
	r := partyDice()
	st := &buskerState{}
	roll := r.Float64()
	st.duo = roll < 0.4
	if !st.duo && roll > 0.7 {
		st.solo = 1
	}
	spec := guitaristSpec()
	if st.solo == 1 {
		spec = singerSpec()
	}
	s, ok := g.pevShowStart(spec, buskActLen, true)
	if !ok {
		return false
	}
	s.tip = Vec{s.spot.X, s.spot.Y + 9}
	st.show = *s
	if st.duo {
		sp := Vec{s.spot.X + 24, s.spot.Y}
		if g.pevClear(sp, 8) {
			if path := g.lvl.PathG(s.pf.entry, sp); path != nil {
				st.second = newPevPerf(singerSpec(), s.pf.entry, DirDown)
				st.second.path = path
			}
		}
		if st.second == nil {
			st.duo = false
		}
	}
	a.D = st
	return true
}

func buskerUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*buskerState)
	r := partyDice()
	n := 3 + r.Intn(2)
	if st.duo {
		n++
	}
	done := g.pevShowStep(&st.show, dt, n, 3.6, nil)
	if st.second != nil {
		if st.show.phase == showEnter || st.show.phase == showLeave {
			st.second.step(dt)
		}
		if st.prev != st.show.phase {
			switch st.show.phase {
			case showAct:
				st.second.dir = DirDown
			case showLeave:
				st.second.path = g.lvl.PathG(st.second.pos, st.second.entry)
			}
		}
	}
	st.prev = st.show.phase
	if done {
		st.show.end(g)
		return true
	}
	if st.show.phase == showAct {
		age := g.time - st.show.t0
		if ph := int(age / 6.5); ph >= st.cheered && ph > 0 {
			st.cheered = ph + 1
			st.show.cheer(r, buskerLines[:5])
		}
	}
	return false
}

// drawGroundHatPix: a hat lying upside down on the ground with the coins in it (n up to 9), centred at (x, y).
func drawGroundHatPix(plot plotFn, x, y, px, n int) {
	dark, rim, band := color.RGBA{0x3a, 0x24, 0x14, 255}, color.RGBA{0x8a, 0x5c, 0x2e, 255}, color.RGBA{0xc8, 0x40, 0x38, 255}
	plot(x-7*px, y-px, 15*px, 3*px, dark) // the brim
	plot(x-6*px, y-px, 13*px, px, rim)
	plot(x-4*px, y-5*px, 9*px, 4*px, dark) // the crown, open side up: the coins show in it
	plot(x-3*px, y-4*px, 7*px, 2*px, color.RGBA{0x16, 0x0e, 0x08, 255})
	plot(x-4*px, y-2*px, 9*px, px, band)
	if n > 9 {
		n = 9
	}
	for i := 0; i < n; i++ {
		plot(x-3*px+(i%5)*3*px/2, y-5*px-(i/5)*px, 2*px, px, pevGold)
	}
}

func buskerItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*buskerState)
	s := &st.show
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	acting := s.phase == showAct
	anim := AnimGuitar
	if st.solo == 1 {
		anim = AnimSing
	}
	pf := s.pf
	items = append(items, drawItem{pf.pos.Y, func() { pf.draw(g, dst, anim, g.time*5) }})
	if st.second != nil {
		p2 := st.second
		items = append(items, drawItem{p2.pos.Y, func() { p2.draw(g, dst, AnimSing, g.time*4) }})
	}
	if acting {
		items = append(items, drawItem{pf.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(pf.pos)
			drawNotesPix(plot, cx, cy-k(48), px, g.time, 3)
			hx, hy := g.pevXY(s.tip)
			drawGroundHatPix(plot, hx, hy, px, s.tips)
			if st.second != nil {
				x2, y2 := g.pevXY(st.second.pos)
				drawNotesPix(plot, x2, y2-k(48), px, g.time+1.7, 2)
			}
		}})
	}
	items = showCoinItems(g, dst, s, items)
	items = showClapItems(g, dst, s, acting && math.Mod(g.time-s.t0, 6.5) > 5.0, items)
	return items
}
