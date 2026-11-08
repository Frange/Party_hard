package main

import (
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// HOW OFTEN THE SPECIAL KILLER (the "Slasher": knife / chainsaw, he arrives in a macabre car or helicopter and
// kills guests) COMES TO A PARTY.
//
// Before: EVERY level had one 28-68 s after the start and, after each one was caught, up to two more 70-130 s later.
// Now (killerRegulation):
//   - a level has a killer with probability killerChance (1/3), at a random moment (killerAtMin..killerAtMax);
//   - the SLOW-PLAYER ACCELERATOR: once the level passes killerLateAfter (3:30) one comes anyway;
//   - at most ONE per level, unless the level passes killerSecondAfter (5:00): then one more (two in total at most);
//   - never in STEALTH (those levels never plan one), not in cinematics (they never run updateSlasher);
//   - 2P: the clock is the level's (g.time), and the budget is shared by both killer slots.
// The time is g.time: game time without pauses (the one the speed bonus uses).
// Not part of the rule: the T-Rex / island monsters (level design), and a guest who goes mad for a while
// (murderT in game.go, a guest, not a new arrival).
// -killer forces one at ~6 s (like -fastevents) in every party; K (debug key) still sends one by hand.
// ---------------------------------------------------------------------------

const (
	killerRegulation  = true
	killerChance      = 1.0 / 3
	killerAtMin       = 30.0
	killerAtMax       = 150.0
	killerLateAfter   = 210.0 // 3:30
	killerSecondAfter = 300.0 // 5:00
	killerForcedAt    = 6.0
)

// dbgKiller: -killer forces a killer in every party (debug).
var dbgKiller bool

type killerPlan struct {
	planned bool
	at      float64
	spawned int // killers that came in this level
}

// killerPlanFor decides whether a level gets a killer and when (its own rand: the dice of the party are not moved).
func killerPlanFor(r *rand.Rand, stealth bool) killerPlan {
	if stealth {
		return killerPlan{}
	}
	if r.Float64() >= killerChance {
		return killerPlan{}
	}
	return killerPlan{planned: true, at: killerAtMin + r.Float64()*(killerAtMax-killerAtMin)}
}

// killerAllowed: how many killers may have come by game time t.
func (k killerPlan) allowed(t float64) int {
	n := 0
	if k.planned && t >= k.at {
		n = 1
	}
	if t >= killerLateAfter {
		n = maxI(n, 1)
	}
	if t >= killerSecondAfter {
		n = 2
	}
	return n
}

// killerLevelReset runs when a level loads.
func (g *Game) killerLevelReset() {
	g.kp = killerPlan{}
	if !killerRegulation {
		return
	}
	g.slasherT, g.slot2.t = 0, 0 // the plan (kp) decides the moment, not the old random timers
	if g.stealth {
		return
	}
	switch {
	case dbgKiller || fastEv:
		g.kp = killerPlan{planned: true, at: killerForcedAt}
	default:
		g.kp = killerPlanFor(rand.New(rand.NewSource(time.Now().UnixNano())), g.stealth)
	}
}

// killerDue: may a new killer come now?
func (g *Game) killerDue() bool {
	if !killerRegulation {
		return !g.slasherDone && g.time >= g.slasherT
	}
	return !g.stealth && g.time >= g.slasherT && g.kp.spawned < g.kp.allowed(g.time)
}
