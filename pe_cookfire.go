package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "cookfire" (COCINERO CON FUEGO): a guest in a cook's hat cooks at the stove, the barbecue or the pizza oven of
// the level; the oil spills and a flare-up turns into a REAL fire (rule of the owner: "que sí que dañe, estilo el
// pirómano y tengan que llamar a los bomberos"). It is the fire of the arsonist (fire.go), not a copy:
//   - one burning spot in front of the cooker (a Fire without a prop), that spreads a little to the flammable props
//     around it with the usual budget of an ignition;
//   - it burns guests and players who stay in it (fireHurt) and frightens the ones nearby, who run;
//   - after a few seconds somebody calls the firefighters (updateFires) and the engine (or the crew on foot) comes and
//     hoses it down (firefighters.go);
//   - if they never come, the flames die by themselves (fireLifeMin..Max, here shorter): the safety cap.
// The deaths of this fire are nobody's work (no stars, no points for anybody), as with the arsonist; they are bodies.
// At most ONE fire at a time: it never starts while anything burns in the level.
//
// WHEN: parties with a cooker (stove, barbecue or pizza oven: the house, the motorway rest area, Italy...), 15 % of
// them, between 60 and 240 s, once. Group "grill" (never together with the barbecue event). Never in the RETO DE HOY.
// ---------------------------------------------------------------------------

const (
	cookWalkMax = 18.0
	cookT       = 7.5 // seconds cooking before the oil spills
	cookFlareT  = 1.2
	cookFireMin = 18.0 // life of the flames when nobody puts them out (the safety cap)
	cookFireMax = 26.0
)

var cookLines = pevTexts("¡Hoy cocino yo!", "Un chorrito de aceite...", "¡Uy! ¡EL ACEITE!", "¡FUEGO EN LA COCINA!", "¡Llamad a los bomberos!",
	"¡Mi sartén!", "¡Gracias, bomberos!", "Yo solo quería freír un huevo")

const (
	ckWalk = iota
	ckCook
	ckFlare
	ckBurn
)

type cookState struct {
	pr    *Prop
	cook  *pevMem
	spot  Vec
	phase int
	t0    float64
	fire  *Fire
	said  bool
}

func (s *cookState) pevHas(gu *Guest) bool {
	return s.cook != nil && s.cook.gu == gu && s.phase <= ckFlare
}

func init() {
	registerPartyEvent(&partyEvent{ID: "cookfire", Name: "Cocinero con fuego", Group: "grill",
		Prob: 0.15, AtMin: 60, AtMax: 240, Dur: 60,
		Eligible: func(g *Game, t partyTraits) bool { return t.cook && t.roam && t.guests >= 12 },
		Start:    cookStart, Items: cookItems,
		Update: pevGuard(cookUpdate, func(g *Game, a *partyAct) { cookEnd(g, a.D.(*cookState)) })})
}

func cookUsable(pr *Prop) bool {
	return (pr.Name == "stove" || pr.Name == "bbq" || pr.Name == "pizzaoven") && bbqUsable(pr) && pr.burn == 0
}

func cookStart(g *Game, a *partyAct) bool {
	if g.liveFires() > 0 || g.fireTruck != nil || len(g.fireCrew) > 0 { // at most one fire at a time
		return false
	}
	for _, pr := range g.lvl.Props {
		if !cookUsable(pr) || !pevDebugOn("cookfire") && !g.pevPlayerFar(Vec{pr.X, pr.Y}, 40) {
			continue
		}
		c := Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y + pr.Foot.H/2}
		var spot Vec
		found := false
		for _, o := range []Vec{{0, pr.Foot.H/2 + 9}, {-pr.Foot.W/2 - 8, pr.Foot.H / 2}, {pr.Foot.W/2 + 8, pr.Foot.H / 2}} {
			p := Vec{c.X + o.X, c.Y + o.Y}
			if g.lvl.Free(p) && !g.lvl.SolidP(p.X, p.Y) && !g.lvl.badGround(p) && !g.lvl.InPool(p) {
				spot, found = p, true
				break
			}
		}
		if !found {
			continue
		}
		var cook *Guest
		for _, gu := range g.guests {
			if g.pevFreeGuest(gu) && (cook == nil || dist(gu.pos, spot) < dist(cook.pos, spot)) && g.lvl.PathG(gu.pos, spot) != nil {
				cook = gu
			}
		}
		if cook == nil {
			continue
		}
		m := &pevMem{gu: cook}
		if !g.pevSendTo(m, spot) {
			continue
		}
		a.D = &cookState{pr: pr, cook: m, spot: spot, t0: g.time}
		if pevDebugOn("cookfire") {
			pevDebugLog(g, "cookfire "+pr.Name, Vec{pr.X, pr.Y})
		}
		return true
	}
	return false
}

func cookEnd(g *Game, s *cookState) {
	if s.phase <= ckFlare {
		pevLetGo(s.cook)
	}
}

func (s *cookState) top() Vec { return Vec{s.pr.X, s.pr.Y - float64(s.pr.K.H)*0.55} }

// cookIgnite: the flare-up becomes a real fire in front of the cooker (fire.go does the rest: spread, burns,
// panic, the call to the firefighters, the safety end).
func cookIgnite(g *Game, s *cookState) {
	r := partyDice()
	at := Vec{(s.spot.X + s.pr.X) / 2, (s.spot.Y+s.pr.Y)/2 + 2}
	if g.lvl.InPool(at) {
		at = s.spot
	}
	f := &Fire{pos: at, origin: &fireOrigin{pos: at}, life: cookFireMin + r.Float64()*(cookFireMax-cookFireMin), w: 18,
		seed: r.Float64() * 10, spreadT: 2 + r.Float64()*2}
	g.blaze.fires = append(g.blaze.fires, f)
	s.fire = f
	g.fireSfx()
	g.snd.Play("ignite")
	g.burst(Vec{at.X, at.Y - 8}, 12, 50, rgb(0xffb030), 0.5, 2, 120)
	g.say(cookLines[3])
	// the cook lets go of the pan and everybody near runs
	pevLetGo(s.cook)
	cg := s.cook.gu
	for _, gu := range g.guests {
		if !gu.out() && gu.py == nil && dist(gu.pos, at) < 80 && lifeFree(gu) {
			g.scare(gu)
		}
	}
	if !cg.out() {
		cg.icon, cg.iconT = cookLines[5], 2
	}
}

func cookUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*cookState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase <= ckFlare {
		if !s.cook.mine() || !bbqUsable(s.pr) && s.phase < ckFlare {
			pevForget(s.cook)
			return true
		}
		s.cook.keep()
	}
	cg := s.cook.gu
	switch s.phase {
	case ckWalk:
		if cg.state == gIdle && cg.timer > 1000 || dist(cg.pos, s.spot) < 3 {
			s.cook.hold(gIdle)
			cg.dir = faceTo(cg.pos, Vec{s.pr.X, s.pr.Y - 8})
			s.phase, s.t0 = ckCook, g.time
			cg.icon, cg.iconT = cookLines[0], 2.2
		} else if age > cookWalkMax {
			pevLetGo(s.cook)
			return true
		}
	case ckCook:
		cg.dir = faceTo(cg.pos, Vec{s.pr.X, s.pr.Y - 8})
		if !s.said && age > cookT*0.5 && cg.icon == "" {
			s.said = true
			cg.icon, cg.iconT = cookLines[1], 2
		}
		if age >= cookT {
			if g.liveFires() > 0 { // something else is burning already: he just finishes, no second fire
				pevLetGo(s.cook)
				return true
			}
			s.phase, s.t0 = ckFlare, g.time
			s.cook.pose(AnimPanic)
			cg.icon, cg.iconT = cookLines[2], 2
			g.snd.Play("crackle")
		}
	case ckFlare:
		if age >= cookFlareT {
			s.phase, s.t0 = ckBurn, g.time
			cookIgnite(g, s)
		}
	case ckBurn:
		if s.fire == nil || s.fire.out {
			// over: somebody near thanks the firefighters (or blames the cook)
			for _, gu := range g.guests {
				if !gu.out() && gu.icon == "" && dist(gu.pos, s.spot) < 140 {
					gu.icon, gu.iconT = pevPick(r, cookLines[6:8]), 2.4
					break
				}
			}
			return true
		}
	}
	return false
}

func cookItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*cookState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	t := g.time - s.t0
	if s.phase == ckCook || s.phase == ckFlare {
		items = append(items, drawItem{s.pr.Y + 0.4, func() {
			x, y := g.pevXY(s.top())
			drawSmokePix(plot, x, y, px, g.time, 0.8)
			if s.phase == ckFlare {
				drawFlamePix(plot, x, y+px, px, g.time, math.Min(1, t/cookFlareT))
			}
		}})
	}
	if s.phase <= ckFlare && s.cook.mine() {
		cg := s.cook.gu
		items = append(items, drawItem{cg.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(cg.pos)
			drawCookKitPix(plot, cx, cy, px, k)
		}})
	}
	return items
}
