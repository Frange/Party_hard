package main

import (
	"math/rand"
	"testing"
)

// a small party: the player, one witness who looks at the spot of the kill, and an officer on the scene
func suspectGame() *Game {
	g := testGame()
	g.rng = rand.New(rand.NewSource(11))
	g.snd = &Sounds{muted: true}
	g.lvl = buildLevel(level1())
	g.p = &Player{pos: Vec{200, 300}, stamina: 6}
	g.policeCalled, g.pendingCar, g.pendingUnit = true, -1, -1
	return g
}

func suspectGuest(g *Game, at Vec, face Dir) *Guest {
	spec := genericSpec(g.rng)
	gu := &Guest{pos: at, dir: face, cs: newCharSet(&spec), state: gIdle, name: "T"}
	g.guests = append(g.guests, gu)
	g.total++
	return gu
}

func suspectCop(g *Game, at Vec, face Dir) *Cop {
	c := &Cop{pos: at, dir: face, state: cSweep, maxChase: 6}
	g.cops = append(g.cops, c)
	return c
}

func TestKillSeenBySomebodyElseClearsThePlayer(t *testing.T) {
	g := suspectGame()
	g.wanted = 70
	g.lastKiller = lastKill{kind: suspPlayer}
	c := suspectCop(g, Vec{230, 300}, DirLeft)
	c.state = cChase
	g.slasher = &Slasher{state: slHunt, pos: Vec{400, 300}, need: 2}
	w := suspectGuest(g, Vec{330, 300}, DirRight) // looks at the spot where the killer strikes
	_ = w
	g.slasherWitness(g.slasher, Vec{380, 300})
	if g.lastKiller.kind != suspSlasher {
		t.Fatalf("the killer was seen but is not the suspect: %v", g.lastKiller.kind)
	}
	if g.wanted != 0 || g.playerIsSuspect() {
		t.Error("the player keeps his cuffs after somebody else was seen killing")
	}
	if c.state == cChase {
		t.Error("the officer still chases the cleared player")
	}
	if p := g.suspectProblems(); len(p) > 0 {
		t.Errorf("inconsistent state: %v", p)
	}
}

func TestPlayerSeenKillingGetsTheCuffsBack(t *testing.T) {
	g := suspectGame()
	g.lastKiller = lastKill{kind: suspSlasher}
	g.slasher = &Slasher{state: slHunt, pos: Vec{400, 300}}
	suspectGuest(g, Vec{230, 300}, DirLeft) // sees the player
	g.p.pos = Vec{200, 300}
	suspectCop(g, Vec{600, 100}, DirDown)
	g.witness()
	if g.lastKiller.kind != suspPlayer {
		t.Fatalf("the player was seen killing but the suspect is %v", g.lastKiller.kind)
	}
	if !g.playerIsSuspect() || g.wanted < 45 {
		t.Errorf("the cuffs did not come back (wanted %.0f)", g.wanted)
	}
}

func TestUnseenKillChangesNothing(t *testing.T) {
	g := suspectGame()
	g.wanted = 50
	g.lastKiller = lastKill{kind: suspPlayer}
	g.slasher = &Slasher{state: slHunt, pos: Vec{400, 300}, need: 2}
	suspectGuest(g, Vec{900, 480}, DirLeft) // far away and looking elsewhere
	suspectCop(g, Vec{900, 500}, DirLeft)
	g.slasherWitness(g.slasher, Vec{380, 300})
	if g.lastKiller.kind != suspPlayer || g.wanted != 50 {
		t.Errorf("nobody saw it: suspect %v, wanted %.0f", g.lastKiller.kind, g.wanted)
	}
	// and the same for the player: a kill with no witness does not make him a suspect
	g2 := suspectGame()
	g2.slasher = &Slasher{state: slHunt, pos: Vec{400, 300}}
	g2.lastKiller = lastKill{kind: suspSlasher}
	suspectGuest(g2, Vec{900, 480}, DirLeft)
	g2.witness()
	if g2.lastKiller.kind != suspSlasher || g2.wanted != 0 {
		t.Error("an unseen kill by the player must not change who is the suspect")
	}
}

func TestOfficersGoForTheArmedGuestSeenKilling(t *testing.T) {
	g := suspectGame()
	g.wanted = 70
	g.lastKiller = lastKill{kind: suspPlayer}
	killer := suspectGuest(g, Vec{300, 300}, DirRight)
	killer.armed, killer.rampage = true, 1
	victim := suspectGuest(g, Vec{308, 300}, DirLeft)
	killer.state, killer.tgt, killer.timer = gKiller, victim, 40
	suspectGuest(g, Vec{360, 300}, DirLeft) // a witness who sees the victim
	c := suspectCop(g, Vec{500, 300}, DirLeft)
	g.updateKiller(killer, 0.05)
	if g.lastKiller.kind != suspGuest || g.lastKiller.guest != killer {
		t.Fatalf("the armed guest was seen killing but is not the suspect (%v)", g.lastKiller.kind)
	}
	if g.suspect != killer {
		t.Error("the police suspect is not the armed guest")
	}
	if g.wanted != 0 {
		t.Error("the player is still wanted")
	}
	if c.state != cArrest {
		t.Errorf("the officer does not go for the killer (state %v)", c.state)
	}
}

func TestOfficersArrestTheKillerWhenHeIsTheLastSeen(t *testing.T) {
	g := suspectGame()
	killer := suspectGuest(g, Vec{300, 300}, DirRight)
	killer.armed = true
	g.lastKiller = lastKill{kind: suspGuest, guest: killer}
	g.suspect = killer
	c := suspectCop(g, Vec{305, 300}, DirLeft)
	c.state = cArrest
	g.updateCop(c, 0.05)
	if !killer.gone || g.suspect != nil || g.lastKiller.kind != suspNone {
		t.Errorf("the killer was not taken (gone %v)", killer.gone)
	}
	if g.wanted != 0 {
		t.Error("the case is closed: nobody is wanted")
	}
}

func TestWantedNeverRaisedForAClearedPlayer(t *testing.T) {
	g := suspectGame()
	g.lastKiller = lastKill{kind: suspSlasher}
	g.slasher = &Slasher{state: slHunt}
	g.markPlayerWanted(70)
	if g.wanted != 0 {
		t.Error("a witness of an old kill raised the search level of a cleared player")
	}
	g.lastKiller = lastKill{kind: suspPlayer}
	g.markPlayerWanted(70)
	if g.wanted != 70 {
		t.Error("the player is the suspect: the search level must rise")
	}
	// the suspect is forgotten when he is gone
	g.lastKiller = lastKill{kind: suspSlasher}
	g.slasher.state = slGone
	g.updateSuspect()
	if g.lastKiller.kind != suspNone {
		t.Error("a gone killer stays the suspect")
	}
}
