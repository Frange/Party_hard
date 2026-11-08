package main

import "testing"

// New rule (2026-11-04): the cuffs are not for "being seen" but for "a witness WARNED a cop". Until he
// does, an ALERT shows over the player. Killing the witness first sets the player free.

func alertSetup(t *testing.T) *Game {
	t.Helper()
	old := progressFrozen
	t.Cleanup(func() { progressFrozen = old })
	g := suspectGame()
	g.policeCalled = false // nobody has called yet: the witness is the caller
	return g
}

// watcher: a guest next to the player looking at him, already reacting to the kill.
func kill1P(g *Game) *Guest {
	gu := suspectGuest(g, Vec{230, 300}, DirLeft)
	suspectCop(g, Vec{900, 500}, DirLeft) // far away: no officer sees the kill
	g.witness()
	return gu
}

// finishCall plays the end of the phone call of the caller.
func finishCall(g *Game, gu *Guest) {
	gu.state, gu.timer = gCall, 0.01
	g.updateGuest(gu, 0.1)
}

func TestAlertSeenKillGivesAlertNoCuffs(t *testing.T) {
	g := alertSetup(t)
	gu := kill1P(g)
	if !gu.sawKiller || gu.state != gAlarm {
		t.Fatalf("the witness should react (state %v)", gu.state)
	}
	if !g.playerAlert(g.p) || g.pendingWitnesses(g.p) != 1 {
		t.Fatal("a pending witness must raise the alert")
	}
	if g.playerIsSuspect() || g.wanted > 0 {
		t.Fatalf("being seen must not put the cuffs on (wanted %.0f)", g.wanted)
	}
}

func TestAlertBecomesCuffsWhenTheWitnessWarns(t *testing.T) {
	g := alertSetup(t)
	gu := kill1P(g)
	finishCall(g, gu)
	if !g.policeCalled || !gu.warned {
		t.Fatal("the call should have been made")
	}
	if !g.playerIsSuspect() || g.wanted < 70 {
		t.Fatalf("the warning puts the cuffs on (wanted %.0f)", g.wanted)
	}
	if g.playerAlert(g.p) {
		t.Fatal("the alert goes away once the cuffs are on")
	}
}

func TestAlertReportToAnOfficerAlsoWarns(t *testing.T) {
	g := alertSetup(t)
	g.policeCalled = true // already on the scene
	gu := suspectGuest(g, Vec{230, 300}, DirLeft)
	c := suspectCop(g, Vec{330, 300}, DirDown) // looks away: he is told, he does not see it
	g.witness()
	if g.copSeesAt(g.p.pos) || g.playerIsSuspect() {
		t.Fatal("setup: the officer must not see the kill himself")
	}
	gu.state, gu.reportTo, gu.timer = gReport, c, 14
	if !g.playerAlert(g.p) || g.playerIsSuspect() {
		t.Fatal("on his way to the officer: alert, no cuffs")
	}
	gu.pos = Vec{c.pos.X - 10, c.pos.Y}
	g.updateReport(gu, 0.1)
	if !gu.warned || !g.playerIsSuspect() || g.playerAlert(g.p) {
		t.Fatal("told the officer: cuffs, no alert")
	}
}

func TestAlertKillTheWitnessBeforeHeWarnsSetsYouFree(t *testing.T) {
	g := alertSetup(t)
	gu := kill1P(g)
	gu.dead = true
	if g.playerAlert(g.p) || g.pendingWitnesses(g.p) != 0 {
		t.Fatal("the dead witness no longer counts")
	}
	if g.playerIsSuspect() || g.wanted > 0 {
		t.Fatal("no cuffs: he never warned")
	}
	finishCall(g, gu) // a dead man tells nobody (the call is not completed)
	if g.playerIsSuspect() {
		t.Fatal("no cuffs after the dead witness")
	}
}

func TestAlertWitnessThatForgetsDropsTheAlert(t *testing.T) {
	g := alertSetup(t)
	gu := kill1P(g)
	g.scare(gu) // runs away without telling anybody
	if g.playerAlert(g.p) || g.playerIsSuspect() {
		t.Fatal("a scared witness who told nobody is not pending")
	}
}

func TestAlertTwoWitnessesOneWarnsOneDies(t *testing.T) {
	g := alertSetup(t)
	a := suspectGuest(g, Vec{230, 300}, DirLeft)
	b := suspectGuest(g, Vec{240, 310}, DirLeft)
	suspectCop(g, Vec{900, 500}, DirLeft)
	g.witness()
	if g.pendingWitnesses(g.p) != 2 && !(g.callerGuest == a || g.callerGuest == b) {
		t.Fatalf("both must be pending, got %d", g.pendingWitnesses(g.p))
	}
	b.state, b.reportTo = gReport, g.cops[0]
	b.timer = 14
	if g.pendingWitnesses(g.p) != 2 {
		t.Fatalf("two pending witnesses, got %d", g.pendingWitnesses(g.p))
	}
	b.dead = true
	if !g.playerAlert(g.p) {
		t.Fatal("one is still pending: the alert stays")
	}
	finishCall(g, a)
	if !g.playerIsSuspect() || g.playerAlert(g.p) {
		t.Fatal("the last one warned: cuffs and no alert")
	}
	a.dead = true // killing him afterwards does not take the cuffs off
	if !g.playerIsSuspect() {
		t.Fatal("after the warning, killing the witness keeps the cuffs")
	}
}

func TestAlertTwoWitnessesBothDieIsFree(t *testing.T) {
	g := alertSetup(t)
	a := suspectGuest(g, Vec{230, 300}, DirLeft)
	b := suspectGuest(g, Vec{240, 310}, DirLeft)
	suspectCop(g, Vec{900, 500}, DirLeft)
	g.witness()
	a.dead, b.dead = true, true
	if g.playerAlert(g.p) || g.playerIsSuspect() {
		t.Fatal("both witnesses are dead: free")
	}
}

func TestAlertOfficerWitnessIsImmediate(t *testing.T) {
	g := alertSetup(t)
	g.policeCalled = true
	suspectCop(g, Vec{230, 300}, DirLeft) // the officer looks at the kill
	g.witness()
	if !g.playerIsSuspect() || g.wanted < 45 {
		t.Fatalf("an officer who sees it himself: cuffs at once (wanted %.0f)", g.wanted)
	}
	if g.playerAlert(g.p) {
		t.Fatal("no alert when the cuffs are on")
	}
}

func TestAlertGuardSeeingDoesNotPutCuffs(t *testing.T) {
	g := alertSetup(t)
	g.guards = append(g.guards, &Guard{pos: Vec{230, 300}, dir: DirLeft})
	suspectCop(g, Vec{900, 500}, DirLeft)
	g.witness()
	if g.playerIsSuspect() || g.wanted > 0 {
		t.Fatal("a guard seeing the kill chases but does not put the cuffs on")
	}
	if g.guards[0].sight < 90 {
		t.Fatal("the guard should keep reacting as before")
	}
}

func TestAlertTwoPlayersAreIndependent(t *testing.T) {
	old := progressFrozen
	oldTwo := twoPlayerMode
	t.Cleanup(func() { progressFrozen = old; twoPlayerMode = oldTwo })
	g := twoGame(true)
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	p1, p2 := g.p, g.p2
	p1.pos, p2.pos = Vec{200, 300}, Vec{700, 300}
	suspectCop(g, Vec{900, 900}, DirLeft)
	gu := suspectGuest(g, Vec{730, 300}, DirLeft) // looks at J2 only
	g.withPlayer(p2, func() { g.witness() })
	if !g.playerAlert(p2) || g.playerAlert(p1) {
		t.Fatalf("the alert is for J2 only (p1=%v p2=%v)", g.playerAlert(p1), g.playerAlert(p2))
	}
	if g.isSuspectP(p1) || g.isSuspectP(p2) {
		t.Fatal("no cuffs yet")
	}
	finishCall(g, gu)
	if !g.isSuspectP(p2) || g.isSuspectP(p1) {
		t.Fatalf("the cuffs are for J2 only (p1=%v p2=%v)", g.isSuspectP(p1), g.isSuspectP(p2))
	}
	if g.playerAlert(p2) || g.playerAlert(p1) {
		t.Fatal("no alert after the warning")
	}
}

func TestAlertTwoPlayersWitnessDiesFreesOnlyThatPlayer(t *testing.T) {
	old := progressFrozen
	oldTwo := twoPlayerMode
	t.Cleanup(func() { progressFrozen = old; twoPlayerMode = oldTwo })
	g := twoGame(true)
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	p1, p2 := g.p, g.p2
	p1.pos, p2.pos = Vec{200, 300}, Vec{700, 300}
	suspectCop(g, Vec{450, 500}, DirDown) // near both, looking away
	a := suspectGuest(g, Vec{230, 300}, DirLeft)
	b := suspectGuest(g, Vec{730, 300}, DirLeft)
	g.withPlayer(p1, func() { g.witness() })
	g.withPlayer(p2, func() { g.witness() })
	if !g.playerAlert(p1) || !g.playerAlert(p2) {
		t.Fatal("each player has his own alert")
	}
	a.dead = true
	if g.playerAlert(p1) || !g.playerAlert(p2) {
		t.Fatal("J1's witness died: only J1 is free")
	}
	g.callerGuest = b // the second witness is the one who phones
	finishCall(g, b)
	if g.isSuspectP(p1) || !g.isSuspectP(p2) {
		t.Fatal("only J2 wears the cuffs")
	}
}

func TestAlertWardrobeStillClearsTheCuffs(t *testing.T) {
	g := alertSetup(t)
	gu := kill1P(g)
	finishCall(g, gu)
	if !g.playerIsSuspect() {
		t.Fatal("setup: cuffs expected")
	}
	g.shakeOffPolice(g.p)
	if g.playerIsSuspect() {
		t.Fatal("changing clothes still takes the cuffs off")
	}
}
