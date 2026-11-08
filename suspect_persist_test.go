package main

import "testing"

// Rule of the user (2026-11-08): once the police have been warned the cuffs stay on the player even if
// the witness dies afterwards ("ya hay polis buscándome"). WANTED is persistent state (g.wanted /
// jail.sus); only the ALERT "!" is computed from the live witnesses.

func persistSetup(t *testing.T) *Game {
	t.Helper()
	old := progressFrozen
	progressFrozen = true
	t.Cleanup(func() { progressFrozen = old })
	return alertSetup(t)
}

// killWitness kills gu in one of the real ways (player, trap, other agent: guard / special killer, run over).
func killWitness(g *Game, gu *Guest, how string) {
	switch how {
	case "player":
		g.killGuest(gu, true)
	case "trap":
		g.trapKill(gu)
	case "guard", "killer", "car":
		g.killCause = kcNone
		g.killGuest(gu, false)
	}
}

var persistCauses = []string{"player", "trap", "guard", "killer", "car"}

// settleFrames runs the suspect bookkeeping a few frames.
func settleFrames(g *Game) {
	for i := 0; i < 5; i++ {
		g.updateSuspect()
	}
}

func TestSuspectPersistsAfterCallerDies(t *testing.T) {
	for _, how := range persistCauses {
		g := persistSetup(t)
		gu := kill1P(g)
		finishCall(g, gu)
		if !g.playerIsSuspect() || g.wanted < 70 {
			t.Fatalf("%s: setup, cuffs expected", how)
		}
		killWitness(g, gu, how)
		settleFrames(g)
		if !gu.dead || !g.playerIsSuspect() || g.wanted < 69 {
			t.Fatalf("%s: the caller died but the cuffs must stay (wanted %.0f)", how, g.wanted)
		}
		if g.playerAlert(g.p) || g.pendingWitnesses(g.p) != 0 {
			t.Fatalf("%s: no '!' after the warning", how)
		}
	}
}

func TestSuspectPersistsAfterReportedWitnessDies(t *testing.T) {
	for _, how := range persistCauses {
		g := persistSetup(t)
		g.policeCalled = true
		gu := suspectGuest(g, Vec{230, 300}, DirLeft)
		c := suspectCop(g, Vec{330, 300}, DirDown)
		g.witness()
		gu.state, gu.reportTo, gu.timer = gReport, c, 14
		gu.pos = Vec{c.pos.X - 10, c.pos.Y}
		g.updateReport(gu, 0.1)
		if !gu.warned || !g.playerIsSuspect() {
			t.Fatalf("%s: setup, report should cuff", how)
		}
		killWitness(g, gu, how)
		settleFrames(g)
		if !g.playerIsSuspect() || g.playerAlert(g.p) {
			t.Fatalf("%s: cuffs must stay, no '!'", how)
		}
	}
}

func TestSuspectNoCuffsIfWitnessDiesBeforeWarning(t *testing.T) {
	for _, how := range persistCauses {
		g := persistSetup(t)
		gu := kill1P(g)
		killWitness(g, gu, how)
		settleFrames(g)
		finishCall(g, gu)
		if g.playerIsSuspect() || g.wanted > 0 || g.playerAlert(g.p) {
			t.Fatalf("%s: dead before warning -> free (wanted %.0f)", how, g.wanted)
		}
	}
}

// the second pending witness: warned one dies, the other keeps being pending but no "!" shows (cuffs on).
func TestSuspectWarnedThenDeadOtherWitnessNoAlert(t *testing.T) {
	g := persistSetup(t)
	a := suspectGuest(g, Vec{230, 300}, DirLeft)
	b := suspectGuest(g, Vec{240, 310}, DirLeft)
	suspectCop(g, Vec{900, 500}, DirLeft)
	g.witness()
	b.state, b.reportTo, b.timer = gReport, g.cops[0], 14
	g.callerGuest = a
	finishCall(g, a)
	g.killGuest(a, true)
	settleFrames(g)
	if !g.playerIsSuspect() || g.playerAlert(g.p) {
		t.Fatal("cuffs without '!' while another witness is still pending")
	}
	g.killGuest(b, false)
	settleFrames(g)
	if !g.playerIsSuspect() || g.wanted < 69 || g.playerAlert(g.p) {
		t.Fatal("all witnesses dead after the warning: cuffs stay, no '!'")
	}
}

// WANTED is read from state, never recomputed from the living witnesses.
func TestSuspectIsStateNotRecomputedFromWitnesses(t *testing.T) {
	g := persistSetup(t)
	g.guests = nil // nobody alive at all
	g.wanted = 70
	if !g.isSuspectP(g.p) {
		t.Fatal("wanted>0 and not cleared: suspect, whatever the witnesses")
	}
	settleFrames(g)
	if g.wanted != 70 {
		t.Fatalf("updateSuspect must not change wanted (%.0f)", g.wanted)
	}
}

func TestSuspectPersistsWithCopMemory(t *testing.T) {
	for _, how := range persistCauses {
		g := persistSetup(t)
		g.dbg.copMemory = true
		gu := kill1P(g)
		finishCall(g, gu)
		killWitness(g, gu, how)
		settleFrames(g)
		if !g.playerIsSuspect() || g.playerAlert(g.p) {
			t.Fatalf("%s: copmemory on: cuffs stay", how)
		}
	}
	g := persistSetup(t)
	g.dbg.copMemory = true
	gu := kill1P(g)
	killWitness(g, gu, "player")
	settleFrames(g)
	if g.playerIsSuspect() || g.playerAlert(g.p) {
		t.Fatal("copmemory on: dead before warning is still free")
	}
}

func TestSuspectPersistWardrobeStillClears(t *testing.T) {
	g := persistSetup(t)
	gu := kill1P(g)
	finishCall(g, gu)
	g.killGuest(gu, false)
	g.shakeOffPolice(g.p)
	if g.playerIsSuspect() {
		t.Fatal("disguise/wardrobe still clears")
	}
}

func TestSuspectPersist2P(t *testing.T) {
	for _, how := range persistCauses {
		for _, who := range []int{1, 2} {
			old := progressFrozen
			oldTwo := twoPlayerMode
			progressFrozen = true
			t.Cleanup(func() { progressFrozen = old; twoPlayerMode = oldTwo })
			g := twoGame(true)
			g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
			p1, p2 := g.p, g.p2
			p1.pos, p2.pos = Vec{200, 300}, Vec{700, 300}
			suspectCop(g, Vec{900, 900}, DirLeft)
			me, other := p1, p2
			at := Vec{230, 300}
			if who == 2 {
				me, other = p2, p1
				at = Vec{730, 300}
			}
			gu := suspectGuest(g, at, DirLeft)
			g.withPlayer(me, func() { g.witness() })
			if !g.playerAlert(me) {
				t.Fatalf("%s J%d: setup alert", how, who)
			}
			finishCall(g, gu)
			if !g.isSuspectP(me) || g.isSuspectP(other) {
				t.Fatalf("%s J%d: setup cuffs only for him", how, who)
			}
			g.withPlayer(me, func() { killWitness(g, gu, how) })
			settleFrames(g)
			if !g.isSuspectP(me) || g.isSuspectP(other) || g.playerAlert(me) || g.playerAlert(other) {
				t.Fatalf("%s J%d: cuffs must stay on him only, no '!'", how, who)
			}
		}
	}
	// 2P, witness dies BEFORE warning: free
	g := twoGame(true)
	old := progressFrozen
	progressFrozen = true
	t.Cleanup(func() { progressFrozen = old })
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	g.p.pos, g.p2.pos = Vec{200, 300}, Vec{700, 300}
	suspectCop(g, Vec{900, 900}, DirLeft)
	gu := suspectGuest(g, Vec{230, 300}, DirLeft)
	g.witness()
	g.killGuest(gu, false)
	settleFrames(g)
	finishCall(g, gu)
	if g.isSuspectP(g.p) || g.playerAlert(g.p) {
		t.Fatal("2P: dead before warning -> free")
	}
}
