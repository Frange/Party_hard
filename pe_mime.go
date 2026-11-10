package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "mime": a mime (white face, black beret, striped shirt, red braces) walks into an open spot and acts out
// being trapped in an invisible glass box: his white gloves slide along the panes, tap, search for the way out. Three or
// four guests stop to watch and toss him coins. If a player comes close he "speaks" with gestures: speech bubbles
// that only have symbols. No points, nothing to pick up. He walks away when he is done.
//
// WHEN: open-air parties with a street or square feel (Paris, fair, festival, beach), 22 % of them, between 50 and
// 210 s, up to twice. Group "street": never together with the juggler or the buskers.
// ---------------------------------------------------------------------------

var mimeIcons = map[string]bool{"eiffel": true, "funfair": true, "festival": true, "beach": true}

const (
	mimeActLen = 19.0
	mimeNear   = 46.0
)

var mimeLines = pevTexts("?!", "...", "!!", "¿?", ":)", "¡Bravo!", "¡Qué arte!", "¡Qué buen mimo!")

type mimeState struct {
	show   pevShow
	talkAt float64
	talk   int

	cheered int
}

func (m *mimeState) pevHas(gu *Guest) bool { return m.show.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "mime", Name: "Mimo", Group: "street",
		Prob: 0.22, AtMin: 50, AtMax: 210, Dur: 40, Cooldown: 140, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool { return t.outdoor && mimeIcons[t.icon] && t.roam && t.guests >= 16 },
		Start:    mimeStart, Items: mimeItems,
		Update: pevGuard(mimeUpdate, func(g *Game, a *partyAct) { a.D.(*mimeState).show.end(g) })})
}

func mimeSpec() CharSpec {
	return CharSpec{
		Skin: pevSkinW, Hair: hx(0x16121c), Shirt: hx(0xf4f2ee), Shirt2: hx(0x16121c), Pants: hx(0x16121c), Shoes: hx(0x16121c),
		Accent: hx(0xf4f2ee), HatCol: hx(0x16121c), Hat: HatBeret, Top: TopLong, Pattern: 2, Susp: hx(0xd03030), Makeup: true, noOverride: true,
	}
}

func mimeStart(g *Game, a *partyAct) bool {
	s, ok := g.pevShowStart(mimeSpec(), mimeActLen, true)
	if !ok {
		return false
	}
	a.D = &mimeState{show: *s}
	return true
}

func mimeUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*mimeState)
	r := partyDice()
	prev := st.show.phase
	done := g.pevShowStep(&st.show, dt, 3+r.Intn(2), 3.4, nil)
	if done {
		st.show.end(g)
		return true
	}
	if st.show.phase == showAct && prev == showEnter {
		st.show.pf.dir = DirDown
	}
	if st.show.phase == showAct {
		for _, pl := range g.players() {
			if pl != nil && !pl.dead && dist(pl.pos, st.show.pf.pos) < mimeNear && g.time >= st.talkAt {
				st.talkAt = g.time + 2.6
				st.talk = (st.talk + 1) % 5
				break
			}
		}
		age := g.time - st.show.t0
		if ph := int(age / 7); math.Mod(age, 7) > 5.2 && ph >= st.cheered {
			st.cheered = ph + 1
			st.show.cheer(r, mimeLines[5:])
		}
	}
	return false
}

// mimeHands: where the two gloves are (sprite pixels relative to the feet, x and y) at time t: they slide along the glass,
// stop and tap.
func mimeHands(t float64) (lx, ly, rx, ry float64, tap int) {
	a := math.Mod(t, 7) // the routine lasts seven seconds
	switch {
	case a < 2: // both palms flat on the front pane, sliding up and down
		return -8, -24 + 3*math.Sin(a*3), 8, -24 - 3*math.Sin(a*3), 0
	case a < 4: // he feels the left wall
		return -12, -27 + 2*math.Sin(a*5), 6, -22, 1
	case a < 5.5: // and the right one
		return -6, -22, 12, -27 + 2*math.Sin(a*5), 2
	}
	return -8 + 3*math.Sin(a*2.4), -30 - 3*math.Abs(math.Sin(a*4)), 8 - 3*math.Sin(a*2.4), -30 - 3*math.Abs(math.Cos(a*4)), 0 // pushing the lid
}

// drawMimeBoxPix: the invisible box: dotted pale edges of a cube around the mime (back=true draws the rear half).
func drawMimeBoxPix(plot plotFn, cx, cy, px int, k func(float64) int, back bool, t float64) {
	ec := rgba(0x9ad8f0, 120)
	w, h := k(15), k(46)
	dx, dy := k(5), -k(5)
	x0, y0 := cx-w, cy-h+k(2)
	x1, y1 := cx+w, cy+k(2)
	dot := func(x, y, l int, horiz bool) {
		for i := 0; i < l; i += 2 * px {
			if horiz {
				plot(x+i, y, px, px, ec)
			} else {
				plot(x, y+i, px, px, ec)
			}
		}
	}
	if back {
		dot(x0+dx, y0+dy, x1-x0, true)
		dot(x0+dx, y1+dy, x1-x0, true)
		dot(x0+dx, y0+dy, y1-y0, false)
		dot(x1+dx, y0+dy, y1-y0, false)
		return
	}
	dot(x0, y0, x1-x0, true)
	dot(x0, y1, x1-x0, true)
	dot(x0, y0, y1-y0, false)
	dot(x1, y0, y1-y0, false)
	drawLinePix(plot, x0, y0, x0+dx, y0+dy, px, ec) // the edges that go back
	drawLinePix(plot, x1, y0, x1+dx, y0+dy, px, ec)
	drawLinePix(plot, x1, y1, x1+dx, y1+dy, px, ec)
	// a glint that slides over the glass
	gx := x0 + k(3) + int(math.Mod(t*18, float64(x1-x0-k(6))))
	plot(gx, y0+k(5), px, 3*px, color.RGBA{0xf0, 0xf8, 0xff, 255})
}

// drawGlovePix: a white glove (3x3 sprite pixels) with a black outline at its centre.
func drawGlovePix(plot plotFn, x, y, px int) {
	plot(x-2*px, y-2*px, 5*px, 5*px, pevInk)
	plot(x-px, y-px, 3*px, 3*px, pevWhite)
}

// drawMimeFrontPix: the gloves on the glass and the little flash of the tap.
func drawMimeFrontPix(plot plotFn, cx, cy, px int, k func(float64) int, t float64) {
	lx, ly, rx, ry, tap := mimeHands(t)
	drawGlovePix(plot, cx+k(lx), cy+k(ly), px)
	drawGlovePix(plot, cx+k(rx), cy+k(ry), px)
	if tap > 0 && int(t*6)%3 == 0 {
		x, y := cx+k(lx)-4*px, cy+k(ly)
		if tap == 2 {
			x, y = cx+k(rx)+4*px, cy+k(ry)
		}
		white := color.RGBA{0xff, 0xff, 0xff, 255}
		plot(x, y-2*px, px, 4*px, white)
		plot(x-px, y-px, 3*px, 2*px, white)
	}
}

func mimeItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*mimeState)
	s := &st.show
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	pf := s.pf
	acting := s.phase == showAct
	items = append(items, drawItem{pf.pos.Y - 0.1, func() {
		if acting {
			cx, cy := g.pevXY(pf.pos)
			drawMimeBoxPix(plot, cx, cy, px, k, true, g.time)
		}
	}})
	items = append(items, drawItem{pf.pos.Y, func() {
		pf.draw(g, dst, AnimIdle, g.time*1.3)
	}})
	items = append(items, drawItem{pf.pos.Y + 0.1, func() {
		if !acting {
			return
		}
		cx, cy := g.pevXY(pf.pos)
		drawMimeBoxPix(plot, cx, cy, px, k, false, g.time)
		drawMimeFrontPix(plot, cx, cy, px, k, g.time-s.t0)
		drawCoinPilePix(plot, cx+k(16), cy, px, s.tips)
		near := false
		for _, pl := range g.players() {
			if pl != nil && !pl.dead && dist(pl.pos, pf.pos) < mimeNear {
				near = true
			}
		}
		if near {
			g.bubble(pf, bubGuest, pf.pos, mimeLines[st.talk%5], false, 0)
		}
	}})
	items = showCoinItems(g, dst, s, items)
	items = showClapItems(g, dst, s, acting && math.Mod(g.time-s.t0, 7) > 5.2, items)
	return items
}
