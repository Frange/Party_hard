package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "bbq": a guest with a cook's hat and an apron grills on the barbecue of the level. Smoke rises from the
// grill. Half of the times the meat burns: a short flare-up, a scream, and the guests nearby come to look. The fire is
// only a drawing (about a second): the barbecue is not touched (it stays the trap it always was, nothing burns,
// nobody gets hurt). If the player uses the barbecue, or it is wrecked or burning, the event just ends.
//
// WHEN: ONLY in OUTDOOR parties that have a barbecue among their objects (the "bbq" prop) standing on open ground: the
// garden of the house party and the motorway rest area (2 of the 24), 30 % of those, between 40 and 200 s, up to twice. Not Big.
// ---------------------------------------------------------------------------

const (
	bbqWalkMax = 16.0
	bbqCookT   = 11.0
	bbqFlareT  = 1.3
	bbqAfterT  = 4.5
)

const (
	bqWalk = iota
	bqCook
	bqFlare
	bqAfter
)

var bbqLines = pevTexts("¡Barbacoa en marcha!", "¡AAAH! ¡MI CHULETÓN!", "¿Qué huele a quemado?", "¡Se quema, se quema!", "¡Cuidado, que sale fuego!",
	"Estaba al punto... por fuera.", "¡Qué bien huele!", "¡Yo quiero una!", "¡Ya casi está!", "Un poco crujiente, nada más.", "¡Apártense, soy profesional!")

type bbqState struct {
	pr    *Prop
	cook  *pevMem
	spot  Vec
	crowd pevCrowd
	phase int
	t0    float64
	burn  bool
	call  bool // the watchers have been called
	said  bool
}

func (b *bbqState) pevHas(gu *Guest) bool {
	return b.cook != nil && b.cook.gu == gu || b.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "bbq", Name: "Barbacoa", Group: "grill",
		Prob: 0.30, AtMin: 40, AtMax: 200, Dur: 30, Cooldown: 130, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool {
			return t.bbq && (t.outdoor || t.icon == "house") && t.roam && t.guests >= 12
		},
		Start: bbqStart, Items: bbqItems,
		Update: pevGuard(bbqUpdate, func(g *Game, a *partyAct) { bbqEnd(g, a.D.(*bbqState)) })})
}

func bbqUsable(pr *Prop) bool {
	return !pr.gone && !pr.used && pr.armT == 0 && !pr.armed && !pr.armWait && !pr.wreck && !pr.lit
}

func bbqStart(g *Game, a *partyAct) bool {
	r := partyDice()
	for _, pr := range g.lvl.Props {
		if pr.Name != "bbq" || !bbqUsable(pr) || !g.lvl.Outdoor(Vec{pr.X, pr.Y}) || !g.pevPlayerFar(Vec{pr.X, pr.Y}, 40) {
			continue
		}
		c := Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y + pr.Foot.H/2}
		var spot Vec
		found := false
		for _, o := range []Vec{{0, pr.Foot.H/2 + 9}, {-pr.Foot.W/2 - 8, pr.Foot.H / 2}, {pr.Foot.W/2 + 8, pr.Foot.H / 2}} {
			p := Vec{c.X + o.X, c.Y + o.Y}
			if g.lvl.Free(p) && !g.lvl.SolidP(p.X, p.Y) && !g.lvl.badGround(p) {
				spot, found = p, true
				break
			}
		}
		if !found {
			continue
		}
		// the cook: the free guest with the SHORTEST WALK (not the nearest as the crow flies: from inside a house the
		// grill behind a wall is a long way round), and only one who gets there in time (bbqWalkMax)
		var cook *Guest
		best := guestSpeed * bbqWalkMax * 0.8
		for _, gu := range g.guests {
			if !g.pevFreeGuest(gu) || dist(gu.pos, spot) >= best {
				continue
			}
			path := g.lvl.PathG(gu.pos, spot)
			n := pevPathLen(path, gu.pos)
			if path == nil && dist(gu.pos, spot) <= 6 { // already there
				n = dist(gu.pos, spot)
			}
			if n < best {
				cook, best = gu, n
			}
		}
		if cook == nil {
			continue
		}
		m := &pevMem{gu: cook}
		if !g.pevSendTo(m, spot) {
			continue
		}
		cook.icon, cook.iconT = bbqLines[10], 2.4
		a.D = &bbqState{pr: pr, cook: m, spot: spot, t0: g.time, burn: r.Float64() < 0.5}
		return true
	}
	return false
}

func bbqEnd(g *Game, st *bbqState) {
	r := partyDice()
	pevRelease(st.cook, r)
	st.crowd.release(g, r)
}

func (st *bbqState) grillTop() Vec { return Vec{st.pr.X, st.pr.Y - 24} }

// bbqCall: some guests nearby come over to the cook.
func bbqCall(g *Game, st *bbqState, n int) {
	st.call = true
	st.crowd.gather(g, partyDice(), st.spot, n, 20, 30, func(gu *Guest) bool { return gu == st.cook.gu })
}

func bbqUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*bbqState)
	r := partyDice()
	age := g.time - st.t0
	if !st.cook.mine() || !bbqUsable(st.pr) {
		bbqEnd(g, st)
		return true
	}
	st.cook.keep()
	st.crowd.update(g, r, nil, 0, nil)
	cg := st.cook.gu
	switch st.phase {
	case bqWalk:
		if cg.state == gIdle && cg.timer > 1000 {
			st.cook.want = gIdle
			cg.dir = faceTo(cg.pos, Vec{st.pr.X, st.pr.Y - 8})
			st.phase, st.t0 = bqCook, g.time
			g.say(bbqLines[0])
		} else if age > bbqWalkMax {
			bbqEnd(g, st)
			return true
		}
	case bqCook:
		cg.dir = faceTo(cg.pos, Vec{st.pr.X, st.pr.Y - 8})
		if !st.burn && !st.call && age > bbqCookT*0.45 {
			bbqCall(g, st, 2)
			for _, m := range st.crowd.mem {
				m.gu.icon, m.gu.iconT = pevPick(r, bbqLines[6:8]), 2.4
			}
		}
		if !st.said && age > bbqCookT*0.5 && cg.icon == "" {
			st.said = true
			cg.icon, cg.iconT = bbqLines[8], 2
		}
		if age >= bbqCookT*0.75 && st.burn {
			st.phase, st.t0 = bqFlare, g.time
			st.cook.pose(AnimPanic)
			cg.icon, cg.iconT = bbqLines[1], 2.6
			g.snd.Play("crackle")
		} else if age >= bbqCookT && !st.burn {
			bbqEnd(g, st)
			return true
		}
	case bqFlare:
		if age >= bbqFlareT && !st.call {
			bbqCall(g, st, 3)
			for _, m := range st.crowd.mem {
				m.gu.icon, m.gu.iconT = pevPick(r, bbqLines[2:5]), 2.4
			}
		}
		if age >= bbqFlareT+0.5 {
			st.phase, st.t0 = bqAfter, g.time
			st.cook.hold(gIdle)
			cg.dir = faceTo(cg.pos, Vec{st.pr.X, st.pr.Y - 8})
			cg.icon, cg.iconT = pevPick(r, bbqLines[5:6:6]), 3
			if r.Float64() < 0.5 {
				cg.icon = bbqLines[9]
			}
		}
	case bqAfter:
		if age >= bbqAfterT {
			bbqEnd(g, st)
			return true
		}
	}
	return false
}

// drawSmokePix: gray puffs that rise from (x, y) and fade; k 0..1 is how much smoke.
func drawSmokePix(plot plotFn, x, y, px int, t, k float64) {
	for i := 0; i < 6; i++ {
		ph := math.Mod(t*0.7+float64(i)/6, 1)
		sx := x + int(math.Round(math.Sin(ph*4+float64(i)*1.7)*3))*px
		sy := y - int(math.Round(ph*26))*px
		a := uint8(math.Max(0, (1-ph)*150*k))
		if a == 0 {
			continue
		}
		g := uint8(0xc8 * uint16(a) / 255)
		sz := 2 + int(ph*3)
		plot(sx-sz*px/2, sy-sz*px/2, sz*px, sz*px, color.RGBA{g, g, g, a})
	}
}

// drawFlamePix: a flare-up of ~1 s over (x, y): orange tongues with a yellow core; u runs 0..1 over its life.
func drawFlamePix(plot plotFn, x, y, px int, t, u float64) {
	h := int(math.Round(math.Sin(math.Pi*math.Min(1, u))*14)) + 2
	for i := -2; i <= 2; i++ {
		w := 1 + (i+2)%2
		hh := h - absI(i)*2 + int(math.Round(math.Sin(t*20+float64(i))*1.5))
		if hh < 2 {
			continue
		}
		// layered like the other fires (fire_art.go): dark red rim, red, orange, yellow
		plot(x+i*2*px-px, y-hh*px+px, w*px+3*px, hh*px-px, fireRamp[5])
		plot(x+i*2*px, y-hh*px, w*px+px, hh*px, fireRamp[4])
		plot(x+i*2*px, y-(hh*4/5)*px, w*px+px, (hh*4/5)*px, fireRamp[3])
		plot(x+i*2*px, y-(hh*3/5)*px, w*px, (hh*3/5)*px, fireRamp[1])
	}
	plot(x-px, y-h/3*px, 3*px, h/3*px, fireRamp[0])
	for i := 0; i < 3; i++ { // sparks that rise and die out
		v := fireFrac(t*1.3 + float64(i)/3)
		if v > 0.85 {
			continue
		}
		col := fireRamp[1]
		if v > 0.5 {
			col = fireRamp[4]
		}
		sx := x + int(math.Round(math.Sin(float64(i)*2.1+v*5)*5))*px
		plot(sx, y-(h+2+int(v*12))*px, px, px, col)
	}
}

// drawCookKitPix: the cook's hat and apron on a guest whose feet are at (cx, cy); headTop is the top of his head.
func drawCookKitPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	drawApronPix(plot, cx, cy, px, k)
	drawChefHatPix(plot, cx, cy-k(31), px)
}

func bbqItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*bbqState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	cg := st.cook.gu
	t := g.time - st.t0
	if st.phase >= bqCook {
		items = append(items, drawItem{st.pr.Y + 0.4, func() { // smoke and flames over the barbecue
			x, y := g.pevXY(st.grillTop())
			smoke := 1.0
			if st.phase == bqAfter {
				smoke = math.Max(0.3, 1-t/bbqAfterT*0.7)
			}
			drawSmokePix(plot, x, y, px, g.time, smoke)
			if st.phase == bqFlare && t < bbqFlareT {
				drawFlamePix(plot, x, y+px, px, g.time, t/bbqFlareT)
			}
		}})
	}
	items = append(items, drawItem{cg.pos.Y + 0.2, func() {
		cx, cy := g.pevXY(cg.pos)
		drawCookKitPix(plot, cx, cy, px, k)
		if st.phase == bqCook || st.phase == bqAfter {
			gx, gy := g.pevXY(st.grillTop())
			sway := int(math.Round(math.Sin(g.time*3) * 2))
			side := 1
			if gx < cx {
				side = -1
			}
			hx, hy := cx+side*k(4), cy-k(22)
			drawLinePix(plot, hx, hy, gx+sway*px-side*3*px, gy+px*2, px, cg.cs.Spec.Skin)
			plot(gx+sway*px-side*3*px-px, gy+px, 3*px, px, color.RGBA{0xb8, 0xb8, 0xc4, 255})
		}
	}})
	return items
}
