package main

import "testing"

// Bug (2026-11-09) "lots of people see bodies and nobody calls the police". Two causes:
//  1. a caller pulled out of his call by something else (a party event, a kiss, a zombie
//     bite...) stayed as g.callerGuest for ever and callerAvailable() said no to everybody;
//  2. when the officers died (the special killer, a trap, a power) their parked car or
//     helicopter kept policeCalled on for the rest of the party.

func TestStaleCallerDoesNotBlockWitnesses(t *testing.T) {
	g := playGame(t, 0, 2)
	var gu *Guest
	for _, o := range g.guests {
		if !o.out() && o.perf == "" {
			gu = o
			break
		}
	}
	g.becomeCaller(gu)
	if g.callerAvailable() {
		t.Fatal("while he is on his way to the phone nobody else calls")
	}
	gu.state = gGoto // a party event took him somewhere else
	if !g.callerAvailable() || g.callerGuest != nil {
		t.Fatal("a caller who is no longer calling must not block the other witnesses")
	}
	g.becomeCaller(gu)
	gu.state = gZombie
	if !g.callerAvailable() {
		t.Fatal("a caller turned into a zombie must not block the other witnesses")
	}
}

func TestDeadCrewDoesNotKeepTheCallOpen(t *testing.T) {
	g := playGame(t, 0, 4)
	v := victimInView(g)
	g.killGuest(v, true)
	g.p.pos = g.lvl.Def.Start
	g.raisePolice(v.pos, false)
	if !runPlay(g, 40, func() bool {
		for _, c := range g.cops {
			if !c.medic {
				return true
			}
		}
		return false
	}) {
		t.Fatal("the police never came")
	}
	// the special killer kills every officer
	for _, c := range append([]*Cop(nil), g.cops...) {
		if !c.medic {
			g.removeCop(c)
		}
	}
	g.updatePlay(1.0 / 60) // one frame: the watchdog closes the call
	if g.policeCalled || g.policeOnCase() {
		t.Fatalf("with every officer dead and nobody on the way the next witness must be able to call (called %v, pending car %.1f unit %d, %d vehicles)", g.policeCalled, g.pendingCar, g.pendingUnit, len(g.veh))
	}
	for _, vh := range g.veh {
		if (vh.kind == 0 || vh.kind == 2) && vh.state == 1 {
			t.Fatal("the crewless police car must leave (it would hold the slot or the helipad)")
		}
	}
	// and a new body seen by the guests is reported again
	v2 := victimInView(g)
	if v2 == nil {
		t.Fatal("no second victim")
	}
	g.killGuest(v2, true)
	g.p.pos = g.lvl.Def.Start
	if !runPlay(g, 45, func() bool { return g.policeCalled }) {
		t.Fatal("nobody called the police about the new body")
	}
}

// a body of the player in front of several guests: somebody runs to call within seconds
func TestWitnessesCallOnRealParties(t *testing.T) {
	calls, n := 0, 0
	for _, lv := range []int{0, 2, 4, 7, 11, 16} {
		g := playGame(t, lv, int64(lv+1))
		v := victimInView(g)
		if v == nil {
			continue
		}
		n++
		g.killGuest(v, true)
		g.p.pos = g.lvl.Def.Start
		alarmed := runPlay(g, 6, func() bool { return g.callerGuest != nil || g.callerGuard != nil || g.policeCalled })
		if alarmed && runPlay(g, 40, func() bool { return g.policeCalled }) || g.policeCalled {
			calls++
		} else {
			t.Logf("level %d: no call (alarmed %v)", lv, alarmed)
		}
	}
	if n < 4 || calls < n-1 {
		t.Fatalf("only %d of %d parties called the police", calls, n)
	}
}
