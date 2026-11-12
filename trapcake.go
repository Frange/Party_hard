package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE EXPLOSIVE CAKE (idea 268, TARTA EXPLOSIVA): a trap the player sets off himself. Sometimes (the party event "tarta",
// only in birthday / house / party venues, own dice) a birthday cake with candles is brought to a little side table. It is
// one more trap of trapmine.go (Trap "cake"): E starts the usual load (the candles fizz: sparks and a crackle), and as soon
// as the load is over and nobody of the team is inside its radius (the same armWait rule as every trap) it BLOWS UP by itself
// (no waiting for somebody to step in: it is a timed cake, not a mine). A cloud of cream:
//   - within cakeKillR everybody dies (guests, guards, police, animals, killers), as with any trap;
//   - within cakeStunR the guests are stunned (stars, cakeStunDur), the guards dazed, a player is stunned a moment;
//   - the cream stays on the floor (a white stain) for cakeCreamLife seconds: a guest who walks into it slips (cakeSlipDur).
// The dead count like every trap's: they are the player's (g.credit; in 2P the owner's), with the "traps" statistic, and the
// usual rules of witnesses, stars and clean streak of a trap kill (triggerTrap). No new trophy.
// ---------------------------------------------------------------------------

const (
	cakeKillR     = 30.0 // world px from the cake: dead
	cakeStunR     = 78.0 // ... stunned and covered in cream
	cakePlayerR   = 22.0 // a player this close dies (he can only be this close if he came back after it was armed)
	cakeStunDur   = 3.2  // seconds a guest stays stunned
	cakePStun     = 1.0  // seconds a player in the stun radius sees stars
	cakeCreamLife = 22.0 // seconds the cream stays slippery
	cakeCreamR    = 24.0 // radius of the slippery cream on the floor
	cakeSlipDur   = 1.4  // seconds a guest who slips stays down
)

var cakeIcons = map[string]bool{"cake": true, "house": true, "roof": true, "disco70": true, "neon": true, "casino": true}

var cakeLines = pevTexts("¡NATA!", "¡Me ha entrado en el ojo!", "¡Mi traje!", "¡Ups!", "¡Resbala!", "¡Qué tarta más mala!")

// cakeSplat: cream on the floor of this level (slippery for a while).
type cakeSplat struct {
	pos    Vec
	until  float64
	slip   map[*Guest]bool
	lvl    *Level
	splash float64 // game time of the blast (the drops in the air)
}

var cakeSplats []*cakeSplat

func init() {
	propKinds["bombcake"] = &PropKind{W: 26, H: 32, FW: 16, FH: 6, Vars: 2, Trap: "cake", Category: "furniture", Draw: drawBombCake}
	registerPartyEvent(&partyEvent{ID: "tarta", Name: "Tarta explosiva", Group: "cake",
		Prob: 0.18, AtMin: 4, AtMax: 90, Dur: 4,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && cakeIcons[t.icon] },
		Start:    cakeStart,
		Update:   cakeEventUpdate})
	// the cream on the floor: reactive (it follows the splats; no dice, not one of the three of a party)
	registerPartyEvent(&partyEvent{ID: "nata", Name: "Nata en el suelo", Group: "", Reactive: true, DailyOK: true,
		Prob: 0.10, AtMin: 0, AtMax: 1, Dur: cakeCreamLife,
		Eligible: func(g *Game, t partyTraits) bool { return len(g.cakeSplatsHere()) > 0 },
		Start:    func(g *Game, a *partyAct) bool { return true },
		Update:   cakeCreamUpdate, Items: cakeCreamItems})
}

// drawBombCake: a round side table with a cloth and, on it, a two-layer pink cake with three candles (var 0); after the
// blast only the plate and some cream are left (var 1). The bottom rows are the table's feet (the footprint).
func drawBombCake(b *pixBuf, v int) {
	cloth, clothD := c(0xf2ece0), c(0xc8bfae)
	b.rect(9, 27, 2, 5, c(0x5a3a1e))
	b.rect(15, 27, 2, 5, c(0x5a3a1e))
	b.rect(5, 18, 16, 10, clothD)
	b.rect(6, 18, 14, 9, cloth)
	for x := 7; x < 20; x += 3 {
		b.rect(x, 19, 1, 8, clothD)
	}
	b.ellipse(13, 18, 9, 3, cloth)
	b.ellipse(13, 18, 7, 2, c(0xe4e8f0)) // the plate
	if v == 1 {
		b.rect(8, 15, 4, 2, c(0xfff8f0))
		b.rect(14, 16, 5, 2, c(0xfff8f0))
		b.rect(11, 14, 2, 1, c(0xff8ab0))
		return
	}
	pink, pinkD, cream := c(0xf48ab4), c(0xc8608a), c(0xfff4e8)
	b.rect(6, 11, 14, 7, pinkD)
	b.rect(7, 11, 12, 6, pink)
	b.rect(6, 11, 14, 2, cream)
	for x := 7; x < 19; x += 3 {
		b.rect(x, 13, 1, 2, cream) // drips
	}
	b.rect(8, 6, 10, 5, pinkD)
	b.rect(9, 6, 8, 4, pink)
	b.rect(8, 6, 10, 2, cream)
	b.rect(12, 5, 2, 1, c(0xd02030)) // the cherry
	for _, x := range []int{9, 13, 16} {
		b.rect(x, 2, 1, 4, c(0x80c0f0))
		b.rect(x, 0, 1, 2, c(0xffc840))
	}
}

func cakeStart(g *Game, a *partyAct) bool {
	l := g.lvl
	if pevDemo("tarta") != "" { // capture: the cake near the player, set off by itself
		return cakeDemoStart(g, a)
	}
	var seeds []Vec
	for _, p := range l.Props {
		if p.K.Trap != "" || p.K.Poison || p.K.Phone {
			seeds = append(seeds, Vec{p.X, p.Y})
		}
	}
	have := countKind(l, "bombcake")
	if have > 0 {
		return false
	}
	before := len(l.Props)
	if !l.ensureBooth("bombcake", 1, func(p *Prop) bool { return p.Name == "bombcake" }, seeds, 28) {
		return false
	}
	l.initNav()
	pr := l.Props[before]
	if !g.pevPlayerFar(Vec{pr.X, pr.Y}, 40) { // never right under the player's nose: next time
		g.pevRemoveProp(pr)
		return false
	}
	a.D = pr
	return true
}

func cakeEventUpdate(g *Game, a *partyAct, dt float64) bool {
	if pevDemo("tarta") != "" {
		return cakeDemoUpdate(g, a)
	}
	return true // the cake stays as a trap of the level; the event is only its arrival
}

// cakeArmed: E on the cake (trapmine.armTrap): the candles fizz.
func (g *Game) cakeArmed(pr *Prop) {
	g.snd.Play("crackle")
}

// cakeGoesOff: trapmine asks whether an armed trap blows up now: the cake does as soon as it is armed.
func cakeGoesOff(pr *Prop) bool { return pr.K.Trap == "cake" }

// triggerCakeTrap: the blast (called by triggerTrapNow with g.credit set, like every trap).
func (g *Game) triggerCakeTrap(pr *Prop) {
	centre := trapCentre(pr)
	r := partyDice()
	g.snd.Play("boom")
	g.snd.Play("splash")
	g.shake = 0.4
	top := Vec{pr.X, pr.Y - 16}
	g.burst(top, 46, 150, rgb(0xfff4e8), 0.9, 3, 70)
	g.burst(top, 18, 110, rgb(0xff8ab0), 0.7, 2, 70)
	g.burst(top, 8, 90, rgb(0xd02030), 0.6, 2, 90)
	g.decals = append(g.decals, Decal{pos: Vec{pr.X, pr.Y + 2}, kind: 5, r: 26})
	pr.Var = 1
	for _, gu := range g.guests {
		if gu.out() {
			continue
		}
		switch d := dist(gu.pos, centre); {
		case d < cakeKillR:
			g.trapKill(gu)
		case d < cakeStunR:
			gu.vipStun = math.Max(gu.vipStun, cakeStunDur)
			gu.icon, gu.iconT = pevPick(r, cakeLines[:3]), 2
		}
	}
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		switch d := dist(gd.pos, centre); {
		case d < cakeKillR:
			g.trapGuard(gd)
		case d < cakeStunR:
			gd.smogT = math.Max(gd.smogT, cakeStunDur)
		}
	}
	g.trapCops(centre, cakeKillR)
	g.trapHitOthers(func(v Vec) bool { return dist(v, centre) < cakeKillR })
	g.eachPlayer(func(p *Player) {
		if p.dead {
			return
		}
		switch d := dist(p.pos, centre); {
		case d < cakePlayerR:
			g.die("Te ha explotado una tarta en la cara")
		case d < cakeStunR:
			p.stunT = math.Max(p.stunT, cakePStun)
		}
	})
	for _, gu := range g.guests {
		if !gu.out() && dist(gu.pos, centre) < 220 {
			g.alarm(gu, "!?")
		}
	}
	cakeSplats = append(cakeSplats, &cakeSplat{pos: Vec{pr.X, pr.Y + 2}, until: g.time + cakeCreamLife, slip: map[*Guest]bool{}, lvl: g.lvl, splash: g.time})
	g.say("¡La tarta ha explotado!")
	g.checkDone()
}

// cakeSplatsHere: the cream that is still slippery in this party.
func (g *Game) cakeSplatsHere() []*cakeSplat {
	out := cakeSplats[:0]
	var here []*cakeSplat
	for _, s := range cakeSplats {
		if s.lvl != g.lvl || g.time > s.until || g.time < s.splash-1 {
			continue // another party (or a time from before this one): forgotten
		}
		out = append(out, s)
		here = append(here, s)
	}
	cakeSplats = out
	return here
}

// cakeCreamUpdate: a guest who walks into the cream slips (once per splat) and stays down a moment with stars.
func cakeCreamUpdate(g *Game, a *partyAct, dt float64) bool {
	ss := g.cakeSplatsHere()
	if len(ss) == 0 {
		return true
	}
	r := partyDice()
	for _, s := range ss {
		for _, gu := range g.guests {
			if gu.out() || s.slip[gu] || !gu.moving || gu.vipStun > 0 || gu.perf != "" || dist(gu.pos, s.pos) > cakeCreamR {
				continue
			}
			s.slip[gu] = true
			gu.vipStun = cakeSlipDur
			gu.icon, gu.iconT = pevPick(r, cakeLines[3:]), 1.4
		}
	}
	return false
}

// cakeCreamItems: drops of cream that fly off for a second after the blast.
func cakeCreamItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	plot := g.plotter(dst)
	px := g.pevPx()
	for _, s := range g.cakeSplatsHere() {
		s := s
		age := g.time - s.splash
		if age > 1.2 || age < 0 {
			continue
		}
		items = append(items, drawItem{s.pos.Y + 20, func() {
			for i := 0; i < 10; i++ {
				ang := float64(i) * 0.628
				d := 10 + age*55
				x, y := g.pevXY(Vec{s.pos.X + math.Cos(ang)*d, s.pos.Y - 10 + math.Sin(ang)*d*0.5 - 18*math.Sin(math.Pi*math.Min(1, age/1.2))})
				col := pevWhite
				if i%3 == 0 {
					col = color.RGBA{0xff, 0x8a, 0xb0, 255}
				}
				plot(x, y, 2*px, 2*px, col)
			}
		}})
	}
	return items
}

// ---- capture: -pevent tarta:boom ---------------------------------------------------------------------------

type cakeDemo struct {
	pr     *Prop
	armed  bool
	posed  bool
	t0     float64
	placed []*Guest
}

func cakeDemoStart(g *Game, a *partyAct) bool {
	k := propKinds["bombcake"]
	fitAllFootprints()
	reach0 := g.lvl.reachFrom(g.lvl.Def.Start)
	for i := 0; i < 60; i++ {
		p, ok := g.pevNearPlayerSpot(20, false)
		if !ok || g.lvl.propClashes("bombcake", k, p, 0) {
			continue
		}
		if pr := g.pevAddProp("bombcake", k, p, 0, reach0, nil); pr != nil {
			a.D = &cakeDemo{pr: pr, t0: g.time}
			pevDemoLog("tarta", p)
			return true
		}
	}
	return false
}

func cakeDemoUpdate(g *Game, a *partyAct) bool {
	d := a.D.(*cakeDemo)
	pr := d.pr
	c := trapCentre(pr)
	if !d.armed && g.time-d.t0 > 1 {
		d.armed = true
		g.p.pos = g.lvl.NearestFree(Vec{c.X - 50, c.Y + 6})
		g.armTrap(pr, g.p)
		g.cakeArmed(pr)
	}
	if d.armed && !d.posed && !pr.used {
		// three guests around it: one inside the deadly radius, two in the cream
		n := 0
		for _, gu := range g.guests {
			if n >= 3 || gu.out() || gu.perf != "" {
				continue
			}
			off := []Vec{{14, 4}, {44, 8}, {-6, 52}}[n]
			gu.pos = g.lvl.NearestFree(Vec{c.X + off.X, c.Y + off.Y})
			gu.state, gu.timer, gu.path = gIdle, 4, nil
			n++
		}
		d.posed = n > 0
	}
	return pr.used && g.time-d.t0 > 40
}
