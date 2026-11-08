package main

import "testing"

// A witness who saw the player kill and ran away scared: with the DEBUG option "police with memory" off
// nothing changes (he forgets); with it on, he tells on the player when he meets an officer.

func memGame(t *testing.T, on bool) (*Game, *Guest) {
	t.Helper()
	oldF, oldD := progressFrozen, debugSnap
	t.Cleanup(func() { progressFrozen, debugSnap = oldF, oldD })
	g := suspectGame()
	g.policeCalled = false
	g.mode = ModePlay
	g.dbg.copMemory = on
	gu := suspectGuest(g, Vec{200, 300}, DirLeft)
	gu.sawKiller, gu.sawP, gu.warned, gu.sawPos = true, g.p, false, g.p.pos
	gu.state, gu.timer = gScared, 5 // scared: no phone, no officer to go to
	return g, gu
}

func TestCopMemoryOffKeepsTheOldBehaviour(t *testing.T) {
	g, gu := memGame(t, false)
	suspectCop(g, Vec{300, 300}, DirUp) // an officer right there, looking elsewhere
	for i := 0; i < 120; i++ {
		g.updateSuspect()
	}
	if gu.warned || g.wanted != 0 || g.playerIsSuspect() {
		t.Errorf("option off: the scared witness must forget (warned=%v wanted=%.0f)", gu.warned, g.wanted)
	}
}

func TestCopMemoryOnScaredWitnessTellsOnYouNearAnOfficer(t *testing.T) {
	g, gu := memGame(t, true)
	for i := 0; i < 60; i++ {
		g.updateSuspect()
	}
	if gu.warned || g.wanted != 0 {
		t.Fatal("no officer around: nothing happens yet (he only remembers)")
	}
	c := suspectCop(g, Vec{500, 300}, DirUp) // 300 px away: inside 420
	g.updateSuspect()
	if !gu.warned || !g.playerIsSuspect() || g.wanted < 45 {
		t.Fatalf("the witness met an officer and said nothing (warned=%v wanted=%.0f)", gu.warned, g.wanted)
	}
	if c.state != cGoto {
		t.Errorf("the officer does not go to the place of the crime (state %v)", c.state)
	}
}

func TestCopMemoryTooFarDoesNotTell(t *testing.T) {
	g, gu := memGame(t, true)
	suspectCop(g, Vec{200 + copMemoryRange + 40, 300}, DirUp)
	g.updateSuspect()
	if gu.warned || g.wanted != 0 {
		t.Error("an officer farther than 420 px must not be enough")
	}
}

func TestCopMemoryDeadWitnessDoesNotTell(t *testing.T) {
	g, gu := memGame(t, true)
	gu.dead = true // killed before he told anybody: the clean move
	suspectCop(g, Vec{300, 300}, DirUp)
	g.updateSuspect()
	if gu.warned || g.wanted != 0 || g.playerIsSuspect() {
		t.Error("a dead witness cannot tell on the player")
	}
}

func TestCopMemoryGuardAlsoCounts(t *testing.T) {
	g, gu := memGame(t, true)
	g.guards = append(g.guards, &Guard{pos: Vec{350, 300}, dir: DirUp, state: gdWait})
	g.updateSuspect()
	if !gu.warned || g.wanted < 45 || g.guards[0].sight < 90 {
		t.Errorf("a guard nearby should be told (warned=%v wanted=%.0f sight=%.0f)", gu.warned, g.wanted, g.guards[0].sight)
	}
}

func TestCopMemoryWitnessOnHisWayToWarnIsLeftToTheAlert(t *testing.T) {
	g, gu := memGame(t, true)
	gu.state = gToPhone // still going to a phone: the normal alert handles him, memory does not
	g.callerGuest = gu
	suspectCop(g, Vec{300, 300}, DirUp)
	g.updateSuspect()
	if gu.warned {
		t.Error("memory must not warn for a witness who is still going to the phone")
	}
}

// The DEBUG menu has the switch and it goes off when the session ends.
func TestCopMemoryDebugMenuRow(t *testing.T) {
	oldF, oldS, oldP := progressFrozen, debugSnap, debugPrevFrozen
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen = oldF, oldS, oldP })
	progressFrozen, debugSnap, debugPrevFrozen = false, nil, false
	g := layerGame(t, false)
	found := false
	for _, r := range g.debugRows() {
		if r.label == "Policía con memoria (el testigo asustado delata)" {
			found = true
			if r.val() != "NO" {
				t.Errorf("the option must start OFF, shows %q", r.val())
			}
			r.act()
			if !g.dbg.copMemory || r.val() != "SÍ" {
				t.Error("the row does not switch the option on")
			}
		}
	}
	if !found {
		t.Fatal("DEBUG menu row for the police memory not found")
	}
	g.debugEnd()
	if g.dbg.copMemory {
		t.Error("leaving the DEBUG session must put the option off")
	}
	if rows := g.debugRows(); debugRow0+float64(len(rows))*debugRowH > 496 {
		t.Errorf("%d DEBUG rows do not fit above the status line", len(rows))
	}
}
