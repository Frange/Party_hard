package main

import "testing"

// two players, the police called and one officer on the scene
func jailGame(t *testing.T) (*Game, *Cop) {
	old := jailHold // the custody flag is a global: leave it as found
	jailHold = false
	t.Cleanup(func() { jailHold = old })
	g := twoGame(true)
	g.policeCalled, g.pendingCar, g.pendingUnit = true, -1, -1
	g.p.pos, g.p2.pos = Vec{200, 300}, Vec{240, 300}
	c := &Cop{pos: Vec{220, 300}, dir: DirLeft, state: cSweep, maxChase: 6}
	g.cops = append(g.cops, c)
	return g, c
}

func seenKill(g *Game, p *Player) {
	g.withPlayer(p, func() { g.suspectSeen(suspPlayer, nil) })
}

func TestCuffsGoToOneAndThenToBoth(t *testing.T) {
	g, _ := jailGame(t)
	p1, p2 := g.p, g.p2
	g.wanted = 70 // the call to the police raised the search
	seenKill(g, p1)
	if !p1.jail.sus || p2.jail.sus {
		t.Fatal("the cuffs go only to the one they saw")
	}
	if !g.isSuspectP(p1) || g.isSuspectP(p2) {
		t.Fatalf("only the first is a suspect (wanted=%.0f)", g.wanted)
	}
	seenKill(g, p2) // they see the other one too
	if !g.isSuspectP(p1) || !g.isSuspectP(p2) {
		t.Fatal("seeing the second one too: the cuffs are for both")
	}
}

func TestKillerSeenClearsBothAndTheNextKillIsOnlyHis(t *testing.T) {
	g, _ := jailGame(t)
	p1, p2 := g.p, g.p2
	seenKill(g, p1)
	seenKill(g, p2)
	g.slasher = &Slasher{state: slHunt, pos: Vec{400, 300}, need: 2}
	g.suspectSeen(suspSlasher, nil)
	if g.isSuspectP(p1) || g.isSuspectP(p2) || g.wanted != 0 {
		t.Fatal("a killer seen killing takes the cuffs off both")
	}
	seenKill(g, p2) // he kills again and somebody sees it
	if g.isSuspectP(p1) || !g.isSuspectP(p2) {
		t.Fatalf("only the one who killed again is a suspect (p1=%v p2=%v)", p1.jail.sus, p2.jail.sus)
	}
	if p := g.suspectProblems(); len(p) != 0 {
		t.Fatal(p)
	}
}

func TestTwoKillersKeepTheCaseOpen(t *testing.T) {
	g, _ := jailGame(t)
	g.slasher = &Slasher{state: slGone}
	g.slot2.s = &Slasher{state: slHunt, pos: Vec{400, 300}}
	g.lastKiller = lastKill{kind: suspSlasher}
	g.updateSuspect()
	if g.lastKiller.kind != suspSlasher {
		t.Fatal("the second killer is still on the loose")
	}
}

func TestNobodySawItTheLastKillerIsTheSuspect(t *testing.T) {
	g, _ := jailGame(t)
	g.p.jail.killT = 5
	g.p2.jail.killT = 9
	g.wanted = 0
	g.markPlayerWanted(70)
	if !g.p2.jail.sus || g.p.jail.sus {
		t.Fatal("with no witness of the kill the police want the last one who killed")
	}
}

func TestOfficerChasesTheSuspectNotTheOther(t *testing.T) {
	g, c := jailGame(t)
	p1, p2 := g.p, g.p2
	g.wanted = 70
	p2.jail.sus = true
	if got := g.copTarget(c); got != p2 {
		t.Fatal("the officer must go after the suspect, not the nearest player")
	}
	if g.copInterest(p1) || !g.copInterest(p2) {
		t.Fatal("interest follows the cuffs")
	}
	p1.jail.sus = true
	if got := g.copTarget(c); got != p1 && got != p2 {
		t.Fatal("nobody")
	}
}

func TestCatchHoldsHimAndPartyGoesOn(t *testing.T) {
	g, c := jailGame(t)
	p1, p2 := g.p, g.p2
	p1.jail.sus = true
	g.wanted = 70
	g.copCatch(c, p1)
	if !p1.jail.held || p1.dead || g.mode != ModePlay {
		t.Fatal("two players: the catch holds him, it does not end the party")
	}
	if p1.jail.sus || g.isSuspectP(p1) {
		t.Fatal("a prisoner is not a suspect any more")
	}
	if !jailHold || c.mayLeave() {
		t.Fatal("the officers stay while somebody is held")
	}
	if g.nearestPlayer(p1.pos) != p2 {
		t.Fatal("the others look at the free one")
	}
}

func TestPrisonBreakAfterAMinute(t *testing.T) {
	g, c := jailGame(t)
	p1 := g.p
	orig := p1.cs
	g.jailPlayer(c, p1)
	g.updateJail(30)
	if !p1.jail.held {
		t.Fatal("still in custody after 30 s")
	}
	g.updateJail(30.5)
	if p1.jail.held || !p1.jail.lastLife {
		t.Fatal("after a minute he escapes and it is his last life")
	}
	if p1.cs == orig || p1.speedMul() >= 1 {
		t.Fatal("prisoner suit and slower")
	}
	if p1.speedMul() != convictSpeed {
		t.Fatal("speed")
	}
	if jailHold {
		t.Fatal("nobody is held any more")
	}
}

func TestCaughtAgainOrDyingAfterTheEscapeIsFinal(t *testing.T) {
	g, c := jailGame(t)
	p1, p2 := g.p, g.p2
	p1.jail.lastLife = true
	g.copCatch(c, p1)
	if !p1.dead || p1.jail.held {
		t.Fatal("no second chance after the escape")
	}
	if g.mode != ModePlay {
		t.Fatal("the other one is still free: the party goes on")
	}
	g.copCatch(c, p2) // the other one is caught with nobody left outside
	if !p2.dead || g.mode != ModeLose {
		t.Fatalf("nobody outside: game over (mode=%d)", g.mode)
	}
}

func TestPartnerDiesWhileHeldIsGameOver(t *testing.T) {
	g, c := jailGame(t)
	g.jailPlayer(c, g.p)
	g.withPlayer(g.p2, func() { g.die("un piano") })
	if g.mode != ModeLose {
		t.Fatal("the free one died while the other was in custody: both are out")
	}
}

func TestBothCaughtIsGameOver(t *testing.T) {
	g, c := jailGame(t)
	g.jailPlayer(c, g.p)
	g.copCatch(c, g.p2)
	if g.mode != ModeLose {
		t.Fatal("both caught: the party is lost")
	}
}

func TestPartnerFreesHimUnseen(t *testing.T) {
	g, c := jailGame(t)
	p1, p2 := g.p, g.p2
	orig := p1.cs
	g.jailPlayer(c, p1)
	c.pos, c.dir = Vec{700, 100}, DirUp // far and looking away
	p2.pos = Vec{p1.jail.pos.X + 10, p1.jail.pos.Y}
	for i := 0; i < 30 && p1.jail.held; i++ {
		g.jailRescue(p1, p2, true, 0.2)
	}
	if p1.jail.held {
		t.Fatalf("5 s of E next to him must free him (progress %.1f)", p1.jail.free)
	}
	if p1.jail.lastLife || p1.cs != orig {
		t.Fatal("freed by the partner: no suit and no ball, no last-life mark")
	}
}

func TestRescueFailsIfAnOfficerSees(t *testing.T) {
	g, c := jailGame(t)
	p1, p2 := g.p, g.p2
	g.jailPlayer(c, p1)
	p2.pos = Vec{p1.jail.pos.X + 10, p1.jail.pos.Y}
	c.pos, c.dir = Vec{p2.pos.X + 30, p2.pos.Y}, DirLeft // looking straight at him
	for i := 0; i < 40; i++ {
		g.jailRescue(p1, p2, true, 0.2)
	}
	if !p1.jail.held {
		t.Fatal("an officer saw them: the rescue cannot work")
	}
	// too far: nothing either
	c.pos, c.dir = Vec{900, 100}, DirUp
	p2.pos = Vec{p1.jail.pos.X + 200, p1.jail.pos.Y}
	g.jailRescue(p1, p2, true, 3)
	if !p1.jail.held || p1.jail.free != 0 {
		t.Fatal("too far to free him")
	}
}

func TestOnePlayerCatchEndsTheParty(t *testing.T) {
	g := twoGame(false)
	c := &Cop{pos: Vec{210, 300}, dir: DirLeft, state: cChase}
	g.cops = append(g.cops, c)
	g.copCatch(c, g.p)
	if g.mode != ModeLose {
		t.Fatal("one player: a catch ends the party as always")
	}
	if g.copTarget(c) != g.p || !g.copInterest(g.p) == (g.wanted > 0) && g.wanted > 0 {
		t.Fatal("one player: the target is always him")
	}
}

func TestShakeOffAndCloseCaseClearTheCuffs(t *testing.T) {
	g, _ := jailGame(t)
	p1, p2 := g.p, g.p2
	g.wanted = 70
	p1.jail.sus, p2.jail.sus = true, true
	g.shakeOffPolice(p1)
	if p1.jail.sus || !p2.jail.sus || g.wanted == 0 {
		t.Fatal("one wardrobe change frees only him: the other one is still wanted")
	}
	g.shakeOffPolice(p2)
	if g.wanted != 0 {
		t.Fatal("nobody left to hunt")
	}
	p1.jail.sus, p2.jail.sus = true, true
	g.closeCase()
	if p1.jail.sus || p2.jail.sus {
		t.Fatal("a closed case leaves no suspects")
	}
}

func TestPrisonerSuitIsStriped(t *testing.T) {
	g, _ := jailGame(t)
	cs := g.prisonerCS(g.p)
	if cs == nil || cs.Spec == nil || cs.Spec.Pattern != 2 || cs.Spec.Top != TopLong {
		t.Fatal("the prisoner suit is a striped long-sleeved top")
	}
	if g.prisonerCS(g.p) != cs {
		t.Fatal("the suit is cached")
	}
	if g.prisonerCS(g.p2) == cs {
		t.Fatal("each player keeps his own build")
	}
}

func TestBallFollowsAtChainLength(t *testing.T) {
	g, _ := jailGame(t)
	p := g.p
	p.jail.lastLife = true
	p.jail.ball = p.pos
	p.pos.X += 60
	g.updateBall(p, 0.016)
	if d := dist(p.jail.ball, p.pos); d > chainLen+0.01 {
		t.Fatalf("the ball must be at a chain's length, it is %.1f away", d)
	}
}
