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
//   - a level has a killer with probability killerChance (0.30; 0.22 until 2026-11-12, the owner asked for a bit
//     more), at a random moment (killerAtMin..killerAtMax); never two parties in a row with one: on average about
//     1 party in 4 (0.23). The old slow-player accelerator
//     (3:30, one comes anyway) was REMOVED: it gave a killer to almost every long party;
//   - at most ONE per level, unless the level passes killerSecondAfter (5:00): then one more (two in total at most);
//   - never in STEALTH (those levels never plan one), not in cinematics (they never run updateSlasher);
//   - 2P: the clock is the level's (g.time), and the budget is shared by both killer slots.
// The time is g.time: game time without pauses (the one the speed bonus uses).
// Not part of the rule: the T-Rex / island monsters (level design), and a guest who goes mad for a while
// (murderT in game.go, a guest, not a new arrival).
// After one is killed (goalkillers.go): the next different killer comes at the latest 2 parties later (Progress.KillerWait).
// -killer forces one at ~6 s (like -fastevents) in every party; K (debug key) still sends one by hand.
// ---------------------------------------------------------------------------

const (
	killerRegulation  = true
	killerChance      = 0.30 // per party; with no back-to-back parties the real rate is p/(1+p) = ~0.23 (about 1 in 4)
	killerAtMin       = 30.0
	killerAtMax       = 150.0
	killerSecondAfter = 300.0 // 5:00
	killerForcedAt    = 6.0
)

var killerRng *rand.Rand

// killerDice: one own generator for the whole session (a new time-seeded one per level could repeat).
func killerDice() *rand.Rand {
	if killerRng == nil {
		killerRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return killerRng
}

// dbgKiller: -killer forces a killer in every party (debug).
var dbgKiller bool

type killerPlan struct {
	planned bool
	at      float64
	spawned int // killers that came in this level
}

// killerPlanFor decides whether a level gets a killer and when (its own rand: the dice of the party are not moved).
func killerPlanFor(r *rand.Rand, stealth bool) killerPlan {
	return killerPlanAfter(r, stealth, false)
}

// killerPlanAfter: prev = the previous party had a killer, so this one never does (no back-to-back killers).
func killerPlanAfter(r *rand.Rand, stealth, prev bool) killerPlan {
	if stealth {
		return killerPlan{}
	}
	if prev {
		r.Float64() // keep the dice stream the same length
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
	if k.planned && t >= killerSecondAfter {
		n = 2 // a party that drags on past 5:00 may get one more (only if it had one already)
	}
	return n
}

// killerLevelReset runs when a level loads.
func (g *Game) killerLevelReset() {
	prev := g.kp.planned
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
		g.kp = killerPlanAfter(killerDice(), g.stealth, prev)
		// after a killer was killed, the next one (a different one: goalkillers.go) is guaranteed within
		// killerNextWithin parties; never right after a party that had one (the no-back-to-back rule above)
		if g.killerWaitStep(prev) && !g.kp.planned {
			d := killerDice()
			g.kp = killerPlan{planned: true, at: killerAtMin + d.Float64()*(killerAtMax-killerAtMin)}
		}
	}
}

// killerDue: may a new killer come now?
func (g *Game) killerDue() bool {
	if !killerRegulation {
		return !g.slasherDone && g.time >= g.slasherT
	}
	return !g.stealth && g.time >= g.slasherT && g.kp.spawned < g.kp.allowed(g.time)
}
