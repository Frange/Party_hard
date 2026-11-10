package main

import "math"

// The authorities against the T-Rex (Isla Nublar, 2026-11-09).
//
//   - Whatever the T-Rex kills on the island is reported like the player's kills (Body.byRex):
//     a witness calls, the local police come first and then the group of the stretch
//     (police_groups.go, k = every death of the party). It never gives the player stars.
//     The officers stay until every body is covered.
//   - The local police and the FBI cannot hurt it: they get eaten or keep out of its way.
//   - The military and the SWAT shoot it: it gets weaker and slower, never below rexFloorHP.
//   - The Power Rangers fight it hand to hand: with chance rexRangerWinP they bring it down;
//     otherwise it stays on its feet and eats some of them.

const (
	rexMaxHP      = 400.0 // a lot of life
	rexFloorHP    = 140.0 // what the guns can take it down to: weakened, never killed
	rexRangerWinP = 0.35  // the Power Rangers win the fight about one time in three
	rexSlowMin    = 0.75  // its speed at 0 life (x1 at full life)
	rexFightRange = 210.0 // how far the armed groups engage it from
	rexMeleeRange = 40.0  // the Power Rangers hit it from this close
)

// rexShotDamage: one burst of fire / one blow of a group. The police and the FBI do nothing.
func rexShotDamage(unit int) float64 {
	switch unit {
	case unitMilitary:
		return 16
	case unitSWAT:
		return 12
	case unitRangers:
		return 24
	}
	return 0
}

// rexDamage: the life the T-Rex is left with after a hit of this group. Only the Power
// Rangers who won their roll can take it to 0; everybody else stops at rexFloorHP.
func rexDamage(hp float64, unit int, rangersWin bool) float64 {
	d := rexShotDamage(unit)
	if d <= 0 {
		return hp
	}
	floor := rexFloorHP
	if unit == unitRangers && rangersWin {
		floor = 0
	}
	if hp <= floor {
		return hp
	}
	return math.Max(floor, hp-d)
}

// rexSlow: how fast it still moves with this much life.
func rexSlow(hp float64) float64 {
	return rexSlowMin + (1-rexSlowMin)*clampF(hp/rexMaxHP, 0, 1)
}

// dbgRexKill: -trexkill (debug) sends the Power Rangers to the island and they always win.
var dbgRexKill bool

// rexRangersRoll decides, once per T-Rex, whether the Power Rangers will win.
func (g *Game) rexRangersRoll(r *JRex) bool {
	if !r.rolled {
		r.rolled = true
		r.rangersWin = dbgRexKill || g.rng.Float64() < rexRangerWinP
	}
	return r.rangersWin
}

// copRexHook: the officers that fight the T-Rex this frame (true = the normal behaviour is skipped).
func (g *Game) copRexHook(c *Cop, dt float64) bool {
	r := g.rex
	if r == nil || !g.isIsland() || r.state == jrSleep || c.medic || c.state == cLeave || c.state == cArrest {
		return false
	}
	if rexShotDamage(c.kind) <= 0 {
		return false
	}
	d := dist(c.pos, r.pos)
	if d > rexFightRange || !g.lvl.LOS(c.pos, Vec{r.pos.X, r.pos.Y - 10}) {
		return false
	}
	c.dir = faceTo(c.pos, r.pos)
	c.shootT -= dt
	if c.kind == unitRangers {
		if d > rexMeleeRange {
			c.repath -= dt
			if c.repath <= 0 || len(c.path) == 0 {
				c.repath = 0.4
				c.path = g.lvl.Path(c.pos, r.pos)
			}
			g.copWalk(c, copChaseSpeed, dt)
			return true
		}
		c.moving = false
		if c.shootT <= 0 {
			c.shootT = 0.5 + g.rng.Float64()*0.3
			g.rexHit(c)
		}
		return true
	}
	// military and SWAT: they stop and fire
	c.moving = false
	if c.shootT <= 0 {
		c.shootT = 1.1 + g.rng.Float64()*0.7
		g.rexHit(c)
	}
	return true
}

// rexHit: one hit on the T-Rex, with visible damage.
func (g *Game) rexHit(c *Cop) {
	r := g.rex
	win := c.kind == unitRangers && g.rexRangersRoll(r)
	before := r.hp
	r.hp = rexDamage(r.hp, c.kind, win)
	at := Vec{r.pos.X + (g.rng.Float64()-0.5)*30, r.pos.Y - 24 - g.rng.Float64()*20}
	if c.kind == unitRangers {
		g.snd.Play("stab")
		g.burst(at, 10, 90, rgb(0xffe040), 0.4, 2, 0) // the sparks of a karate kick
		if c.icon == "" {
			c.icon, c.iconT = "¡HIYAA!", 0.8
		}
	} else {
		g.snd.Play("boom")
		g.burst(at, 6, 70, rgb(0xffb040), 0.3, 2, 0)
		if c.icon == "" {
			c.icon, c.iconT = "¡FUEGO!", 1
		}
	}
	if r.hp < before {
		g.burst(at, 8, 60, rgb(0xb01818), 0.5, 2, 200) // it bleeds
	}
	if r.hp <= 0 {
		g.rexDies()
		return
	}
	// the Power Rangers losing the fight: it turns on them
	if c.kind == unitRangers && !win && r.state != jrEat && g.rng.Float64() < 0.35 {
		cs := g.copCSFor(c)
		cc := c
		r.cur = rexPrey{id: cc, at: func() Vec { return cc.pos }, cs: cs, alive: func() bool { return g.copInList(cc) },
			kill: func() { g.rexKillCop(cc, cs) }}
		r.state, r.repath = jrChase, 0
	}
}

// rexDies: the Power Rangers bring it down. It is not a guest: the party goes on as before.
func (g *Game) rexDies() {
	r := g.rex
	if r == nil {
		return
	}
	if r.eatBody != nil {
		r.eatBody = nil
	}
	g.snd.Play("rex_roar")
	g.shake = 0.7
	for i := 0; i < 4; i++ {
		g.rexSplat(Vec{r.pos.X + (g.rng.Float64()-0.5)*50, r.pos.Y - 10}, nil)
	}
	g.burst(Vec{r.pos.X, r.pos.Y - 20}, 50, 140, rgba(0xb8a888, 220), 1.2, 3, 40)
	g.say("¡Los Power Rangers han tumbado al T-Rex! Sin Megazord.")
	g.rex = nil
}

// partyDeaths: the guests dead in this party, whoever killed them (the police groups count them all).
func (g *Game) partyDeaths() int {
	n := 0
	for _, gu := range g.guests {
		if gu.dead {
			n++
		}
	}
	return n
}

// rexWitnesses: whoever sees the T-Rex eat somebody knows about the body; the nearest runs to call.
func (g *Game) rexWitnesses(b *Body) {
	if !b.byRex {
		return
	}
	var best *Guest
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" || gu.state == gZombie || gu.state == gLift {
			continue
		}
		if g.sees(gu.pos, gu.dir, b.pos, guestRange*1.5, -0.2) { // a T-Rex eating is hard to miss
			g.knowBody(gu, b)
			if best == nil || dist(gu.pos, b.pos) < dist(best.pos, b.pos) {
				best = gu
			}
		}
	}
	if best != nil && g.callerAvailable() {
		if g.alertPos == (Vec{}) {
			g.alertPos = b.pos
		}
		g.becomeCaller(best)
	}
}
