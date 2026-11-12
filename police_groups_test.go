package main

import (
	"flag"
	"math/rand"
	"testing"
)

// the user's examples: 60 guests split in equal stretches between the unlocked groups
func TestPoliceGroupForStretches(t *testing.T) {
	cases := []struct{ n, last int }{{1, 60}, {2, 30}, {3, 20}, {4, 15}, {5, 12}}
	for _, c := range cases {
		for k := 1; k <= 60; k++ {
			want := (k - 1) / c.last
			if c.n == 1 {
				want = 0
			}
			if got := policeGroupFor(k, 60, c.n); got != want {
				t.Fatalf("N=%d kill %d: group %d, want %d", c.n, k, got, want)
			}
		}
	}
	// police+FBI: police up to the 30th kill, FBI from the 31st to the 60th
	if policeGroupFor(30, 60, 2) != 0 || policeGroupFor(31, 60, 2) != 1 || policeGroupFor(60, 60, 2) != 1 {
		t.Fatal("60 guests, 2 groups: 1-30 police, 31-60 FBI")
	}
	// no kill yet, more kills than guests (guards count too) and nonsense input stay in range
	if policeGroupFor(0, 60, 5) != 0 || policeGroupFor(99, 60, 5) != 4 || policeGroupFor(5, 0, 3) != 0 {
		t.Fatal("out of range input")
	}
}

func withGroups(t *testing.T, n int) {
	t.Helper()
	old := dbgPoliceGroups
	dbgPoliceGroups = n
	t.Cleanup(func() { dbgPoliceGroups = old })
}

// a fresh save: only the local police, whatever the kills
func TestPoliceGroupsOnlyPolice(t *testing.T) {
	g := freshShopGame()
	withGroups(t, 0)
	if gs := policeGroups(); len(gs) != 1 || gs[0] != unitPolice {
		t.Fatalf("a fresh save has only the police, got %v", gs)
	}
	g.total = 60
	g.policeCalled = true
	g.pendingUnit, g.pendingCar = -1, -1
	g.cops = []*Cop{{kind: unitPolice, state: cSweep}}
	for k := 0; k <= 70; k++ {
		g.kills = k
		if u := g.groupUnitNow(); u != unitPolice {
			t.Fatalf("kill %d: unit %d with only the police unlocked", k, u)
		}
		g.updatePoliceGroup()
		if g.pendingUnit >= 0 || g.respKind != unitPolice {
			t.Fatalf("kill %d: something above the police was sent (pending %d)", k, g.pendingUnit)
		}
	}
}

// the groups come from the kills milestones in the saved progress: 500 FBI, 1000 military, 1500 SWAT, 2000 rangers
func TestPoliceGroupUnlockThresholds(t *testing.T) {
	withGroups(t, 0)
	steps := []struct{ kills, groups int }{{0, 1}, {499, 1}, {500, 2}, {999, 2}, {1000, 3}, {1499, 3}, {1500, 4}, {1999, 4}, {2000, 5}, {5000, 5}}
	for _, s := range steps {
		g := freshShopGame()
		g.progress.Stats["kills"] = s.kills
		g.trophyDirty = true
		g.checkTrophies()
		g.syncShop()
		if got := len(policeGroups()); got != s.groups {
			t.Fatalf("%d kills: %d groups, want %d (%v)", s.kills, got, s.groups, policeGroups())
		}
	}
}

// -unlocked N simulates the groups without touching the progress, and freezes it like any debug flag
func TestPoliceGroupsDebugFlag(t *testing.T) {
	g := freshShopGame()
	withGroups(t, 3)
	if gs := policeGroups(); len(gs) != 3 || gs[1] != unitFBI || gs[2] != unitMilitary {
		t.Fatalf("-unlocked 3: %v", gs)
	}
	if g.hasOwned("police:1") || len(g.progress.Owned) != 0 {
		t.Fatal("the debug flag must not write anything to the progress")
	}
	dbgPoliceGroups = 9
	if len(policeGroups()) != len(groupStat) {
		t.Fatal("more groups than exist are capped")
	}
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	fs.Int("unlocked", 0, "")
	if err := fs.Parse([]string{"-unlocked", "2"}); err != nil {
		t.Fatal(err)
	}
	if !debugFlagsActive(fs) {
		t.Fatal("-unlocked is a debug flag: the progress must be frozen")
	}
}

// every higher group searches a bit harder than the one before
func TestPoliceGroupsSearchHarder(t *testing.T) {
	for u := unitFBI; u < len(groupStat); u++ {
		a, b := groupStat[u-1], groupStat[u]
		if b.crew < a.crew || b.sight <= a.sight || b.pace <= a.pace {
			t.Fatalf("group %d does not search harder than %d: %+v vs %+v", u, u-1, b, a)
		}
		if b.sight > 1.3 || b.pace > 1.3 {
			t.Fatalf("group %d: the boost must stay small: %+v", u, b)
		}
	}
}

// playGame: a real party loaded headless, in play, the player untouchable.
func playGame(t *testing.T, level int, seed int64) *Game {
	t.Helper()
	g := layerGame(t, false)
	// the audit game is shared by many tests: give back what this one changes
	god, rng := g.god, g.rng
	t.Cleanup(func() {
		g.god, g.rng = god, rng
		g.cops, g.veh, g.bodies = nil, nil, nil
		g.policeCalled, g.pendingCar, g.pendingUnit, g.pendingAmb = false, -1, -1, -1
		g.ambCalled, g.callerGuest, g.callerGuard = false, nil, nil
		g.mode = ModeMenu
	})
	// seeded BEFORE loadLevel too: the level draws its guests (stars, hippies, kids...) from g.rng, and the shared
	// game's rng carries whatever the previous tests left, so the party (and g.total) changed with the test order
	g.rng = rand.New(rand.NewSource(seed))
	g.loadLevel(level)
	g.rng = rand.New(rand.NewSource(seed))
	g.mode = ModePlay
	g.titleT = 0
	g.god = true
	g.time = 30
	return g
}

// victimInView: a guest with at least three others close by with a clear line to him.
func victimInView(g *Game) *Guest {
	for _, v := range g.guests {
		if v.out() || v.perf != "" {
			continue
		}
		n := 0
		for _, o := range g.guests {
			if o != v && !o.out() && dist(o.pos, v.pos) < 90 && g.lvl.LOS(o.pos, v.pos) {
				n++
			}
		}
		if n >= 3 {
			return v
		}
	}
	return nil
}

func runPlay(g *Game, secs float64, stop func() bool) bool {
	for i := 0; i < int(secs*60); i++ {
		g.updatePlay(1.0 / 60)
		if stop() {
			return true
		}
	}
	return false
}

// A call: the local police come first (1 officer for one body), then the group of the
// stretch (police+FBI, 60 guests, 40 kills -> FBI), and when the FBI gets out the police go.
func TestPoliceFirstResponseThenGroup(t *testing.T) {
	withGroups(t, 2)
	g := playGame(t, 0, 3)
	v := victimInView(g)
	if v == nil {
		t.Fatal("no victim")
	}
	g.killGuest(v, true)
	g.p.pos = g.lvl.Def.Start
	g.kills, g.total = 40, 60
	g.raisePolice(v.pos, false)
	if g.respKind != unitPolice {
		t.Fatal("the first response is the local police")
	}
	var first []*Cop
	if !runPlay(g, 40, func() bool {
		for _, c := range g.cops {
			if !c.medic {
				first = append(first, c)
			}
		}
		return len(first) > 0
	}) {
		t.Fatal("the police never got there")
	}
	for _, c := range first {
		if c.kind != unitPolice {
			t.Fatalf("first on scene: unit %d, want the local police", c.kind)
		}
	}
	if len(first) < 1 || len(first) > 2 {
		t.Fatalf("first response: %d officers, want 1-2", len(first))
	}
	if !runPlay(g, 40, func() bool {
		for _, c := range g.cops {
			if c.kind == unitFBI {
				return true
			}
		}
		return false
	}) {
		t.Fatalf("the FBI was never sent (pending %d resp %d)", g.pendingUnit, g.respKind)
	}
	// 2026-11-11: an officer relieved before his 20 s seals off the place (cCordon) until he has
	// done them, but he is off the case: no chasing, no searching (TestEscalationOldTierStaysMinimum)
	for _, c := range first {
		if c.state != cLeave && c.state != cCordon && containsCop(g.cops, c) {
			t.Fatalf("the police stay on scene (state %d) after the FBI got out", c.state)
		}
	}
	if g.respKind != unitFBI {
		t.Fatalf("the case is the FBI's now, got %d", g.respKind)
	}
}

func containsCop(cs []*Cop, c *Cop) bool {
	for _, o := range cs {
		if o == c {
			return true
		}
	}
	return false
}

// below the FBI's stretch only the police come, even with the FBI unlocked
func TestPoliceGroupBelowStretchOnlyPolice(t *testing.T) {
	withGroups(t, 2)
	g := playGame(t, 0, 5)
	v := victimInView(g)
	g.killGuest(v, true)
	g.p.pos = g.lvl.Def.Start
	g.kills, g.total = 10, 60
	g.raisePolice(v.pos, false)
	runPlay(g, 60, func() bool { return false })
	for _, c := range g.cops {
		if !c.medic && c.kind != unitPolice {
			t.Fatalf("unit %d came at 10 kills of 60 with 2 groups", c.kind)
		}
	}
	if g.pendingUnit > unitPolice {
		t.Fatalf("pending %d", g.pendingUnit)
	}
}

// two bodies found -> two officers in the first car; one body -> one
func TestPoliceFirstCrewByBodies(t *testing.T) {
	g := playGame(t, 0, 1)
	g.bodies = nil
	if g.groupCrew(unitPolice) != 1 {
		t.Fatal("no body: one officer")
	}
	g.bodies = []*Body{{reported: true}}
	if g.groupCrew(unitPolice) != 1 {
		t.Fatal("one body: one officer")
	}
	g.bodies = append(g.bodies, &Body{reported: true}, &Body{reported: true})
	if g.groupCrew(unitPolice) != 2 {
		t.Fatal("several bodies: two officers")
	}
	if g.groupCrew(unitMilitary) != 3 || g.groupCrew(unitRangers) != 5 {
		t.Fatal("the higher groups keep their crews")
	}
}
