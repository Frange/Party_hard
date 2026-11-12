package main

// ---------------------------------------------------------------------------
// POLICE WITH MEMORY (DEBUG option, off by default: DEBUG menu row or -copmemory).
//
// Normal rule (suspect.go): a witness who saw the player kill and does not warn anybody (he got
// scared and ran, gScared / gFlee) simply forgets: no alert, no cuffs.
// With this option on, he REMEMBERS (guest.sawKiller stays set and warned stays false). If he later
// crosses an officer (or a guard) closer than copMemoryRange he tells on the player: guestWarned,
// the wanted level goes up and the officer walks to where it happened, like a normal report.
// Killing the witness while he has not warned still clears the player (a dead guest does not talk).
// ---------------------------------------------------------------------------

// copMemoryRange: how close an officer or a guard must be for the witness to tell on the player
// (the same distance in which a fresh witness looks for an officer to report to: nearCop).
const copMemoryRange = 420.0

// copMemoryWitness: gu saw a player kill, has told nobody, and is not on his way to do it now
// (the alert covers that case): he is scared, running or back to his business.
func (g *Game) copMemoryWitness(gu *Guest) bool {
	if gu == nil || gu.out() || gu.tech || gu.py != nil || !gu.sawKiller || gu.warned || gu.sawP == nil {
		return false
	}
	if g.witnessPending(gu, gu.sawP) {
		return false
	}
	switch gu.state {
	case gScared, gFlee, gIdle, gGoto:
		return true
	}
	return false
}

// copMemoryAuthority: the officer (not a paramedic) or the guard nearest to pos within copMemoryRange.
func (g *Game) copMemoryAuthority(pos Vec) (*Cop, *Guard) {
	var bc *Cop
	var bg *Guard
	best := copMemoryRange
	for _, c := range g.cops {
		if c.medic || c.state == cLeave || c.state == cCordon { // sealing off a closed case: off duty (ambulance.go)
			continue
		}
		if d := dist(c.pos, pos); d < best {
			best, bc, bg = d, c, nil
		}
	}
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		if d := dist(gd.pos, pos); d < best {
			best, bc, bg = d, nil, gd
		}
	}
	return bc, bg
}

// updateCopMemory runs every frame (from updateSuspect) and does nothing unless the option is on.
func (g *Game) updateCopMemory() {
	if !g.dbg.copMemory || g.mode != ModePlay {
		return
	}
	for _, gu := range g.guests {
		if !g.copMemoryWitness(gu) {
			continue
		}
		if c, gd := g.copMemoryAuthority(gu.pos); c != nil || gd != nil {
			g.copMemoryTell(gu, c, gd)
		}
	}
}

// copMemoryTell: the witness points at the player in front of the officer / the guard.
func (g *Game) copMemoryTell(gu *Guest, c *Cop, gd *Guard) {
	gu.icon, gu.iconT = g.line(guestCategory(gu), "accused"), 2.5
	g.guestWarned(gu) // the warning is given: the alert / nothing becomes the cuffs
	if gu.sawP != nil && gu.sawP != g.p {
		g.markSuspect(gu.sawP)
	}
	g.markPlayerWanted(70)
	g.killPos = gu.sawPos
	switch {
	case c != nil:
		c.icon, c.iconT = g.line(copCategory(c.kind), "understood"), 2
		if c.state != cChase {
			g.releaseCop(c)
			c.state, c.wait = cGoto, 0
			c.path = g.lvl.Path(c.pos, gu.sawPos)
		}
	case gd != nil:
		gd.sight = 99 // the guard goes for the player (he puts no cuffs on by himself)
	}
}
