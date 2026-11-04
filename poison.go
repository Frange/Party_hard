package main

import (
	"fmt"
	"math"
)

// ---------------------------------------------------------------------------
// Poison. A bottle holds 5 doses: each one can be offered to a guest ("a
// poisoned drink") or two go into a drink bowl (the punch), which poisons
// whoever drinks from it. Nobody drops dead at once: after 15-40 seconds the
// victim goes through three stages — dizzy (staggering, "@_@"), sick (retching
// and vomiting) and finally the collapse and the death.
//
// A poisoned guest never goes to tell the police about their own state: only
// somebody who actually saw the player offering the drink knows (see
// offerPoison), and the body is found and reported like any other.
// ---------------------------------------------------------------------------

const (
	poisonDoses      = 5   // doses in a bottle
	poisonBowlDoses  = 2   // doses a drink bowl takes
	poisonMin        = 15  // seconds before the end (minimum)
	poisonSpread     = 25  // ... plus up to this much
	poisonDizzyUntil = 0.4 // fraction of the time spent dizzy
	poisonSickUntil  = 0.85
)

// poisonStage: 0 healthy, 1 dizzy, 2 sick, 3 collapsing.
func poisonStage(gu *Guest) int {
	if gu.poison <= 0 || gu.poisonT0 <= 0 {
		return 0
	}
	f := 1 - gu.poison/gu.poisonT0
	switch {
	case f < poisonDizzyUntil:
		return 1
	case f < poisonSickUntil:
		return 2
	}
	return 3
}

// dosesText is the message that tells how much poison is left.
func dosesText(n int) string {
	switch {
	case n <= 0:
		return "Se acabó el veneno"
	case n == 1:
		return "Veneno: 1 dosis"
	}
	return fmt.Sprintf("Veneno: %d dosis", n)
}

// useDoses takes n doses from the player's bottle; the bottle is gone when it is empty.
func (g *Game) useDoses(n int) {
	p := g.p
	p.doses -= n
	if p.doses <= 0 {
		p.doses, p.item = 0, ""
	}
}

// poisonTarget is the guest the player could offer a drink to (nearest in reach), or nil.
func (g *Game) poisonTarget() *Guest {
	p := g.p
	if p.item != "poison" || p.doses <= 0 || p.dead || p.hidden || p.carrying != nil {
		return nil
	}
	var best *Guest
	bd := 28.0
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" || gu.poison > 0 || gu.offered {
			continue
		}
		switch gu.state {
		case gZombie, gLift, gToilet, gSwim, gSleep, gFlee, gScared, gRun:
			continue
		}
		if d := dist(gu.pos, p.pos); d < bd {
			best, bd = gu, d
		}
	}
	return best
}

// offerPoison: the player hands a guest a poisoned drink.
func (g *Game) offerPoison(gu *Guest) {
	p := g.p
	g.useDoses(1)
	gu.state, gu.path, gu.partner = gDrink, nil, nil
	gu.dur, gu.timer, gu.offered = 1.6, 1.6, true
	gu.dir = faceTo(gu.pos, p.pos)
	g.snd.Play("splash")
	g.award(gu.pos, 15, "veneno")
	g.say(dosesText(p.doses))
	// whoever is looking at the player right now will remember who handed over that drink
	for _, o := range g.guests {
		if o != gu && g.canWatch(o) && g.sees(o.pos, o.dir, p.pos, guestRange, guestDot) {
			o.sawKiller, o.sawPos = true, p.pos
		}
	}
}

// startPoison: the poison is in; the countdown starts and nothing shows yet.
func (g *Game) startPoison(gu *Guest) {
	if gu.poison > 0 || gu.out() {
		return
	}
	t := poisonMin + g.rng.Float64()*poisonSpread
	gu.poison, gu.poisonT0, gu.poisonTick = t, t, 3+g.rng.Float64()*2
	gu.offered = false
}

// poisonedStagger is the sideways sway of a dizzy guest on the move.
func poisonedStagger(gu *Guest, t float64) float64 {
	if poisonStage(gu) == 0 || !gu.moving {
		return 0
	}
	return math.Round(math.Sin(t*4.5+gu.off) * 1.6)
}

// updatePoison runs the stages. It returns true when the guest died (nothing else to do this frame).
func (g *Game) updatePoison(gu *Guest, dt float64) bool {
	if gu.poison <= 0 {
		return false
	}
	gu.poison -= dt
	if gu.poison <= 0 {
		g.credit = true
		g.killGuest(gu, false)
		g.credit = false
		g.kills++
		g.stat("kills", 1)
		g.stat("poison", 1)
		g.sayNews("kill", map[string]string{"actor": "El veneno", "victim": guestLabel(gu)})
		return true
	}
	// feeling ill is not a reason to call the police
	switch gu.state {
	case gToPhone, gReport, gCall:
		gu.state, gu.path, gu.timer = gIdle, nil, 1
	}
	stage := poisonStage(gu)
	switch stage {
	case 3:
		// the knees give way: they stop where they are, a last "x_x"
		if gu.state != gVomit && gu.state != gIdle {
			gu.state, gu.path, gu.timer = gIdle, nil, 1.5
		}
		if gu.iconT <= 0 {
			gu.icon, gu.iconT = "x_x", 1.2
		}
		return false
	}
	gu.poisonTick -= dt
	if gu.poisonTick > 0 {
		return false
	}
	switch stage {
	case 1: // dizzy
		gu.poisonTick = 2.5 + g.rng.Float64()*2.5
		if gu.iconT <= 0 {
			gu.icon, gu.iconT = []string{"@_@", "UUUH", "uuf...", "¿qué me pasa?"}[g.rng.Intn(4)], 1.6
		}
		if gu.state == gGoto && g.rng.Float64() < 0.3 {
			// leans on the nearest wall for a moment
			gu.state, gu.path, gu.timer = gIdle, nil, 2+g.rng.Float64()*1.5
		}
	case 2: // sick: retching and throwing up
		gu.poisonTick = 3 + g.rng.Float64()*2.5
		gu.state, gu.path, gu.timer = gVomit, nil, 2.2
		gu.icon, gu.iconT = []string{"BLEEERGH", "¡BLEH!", "*arcada*"}[g.rng.Intn(3)], 2
		// the ones nearby make a face
		for _, o := range g.guests {
			if o != gu && !o.out() && o.perf == "" && o.iconT <= 0 && dist(o.pos, gu.pos) < 44 && g.rng.Float64() < 0.25 {
				o.icon, o.iconT = []string{"¡Puaj!", "¡Qué asco!", "Aparta..."}[g.rng.Intn(3)], 1.4
			}
		}
	}
	return false
}
