package main

import (
	"math/rand"
	"testing"
)

// Traps do not choose: anybody who sets them off (or stands in the effect) is a victim.

func TestTrapKillsEveryoneInReach(t *testing.T) {
	g, tr := mineGame(t, false)
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime+0.2)
	caller := addGuestAt(g, Vec{410, 304})
	caller.state, caller.timer = gCall, 10 // on the phone to the police
	gd := &Guard{pos: Vec{395, 304}, cs: g.plainGuestCS()}
	g.guards = append(g.guards, gd)
	cop := &Cop{pos: Vec{400, 310}}
	soldier := &Cop{pos: Vec{405, 300}, kind: unitMilitary}
	medic := &Cop{pos: Vec{398, 308}, medic: true}
	g.cops = append(g.cops, cop, soldier, medic)
	an := &Animal{pos: Vec{402, 296}}
	g.animals = append(g.animals, an)
	g.slasher = &Slasher{state: slHunt, pos: Vec{392, 300}, cs: g.plainGuestCS(), hp: 3}
	tickProps(g, 0.1)
	if !tr.used {
		t.Fatal("must have gone off")
	}
	if !caller.dead || !gd.dead || !an.dead || g.slasher.state != slGone {
		t.Fatalf("caller=%v guard=%v animal=%v killer gone=%v", caller.dead, gd.dead, an.dead, g.slasher.state == slGone)
	}
	if len(g.cops) != 0 {
		t.Fatalf("police, soldier and medic must all die, %d left", len(g.cops))
	}
}

func TestTrapDetectsMedicAndKiller(t *testing.T) {
	for _, who := range []string{"medic", "soldier", "killer"} {
		g, tr := mineGame(t, false)
		g.armTrap(tr, g.p)
		tickProps(g, trapArmTime+0.2)
		switch who {
		case "medic":
			g.cops = append(g.cops, &Cop{pos: Vec{405, 304}, medic: true})
		case "soldier":
			g.cops = append(g.cops, &Cop{pos: Vec{405, 304}, kind: unitMilitary})
		case "killer":
			g.slasher = &Slasher{state: slHunt, pos: Vec{405, 304}, cs: g.plainGuestCS(), hp: 3}
		}
		tickProps(g, 0.1)
		if !tr.used {
			t.Errorf("%s must set the trap off", who)
		}
	}
}

func TestTrapIgnoresDeadAnimals(t *testing.T) {
	g, tr := mineGame(t, false)
	g.armTrap(tr, g.p)
	tickProps(g, trapArmTime+0.2)
	g.animals = append(g.animals, &Animal{pos: Vec{405, 304}, dead: true})
	tickProps(g, 0.1)
	if tr.used {
		t.Fatal("a dead animal must not set it off")
	}
}

func TestGasCloudChokesGuardsAnimalsAndKiller(t *testing.T) {
	g, _ := mineGame(t, false)
	at := Vec{500, 304}
	g.gas = append(g.gas, gasCloud{pos: at, t: 7})
	gd := &Guard{pos: at, cs: g.plainGuestCS()}
	g.guards = append(g.guards, gd)
	an := &Animal{pos: at}
	g.animals = append(g.animals, an)
	g.slasher = &Slasher{state: slHunt, pos: at, cs: g.plainGuestCS(), hp: 3}
	g.cops = append(g.cops, &Cop{pos: Vec{900, 900}, medic: true})
	for i := 0; i < 60*3; i++ {
		g.updateTraps2(1.0 / 60)
	}
	if !gd.dead || !an.dead || g.slasher.state != slGone {
		t.Fatalf("guard=%v animal=%v killer gone=%v", gd.dead, an.dead, g.slasher.state == slGone)
	}
}

// A blow that does not kill (the only one the player gives: J1 beating J2) is reported like a kill.
func TestBeatingIsSeenAndReportedLikeAKill(t *testing.T) {
	old, oldTwo := progressFrozen, twoPlayerMode
	t.Cleanup(func() { progressFrozen = old; twoPlayerMode = oldTwo })
	g := twoGame(true)
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	g.p.pos, g.p2.pos = Vec{200, 300}, Vec{215, 300}
	suspectCop(g, Vec{900, 900}, DirLeft)
	gu := suspectGuest(g, Vec{235, 300}, DirLeft) // looks at the fight
	if !g.hitPlayer(g.p, g.p2) {
		t.Fatal("the blow must land")
	}
	if !gu.sawKiller || gu.state != gAlarm || !g.playerAlert(g.p) {
		t.Fatalf("the witness must raise the alert (state %v)", gu.state)
	}
	if g.isSuspectP(g.p) {
		t.Fatal("no cuffs before the call")
	}
	finishCall(g, gu)
	if !g.policeCalled || !g.isSuspectP(g.p) || g.isSuspectP(g.p2) {
		t.Fatalf("the aggressor (J1) is the suspect: called=%v j1=%v j2=%v", g.policeCalled, g.isSuspectP(g.p), g.isSuspectP(g.p2))
	}
	if g.run.seen[0] != 0 {
		t.Fatal("a blow is not a kill: no 'seen killing' statistic")
	}
}

func TestBeatingWithNobodyWatchingDoesNothing(t *testing.T) {
	old, oldTwo := progressFrozen, twoPlayerMode
	t.Cleanup(func() { progressFrozen = old; twoPlayerMode = oldTwo })
	g := twoGame(true)
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	g.p.pos, g.p2.pos = Vec{200, 300}, Vec{215, 300}
	suspectGuest(g, Vec{235, 300}, DirRight) // looks away
	g.hitPlayer(g.p, g.p2)
	if g.playerAlert(g.p) || g.pendingWitnesses(g.p) != 0 {
		t.Fatal("nobody saw it: no alert")
	}
}

func ambulancesOn(g *Game) int {
	n := 0
	for _, v := range g.veh {
		if v.kind == 1 || v.kind == 3 {
			n++
		}
	}
	return n
}

func TestManyBodiesBringTwoAmbulances(t *testing.T) {
	for _, tc := range []struct{ bodies, want int }{{3, 1}, {ambTwinBodies - 1, 1}, {ambTwinBodies, 2}, {9, 2}} {
		g := suspectGame()
		g.pendingAmb = -1
		for i := 0; i < tc.bodies; i++ {
			g.bodies = append(g.bodies, &Body{reported: true})
		}
		g.maybeAmbulance() // the call
		g.time += 8
		g.maybeAmbulance() // they arrive
		if got := ambulancesOn(g); got != tc.want {
			t.Errorf("%d bodies: %d ambulances, want %d", tc.bodies, got, tc.want)
		}
		g.time += 60
		g.maybeAmbulance()
		g.time += 10
		g.maybeAmbulance()
		if got := ambulancesOn(g); got > 2 {
			t.Errorf("%d bodies: never more than two ambulances, got %d", tc.bodies, got)
		}
	}
}

// Rule (2026-11-08): nobody knows who set a trap, so its deaths never bring the police after the player.
func TestTrapDeathsRaiseNoHeat(t *testing.T) {
	old := progressFrozen
	t.Cleanup(func() { progressFrozen = old })
	g := suspectGame()
	g.policeCalled = false
	g.as = &Assets{}
	g.lvl.Props = nil
	tr := &Prop{Name: "bomb", K: &PropKind{Trap: "explode", H: 20}, X: 400, Y: 304, Foot: Rect{395, 300, 10, 8}, by: g.p}
	g.lvl.Props = append(g.lvl.Props, tr)
	var victims []*Guest
	for i := 0; i < 3; i++ {
		victims = append(victims, suspectGuest(g, Vec{396 + float64(i)*4, 304}, DirDown))
	}
	w1 := suspectGuest(g, Vec{330, 304}, DirRight) // witnesses looking at the blast
	w2 := suspectGuest(g, Vec{470, 304}, DirLeft)
	suspectGuest(g, Vec{600, 500}, DirLeft)
	gd := &Guard{pos: Vec{404, 300}, cs: g.plainGuestCS()}
	g.guards = append(g.guards, gd)
	cop := suspectCop(g, Vec{398, 308}, DirDown)
	g.slasher = &Slasher{state: slHunt, pos: Vec{402, 310}, cs: g.plainGuestCS(), hp: 3}
	g.p.pos = Vec{200, 300}
	g.triggerTrap(tr)
	for _, v := range victims {
		if !v.dead {
			t.Fatal("the blast must kill the guests")
		}
	}
	if !gd.dead || g.slasher.state != slGone {
		t.Fatal("guard and killer die too")
	}
	for _, c := range g.cops {
		if c == cop {
			t.Fatal("the cop dies too")
		}
	}
	for _, b := range g.bodies {
		if b.byPlayer {
			t.Fatal("a trap body must not be marked as the player's")
		}
	}
	for i := 0; i < 600; i++ {
		g.time += 0.1
		for _, gu := range []*Guest{w1, w2} {
			g.updateGuest(gu, 0.1)
		}
	}
	if g.wanted != 0 || g.stars != 0 || g.isSuspectP(g.p) || g.playerAlert(g.p) || g.playerHeat() {
		t.Fatalf("no heat from traps: wanted=%.0f stars=%d suspect=%v alert=%v", g.wanted, g.stars, g.isSuspectP(g.p), g.playerAlert(g.p))
	}
	if g.policeCalled && g.pendingWitnesses(g.p) != 0 {
		t.Fatal("no witness may call for the player")
	}
}

func TestPoisonedBowlNeverLosesItsPoison(t *testing.T) {
	g := poisonGame()
	g.lvl.Props = nil
	bowl := &Prop{Name: "bowl", K: &PropKind{Poison: true}, Foot: Rect{300, 300, 10, 8}, armT: 40}
	g.lvl.Props = append(g.lvl.Props, bowl)
	for i := 0; i < 60*120; i++ { // two minutes
		g.updateProps(1.0 / 60)
	}
	if bowl.armT <= 0 {
		t.Fatal("the bowl lost its poison")
	}
	o := poisonGuest(g, Vec{305, 304})
	o.state, o.timer, o.dur = gDrink, 0.05, 1
	for i := 0; i < 5 && o.poison <= 0; i++ {
		g.updateGuest(o, 1.0/30)
	}
	if o.poison <= 0 {
		t.Fatal("after two minutes the drinker is still poisoned")
	}
}

func TestSnakeChanceAndPity(t *testing.T) {
	snakeReseed(5)
	t.Cleanup(func() { snakeReseed(0) })
	r := rand.New(rand.NewSource(9))
	misses, hits, run, maxRun := 0, 0, 0, 0
	n := 4000
	for i := 0; i < n; i++ {
		if snakeDecide(r, &misses) {
			hits++
			run = 0
		} else {
			run++
			if run > maxRun {
				maxRun = run
			}
		}
	}
	if maxRun > snakePityAfter {
		t.Errorf("%d parties in a row without snake, pity is %d", maxRun, snakePityAfter)
	}
	if rate := float64(hits) / float64(n); rate < 0.2 || rate > 0.4 {
		t.Errorf("snake rate %.2f", rate)
	}
	m := snakePityAfter
	if !snakeDecide(rand.New(rand.NewSource(1)), &m) || m != 0 {
		t.Error("pity must force one and reset the counter")
	}
}

func TestSnakeDoesNotMoveGameRng(t *testing.T) {
	snakeReseed(3)
	t.Cleanup(func() { snakeReseed(0) })
	g := poisonGame()
	a, b := rand.New(rand.NewSource(2)), rand.New(rand.NewSource(2))
	g.rng = a
	for i := 0; i < 20; i++ {
		snakeDecide(snakeDice(), &snakeMisses)
	}
	if a.Int63() != b.Int63() {
		t.Fatal("the snake dice moved g.rng")
	}
}

func TestCopTiredCooldown(t *testing.T) {
	if copTiredCooldown != 3 {
		t.Fatal("cooldown is 3 s")
	}
	g := suspectGame()
	c := suspectCop(g, Vec{230, 300}, DirLeft)
	c.state, c.timer, c.tiredCD = cTired, 5, copTiredCooldown
	c.onSceneT = 0
	g.wanted = 50
	g.time = 50
	g.p.pos = Vec{200, 300}
	g.updateCop(c, 1.0)
	g.updateCop(c, 1.0)
	if c.state == cChase {
		t.Fatal("a tired cop cannot start running again")
	}
	c.state, c.tiredCD = cSweep, 2 // rested but inside the 3 s
	g.updateCop(c, 0.5)
	if c.state == cChase {
		t.Fatal("no chase before the cooldown ends")
	}
	c.tiredCD, c.pos, c.dir = 0, Vec{230, 300}, DirLeft
	g.updateCop(c, 0.1)
	if c.state != cChase {
		t.Fatalf("after the cooldown he chases again (state %v)", c.state)
	}
}

func TestDanceBotherNeedsInsistingAndIsRare(t *testing.T) {
	old := progressFrozen
	t.Cleanup(func() { progressFrozen = old })
	count := func(dances int, seed int64) (punches int) {
		for s := int64(0); s < 200; s++ {
			g := suspectGame()
			g.p.pos = Vec{200, 300}
			gu := suspectGuest(g, Vec{220, 300}, DirLeft)
			r := rand.New(rand.NewSource(seed + s))
			g.time = 10
			for i := 0; i < dances; i++ {
				g.time += 4.5
				g.danceAnnoy(g.p, r)
			}
			if gu.state == gPunch {
				punches++
			}
		}
		return
	}
	if n := count(int(danceLimit)-1, 1); n != 0 {
		t.Fatalf("dancing a little never provokes a punch (%d)", n)
	}
	n := count(12, 1000)
	if n == 0 || n >= 120 {
		t.Fatalf("insisting sometimes (not always) gets a punch: %d of 200", n)
	}
	// cooldown after a reaction
	g := suspectGame()
	g.p.pos = Vec{200, 300}
	g.p.danceCool = 1e9
	gu := suspectGuest(g, Vec{220, 300}, DirLeft)
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		g.time += 4.5
		g.danceAnnoy(g.p, r)
	}
	if gu.state == gPunch {
		t.Fatal("no reaction during the cooldown")
	}
}

func TestIdleTimer(t *testing.T) {
	tm := 0.0
	for i := 0; i < 9*60; i++ {
		tm = idleStep(tm, 1.0/60, false)
	}
	if tm >= idleShowAfter {
		t.Fatal("9 s still: no marker yet")
	}
	for i := 0; i < 2*60; i++ {
		tm = idleStep(tm, 1.0/60, false)
	}
	if tm < idleShowAfter {
		t.Fatal("11 s still: marker")
	}
	if idleStep(tm, 1.0/60, true) != 0 {
		t.Fatal("moving resets it")
	}
	g := twoGame(true)
	g.mode = ModePlay
	for i := 0; i < 11*60; i++ {
		g.p2.pos.X += 0.5 // only J2 walks
		g.updateIdle(1.0 / 60)
	}
	if g.p.idleT < idleShowAfter || g.p2.idleT != 0 {
		t.Fatalf("per player: j1=%.1f j2=%.1f", g.p.idleT, g.p2.idleT)
	}
}

// A falling piano (and every accident) never blames the player: no alert, no BUSCADO, no cuffs, no stars.
func TestFallingPianoDoesNotBlameThePlayer(t *testing.T) {
	old := progressFrozen
	t.Cleanup(func() { progressFrozen = old })
	g := suspectGame()
	g.policeCalled = false
	g.as = &Assets{}
	g.p.pos = Vec{250, 300} // right next to it, in plain view
	under := suspectGuest(g, Vec{300, 300}, DirDown)
	w1 := suspectGuest(g, Vec{330, 300}, DirLeft)
	w2 := suspectGuest(g, Vec{270, 300}, DirRight)
	cop := suspectCop(g, Vec{280, 310}, DirLeft)
	g.pianos = append(g.pianos, pianoDrop{pos: Vec{300, 300}})
	for i := 0; i < 90; i++ {
		g.time += 1.0 / 30
		g.updateTraps2(1.0 / 30)
		g.updateGuest(w1, 1.0/30)
		g.updateGuest(w2, 1.0/30)
		g.updateCop(cop, 1.0/30)
	}
	if !under.dead || len(g.bodies) == 0 {
		t.Fatal("the piano must kill the guest under it")
	}
	for i := 0; i < 600; i++ {
		g.time += 0.1
		g.updateGuest(w1, 0.1)
		g.updateGuest(w2, 0.1)
		g.updateCop(cop, 0.1)
	}
	if g.wanted != 0 || g.stars != 0 || g.isSuspectP(g.p) || g.playerAlert(g.p) || g.playerHeat() {
		t.Fatalf("blamed for an accident: wanted=%.0f stars=%d", g.wanted, g.stars)
	}
}

func TestNobodyArrivesWithTenOrFewerAlive(t *testing.T) {
	g := suspectGame()
	g.mode = ModePlay
	g.time = 100
	for i := 0; i < lateJoinMin+1; i++ {
		suspectGuest(g, Vec{300 + float64(i)*10, 300}, DirDown)
	}
	if g.arrivalsClosed() {
		t.Fatal("11 alive: arrivals still open")
	}
	g.guests[0].dead = true // 10 left
	if !g.arrivalsClosed() || g.guestRoom() != 0 {
		t.Fatal("10 alive: closed")
	}
	n := len(g.guests)
	g.addGuestAt(Vec{400, 300}) // a limousine group that was already on its way
	g.arrivalT, g.arrivals = 0, 0
	g.updateArrivals(0.1)
	if len(g.guests) != n || len(g.veh) != 0 || len(g.elevators) != 0 {
		t.Fatal("no new guests, limousines or elevators")
	}
}
