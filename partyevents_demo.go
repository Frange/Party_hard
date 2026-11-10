package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// DEMO EVENTS of the party-event frame (partyevents.go). Two cheap ones to prove the frame; THE REST OF THE EVENTS
// (conga, magician, stray dog, waiters, skaters, mime, reporter...) ARE STILL TO BE ADDED, each in its own file.
//
//  1. "toast" (host's toast): a free guest stands on a crate, raises a glass and gives a three-line speech; the
//     guests near him clap. Nobody is moved: the speaker is held where he is (state idle) and released at the end;
//     if he is hurt, killed, or the player pushes things around, the toast just ends.
//  2. "emergency" (emergency lights): while the blackout lasts, red lamps blink at the doors and the fuse box and a
//     faint red wash pulses over the dark. Reactive: it follows the blackout (blackout.go), it rolls no dice.
//  3. Adapters of partylife.go: "selfie", "contest", "pizza" (External: listed, never scheduled here).
// ---------------------------------------------------------------------------

const (
	toastNear    = 62.0 // who claps
	toastNeedN   = 3    // free guests around the speaker, at least
	toastAbort   = 14.0 // the speaker was moved farther than this: it ends
	toastLineAt  = 1.6  // seconds to climb the crate
	toastLineGap = 2.6
)

var toastLines = [][]string{
	{"¡Amigos, un momento!", "¡Silencio, que hablo yo!", "¡Atención, por favor!"},
	{"¡Gracias por venir a mi fiesta!", "¡Esta es la mejor noche del año!", "¡Quien baile peor, paga!"},
	{"¡Por nosotros! ¡Chin chin!", "¡Por la música y el queso!", "¡Que no pare la fiesta!"},
}

var toastClaps = []string{"¡Bravo!", "¡Eso!", "¡Viva!", "¡Otra!"}

type toastState struct {
	sp     *Guest
	anchor Vec
	lines  [3]string
	near   []*Guest
	cheer  bool
}

func init() {
	for _, ls := range toastLines {
		for _, l := range ls {
			if _, ok := langDataEN[l]; !ok {
				langDataEN[l] = toastEN[l]
			}
		}
	}
	for _, c := range toastClaps {
		if _, ok := langDataEN[c]; !ok {
			langDataEN[c] = toastEN[c]
		}
	}
	registerPartyEvent(&partyEvent{ID: "toast", Name: "Brindis del anfitrión", Group: "speech", Big: true,
		Prob: 0.25, AtMin: 50, AtMax: 220, Dur: 10, Cooldown: 140, MaxRuns: 2, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return (t.indoor || t.bar) && t.roam && t.guests >= 20 },
		Start:    toastStart, Update: toastUpdate, Items: toastItems})
	registerPartyEvent(&partyEvent{ID: "emergency", Name: "Luces de emergencia", Group: "light", Reactive: true, DailyOK: true,
		Prob: 1, AtMin: 0, AtMax: 0, Dur: 12, Prio: 9,
		Eligible: func(g *Game, t partyTraits) bool { return g.blackoutOn() },
		Start:    func(g *Game, a *partyAct) bool { return true },
		Update:   func(g *Game, a *partyAct, dt float64) bool { return !g.blackoutOn() },
		Over:     emergencyOver})
	// adapters: partylife.go keeps its dice and frequencies (selfieChance..., contestChance 0.30, pizzaChance 1/6)
	registerPartyEvent(&partyEvent{ID: "selfie", Name: "Selfies (partylife.go)", Group: "", Prob: 1, Dur: selfieDur,
		External: func(g *Game) bool { return g.life.ready && g.life.selfieT > 0 }})
	registerPartyEvent(&partyEvent{ID: "contest", Name: "Concurso de baile (partylife.go)", Group: "dance", Big: true, Prob: contestChance,
		AtMin: 60, AtMax: 200, Dur: contestGather + contestDance + contestWin,
		External: func(g *Game) bool { return g.life.contestAt > 0 }})
	registerPartyEvent(&partyEvent{ID: "pizza", Name: "Pizzero equivocado (partylife.go)", Group: "", Prob: pizzaChance,
		AtMin: 35, AtMax: 155, Dur: pizzaWaitMax,
		External: func(g *Game) bool { return g.life.pizzaAt > 0 }})
}

var toastEN = map[string]string{
	"¡Amigos, un momento!": "Friends, one moment!", "¡Silencio, que hablo yo!": "Quiet, I am talking!", "¡Atención, por favor!": "Attention, please!",
	"¡Gracias por venir a mi fiesta!": "Thanks for coming to my party!", "¡Esta es la mejor noche del año!": "Best night of the year!",
	"¡Quien baile peor, paga!": "Worst dancer pays!", "¡Por nosotros! ¡Chin chin!": "To us! Cheers!",
	"¡Por la música y el queso!": "To music and cheese!", "¡Que no pare la fiesta!": "Keep the party going!",
	"¡Bravo!": "Bravo!", "¡Eso!": "That's it!", "¡Viva!": "Hooray!", "¡Otra!": "Encore!",
}

func toastGuestOK(g *Game, gu *Guest) bool {
	return lifeFree(gu) && gu.state == gIdle && !g.inSelfie(gu) && !g.inContest(gu)
}

func toastStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Guest // idle guests: the speaker; any free guest around may clap
	for _, gu := range g.guests {
		if toastGuestOK(g, gu) {
			cand = append(cand, gu)
		}
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	for _, sp := range cand {
		n := 0
		for _, o := range g.guests {
			if o != sp && lifeFree(o) && dist(o.pos, sp.pos) < toastNear {
				n++
			}
		}
		if n < toastNeedN {
			continue
		}
		st := &toastState{sp: sp, anchor: sp.pos}
		for i, ls := range toastLines {
			st.lines[i] = ls[r.Intn(len(ls))]
		}
		sp.state, sp.timer, sp.path, sp.moving = gIdle, 9999, nil, false
		a.D = st
		return true
	}
	return false
}

// toastEnd releases the speaker (idle again, he decides what to do next) and the ones who were clapping.
func toastEnd(g *Game, st *toastState) {
	if st.sp != nil && !st.sp.out() && st.sp.state == gIdle && st.sp.timer > 1000 {
		st.sp.timer = 0.4
	}
}

func toastUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*toastState)
	t := g.time - a.t0
	sp := st.sp
	if sp == nil || sp.out() || sp.state != gIdle || dist(sp.pos, st.anchor) > toastAbort {
		toastEnd(g, st)
		return true
	}
	st.near = st.near[:0]
	for _, o := range g.guests {
		if o != sp && lifeFree(o) && dist(o.pos, sp.pos) < toastNear {
			st.near = append(st.near, o)
		}
	}
	if !st.cheer && t >= toastLineAt+toastLineGap*2+0.8 { // after the last line: one of them shouts
		st.cheer = true
		if len(st.near) > 0 {
			o := st.near[int(math.Abs(sp.pos.X))%len(st.near)]
			o.icon, o.iconT = toastClaps[int(math.Abs(sp.pos.Y))%len(toastClaps)], 2.2
		}
	}
	if t >= toastLineAt+toastLineGap*3+1.2 {
		toastEnd(g, st)
		return true
	}
	return false
}

// toastLine: the line that is on now ("" before the speech).
func toastLine(st *toastState, t float64) string {
	i := int((t - toastLineAt) / toastLineGap)
	if t < toastLineAt || i < 0 || i > 2 {
		return ""
	}
	return st.lines[i]
}

var (
	crateTop  = color.RGBA{0xb0, 0x7a, 0x3c, 255}
	crateSide = color.RGBA{0x84, 0x58, 0x28, 255}
	crateDark = color.RGBA{0x4a, 0x2e, 0x14, 255}
	glassCol  = color.RGBA{0xd8, 0xf2, 0xff, 255}
	wineCol   = color.RGBA{0xd2, 0x30, 0x50, 255}
	clapCol   = color.RGBA{0xff, 0xe0, 0x70, 255}
)

// drawCratePix: a crate of 16x8 (at scale 1) with its feet at (cx, cy).
func drawCratePix(plot plotFn, cx, cy int, px int) {
	w, h := 16*px, 8*px
	x0, y0 := cx-w/2, cy-h+2*px
	plot(x0-px, y0-px, w+2*px, h+2*px, crateDark)
	plot(x0, y0, w, 3*px, crateTop)
	plot(x0, y0+3*px, w, h-3*px, crateSide)
	plot(x0+w/2-px/2, y0+3*px, px, h-3*px, crateDark)
}

// drawGlassPix: a raised glass (cheers) at the hand position.
func drawGlassPix(plot plotFn, x, y, px int) {
	plot(x-px, y-px, 3*px, 3*px, glassCol)
	plot(x, y, px, px, wineCol)
	plot(x, y+2*px, px, 2*px, glassCol)
	plot(x-px, y+4*px, 3*px, px, glassCol)
}

func toastItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*toastState)
	sp := st.sp
	if sp == nil || sp.out() {
		return items
	}
	plot := g.plotter(dst)
	sc := g.lifeScale()
	px := int(math.Max(1, math.Round(sc)))
	// the crate goes behind his feet; the glass and the speech in front
	items = append(items, drawItem{sp.pos.Y - 0.1, func() {
		drawCratePix(plot, int(math.Round(g.wx(sp.pos.X))), int(math.Round(g.wy(sp.pos.Y))), px)
	}})
	items = append(items, drawItem{sp.pos.Y + 0.1, func() {
		t := g.time - a.t0
		cx, cy := int(math.Round(g.wx(sp.pos.X))), int(math.Round(g.wy(sp.pos.Y)))
		if t >= toastLineAt {
			up := 0
			if int(t*4)%2 == 0 {
				up = px
			}
			drawGlassPix(plot, cx+7*px, cy-int(40*sc)-up, px)
		}
		if l := toastLine(st, t); l != "" {
			g.bubble(sp, bubGuest, sp.pos, l, false, 0)
		}
		for i, o := range st.near { // clapping hands: two sparks that open and close
			if t < toastLineAt+0.6 {
				break
			}
			ox, oy := int(math.Round(g.wx(o.pos.X))), int(math.Round(g.wy(o.pos.Y)))-int(34*sc)
			gap := 2 * px
			if (int(t*6)+i)%2 == 0 {
				gap = 5 * px
			}
			plot(ox-gap-px, oy, px, 2*px, clapCol)
			plot(ox+gap, oy, px, 2*px, clapCol)
		}
	}})
	return items
}

// emergencyOver: red lamps at the doors and the fuse box, blinking, and a faint red wash over the dark.
func emergencyOver(g *Game, dst *ebiten.Image, a *partyAct) {
	k := g.blackoutK()
	if k <= 0 || g.lvl == nil {
		return
	}
	on := math.Mod(g.time*1.6, 1) < 0.55
	pulse := 0.5 + 0.5*math.Sin(g.time*5)
	fillRect(dst, 0, 0, screenW, screenH, rgba(0xff1010, uint8(18+22*pulse*k)))
	lamps := []Vec{g.blk.panel}
	for _, d := range g.lvl.Def.Doors {
		lamps = append(lamps, Vec{d.R.X + d.R.W/2, d.R.Y})
	}
	for _, p := range g.lvl.Lights {
		if len(lamps) >= 8 {
			break
		}
		lamps = append(lamps, p)
	}
	for _, p := range lamps {
		x, y := g.wx(p.X), g.wy(p.Y)
		if x < -20 || x > screenW+20 || y < -20 || y > screenH+20 {
			continue
		}
		fillRect(dst, x-3, y-3, 6, 5, rgb(0x2a0808))
		c := rgb(0x6a1010)
		if on {
			c = rgb(0xff3030)
			if g.as != nil && g.as.glow != nil {
				g.glowAt(dst, p, 34, rgba(0xff2020, 150))
			}
		}
		fillRect(dst, x-2, y-2, 4, 3, c)
	}
}
