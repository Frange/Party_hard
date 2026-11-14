package main

import (
	"fmt"
	"testing"
)

// The user's party 1 asked for 116 kills (BAJAS 85/116, later 122/133): g.total, the kill target of the HUD,
// is everybody the party ever had, and the late arrivals kept adding to it while the dead made room under the
// live cap. Now the whole target is capped (cap.go goalMax: 65 for one player): the start leaves room for the
// arrivals and every route that adds people (taxis, bus, train, elevators, God, famous powers, themed groups,
// kids, cheerleaders, exhibitionists, Ghostbusters) stops when it is full.

// killTargetLevels: the 26 campaign parties, whatever the previous tests left in levelDefs.
func killTargetLevels(t *testing.T) int {
	t.Helper()
	if len(levelDefs) < len(campaignLevels) {
		// the full campaign is what newGame builds too (arrival_test.go does the same): it is NOT put back. Putting
		// back the 5 classic levels undid the shared audit game built meanwhile (playGame) and left campaignN = 26
		// with 5 levels: the medal tests saw "no campaign" and TestPirateBoatArrivals indexed out of range (-shuffle)
		levelDefs = builtinLevels()
	}
	if len(campaignLevels) != 28 {
		t.Fatalf("the campaign has %d parties, want 28", len(campaignLevels))
	}
	return len(campaignLevels)
}

// killTargetStress plays the level with a kill every 2 s and every arrival route forced again and again.
func killTargetStress(t *testing.T, g *Game, secs float64) (maxTotal int) {
	t.Helper()
	maxTotal = g.total
	for i := 0; i < int(secs*60) && g.mode == ModePlay; i++ {
		g.updatePlay(1.0 / 60)
		switch i % 120 {
		case 0: // a kill every 2 seconds
			for _, gu := range g.guests {
				if !gu.out() && gu.perf == "" && gu.py == nil {
					g.killGuest(gu, true)
					break
				}
			}
		case 30:
			for k := 0; k < 4; k++ {
				g.addGuestAt(g.lvl.RandomRoam(g.rng))
			}
		case 50:
			g.forceEv = evGod + 1 // the hands of God: 5-10 new guests
		case 70:
			g.powerFamous()
			g.pwFamAt = -999
		case 90:
			g.startTheme(&Event{})
		case 100:
			g.limoArrives(&Vehicle{alt: 1 + g.rng.Intn(6)})
		case 110:
			g.addKidAt("sam_sam", g.lvl.RandomRoam(g.rng))
		}
		if g.total > maxTotal {
			maxTotal = g.total
		}
		if g.total > g.goalMax() {
			t.Fatalf("%s: total %d above the cap %d at %.0f s", g.lvl.Def.Title, g.total, g.goalMax(), float64(i)/60)
		}
	}
	return maxTotal
}

// killAll ends the party: every guest (performers too) and every guard; the killer, if he came, is caught.
func killAll(g *Game) {
	for g.mode == ModePlay && g.aliveCount() > 0 {
		hit := false
		for _, gu := range g.guests {
			if !gu.out() && gu.py == nil {
				g.killGuest(gu, true)
				hit = true
				break
			}
		}
		if !hit {
			for _, gd := range g.guards {
				if !gd.dead && gd.vip == 0 {
					g.killGuard(gd)
					hit = true
					break
				}
			}
		}
		if !hit {
			for _, s := range g.allSlashers() {
				if s.state == slHunt || s.state == slArrive {
					s.state = slGone
					hit = true
				}
			}
			g.checkDone()
		}
		if !hit {
			return
		}
	}
}

// Every one of the 28 parties: the start keeps room for the arrivals (and is a real party: >= 40), the
// target never passes 65 however many arrivals are forced, the HUD writes exactly kills/total, and killing
// the people of the target ends the party (nobody impossible to reach is left in it).
func TestKillTargetCappedAllLevels(t *testing.T) {
	n := killTargetLevels(t)
	for i := 0; i < n; i++ {
		g := playGame(t, i, int64(7+i))
		name := fmt.Sprintf("L%02d %s", i+1, g.lvl.Def.Title)
		capN := goalCap + bigCrowd[g.lvl.Def.Icon] // 65, or 95 in the big-crowd parties (cap.go)
		if g.goalMax() != capN {
			t.Fatalf("%s: one player cap %d, want %d", name, g.goalMax(), capN)
		}
		if g.total < 40 || g.total > capN-g.startRoom() {
			t.Errorf("%s: starts with a target of %d, want 40..%d", name, g.total, capN-g.startRoom())
		}
		mx := killTargetStress(t, g, 100)
		if mx > capN {
			t.Errorf("%s: target reached %d", name, mx)
		}
		rec := hudRecord(g)
		want := fmt.Sprintf("%d/%d", g.total-g.aliveCount(), g.total)
		found := false
		for _, tx := range rec.texts {
			found = found || tx.s == want
		}
		if !found {
			t.Errorf("%s: the HUD does not write %q", name, want)
		}
		total := g.total
		killAll(g)
		if g.aliveCount() != 0 {
			t.Errorf("%s: %d left alive after killing everybody", name, g.aliveCount())
		}
		if g.total != total || g.total > capN {
			t.Errorf("%s: total moved from %d to %d at the end", name, total, g.total)
		}
		if g.mode == ModePlay && !g.slow.active {
			t.Errorf("%s: everybody of the target is out but the party did not end (mode %d)", name, g.mode)
		}
		t.Logf("%s: total %d (max %d)", name, g.total, mx)
	}
}

// The pieces: the reserve of the special killer and the arsonist, the start trim and the two players cap.
func TestKillTargetGuestRoom(t *testing.T) {
	snapshotGlobals(t)
	defs := testLevels(t)
	g := capGame(defs[0], 3)
	g.pickStartTarget()
	g.fillGuests(defs[0].Start)
	g.total = len(g.guests)
	// 2 places kept: the killer who has not come yet and the arsonist who has not died
	if r := g.goalReserve(); r != 2 {
		t.Fatalf("reserve %d, want 2", r)
	}
	g.total = goalCap - 2
	if g.guestRoom() != 0 {
		t.Fatalf("total %d of %d (2 reserved): room %d, want 0", g.total, goalCap, g.guestRoom())
	}
	before := len(g.guests)
	g.addGuestAt(g.lvl.RandomRoam(g.rng))
	g.powerFamous()
	g.startTheme(&Event{})
	if len(g.guests) != before || g.total != goalCap-2 {
		t.Fatalf("arrivals past the target: %d guests (was %d), total %d", len(g.guests), before, g.total)
	}
	g.total = goalCap - 5
	if g.guestRoom() != 3 {
		t.Fatalf("room %d, want 3", g.guestRoom())
	}
	// the hands of God create only the ones that fit and say so
	e := &Event{kind: evGod, phase: 1, n: 3}
	e.t = 3
	for k := 0; k < 200 && e.n > 0; k++ {
		g.stepEvent(e, 0.2)
	}
	if g.total != goalCap-2 {
		t.Fatalf("God: total %d, want %d", g.total, goalCap-2)
	}
	// the start: plain guests are dropped down to goalMax-goalStartRoom, the special ones stay
	g.total = 0
	for len(g.guests) < 80 {
		g.guests = append(g.guests, &Guest{pos: g.lvl.RandomRoam(g.rng)})
	}
	g.guests[0].keep, g.guests[1].name = true, "Famoso"
	k0, k1 := g.guests[0], g.guests[1]
	g.trimStartTotal()
	if len(g.guests)+len(g.guards) != goalCap-goalStartRoom {
		t.Fatalf("start trimmed to %d, want %d", len(g.guests)+len(g.guards), goalCap-goalStartRoom)
	}
	if g.guests[0] != k0 || g.guests[1] != k1 {
		t.Fatal("the start trim dropped a special guest")
	}
	// while a level loads (total 0) the start routines are not limited by the target
	if g.goalLeft() <= 0 {
		t.Fatal("goalLeft must not limit the start")
	}
}

func TestKillTargetTwoPlayers(t *testing.T) {
	old := twoPlayerMode
	twoPlayerMode = true
	t.Cleanup(func() { twoPlayerMode = old })
	defs := testLevels(t)
	g := capGame(defs[0], 5)
	g.pickStartTarget()
	if g.twoExtra < 20 || g.goalMax() != goalCap+g.twoExtra {
		t.Fatalf("two players: extra %d, cap %d", g.twoExtra, g.goalMax())
	}
	g.fillGuests(defs[0].Start)
	g.total = len(g.guests)
	for it := 0; it < 2000; it++ {
		g.addGuestAt(g.lvl.RandomRoam(g.rng))
		if it%3 == 0 {
			for _, gu := range g.guests {
				if !gu.out() {
					gu.dead = true
					break
				}
			}
		}
		if g.total > g.goalMax() {
			t.Fatalf("two players: total %d above %d", g.total, g.goalMax())
		}
	}
}
