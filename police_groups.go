package main

import "fmt"

// Police groups (2026-11-09).
//
// The groups the player has unlocked (kills milestones in trophies.go: local police
// always, FBI 500, military 1000, SWAT 1500, Power Rangers 2000) share the party out
// in equal stretches: with N groups and T guests, the k-th death of the party (the
// player's or not, 2026-11-10) is handled by group (k-1)*N/T (capped at N-1). 60 guests and police+FBI: kills 1-30
// police, 31-60 FBI; with 3 groups 20 each, with 4 groups 15, with 5 groups 12.
//
// A call ALWAYS brings the local police first (1 officer for one body found, 2 for
// more). Once they are on scene the group of the current stretch is sent in, and
// when it gets out of its vehicle the lower groups leave without questions
// (changeResponse in game.go).

// groupStat: how hard each group searches. Small steps, all in one place.
var groupStat = [...]struct {
	crew  int     // officers per vehicle (the police: 1 or 2 by the bodies found)
	sight float64 // how far they spot the player (x copRange)
	pace  float64 // how fast they walk and sweep (x copWalkSpeed / copSweepSpeed)
}{
	unitPolice:   {1, 1.00, 1.00},
	unitFBI:      {2, 1.05, 1.04},
	unitMilitary: {3, 1.10, 1.08},
	unitSWAT:     {4, 1.15, 1.12},
	unitRangers:  {5, 1.20, 1.16},
}

// dbgPoliceGroups: -unlocked N (debug) plays as if the first N groups were unlocked.
// It never touches the progress (debugFlagsActive freezes it anyway).
var dbgPoliceGroups int

func groupOK(kind int) bool { return kind >= 0 && kind < len(groupStat) }

func groupSight(kind int) float64 {
	if !groupOK(kind) {
		return 1
	}
	return groupStat[kind].sight
}

func groupPace(kind int) float64 {
	if !groupOK(kind) {
		return 1
	}
	return groupStat[kind].pace
}

// policeGroupFor: which stretch (0 = police, 1 = the next group...) the k-th kill of
// a party of T guests falls in, with N groups unlocked.
func policeGroupFor(k, total, n int) int {
	if n <= 1 || total <= 0 || k <= 1 {
		return 0
	}
	i := (k - 1) * n / total
	if i > n-1 {
		i = n - 1
	}
	return i
}

// policeGroups: the unit of every group the player can get, in order (the local police first).
func policeGroups() []int {
	if dbgPoliceGroups > 0 {
		n := minI(dbgPoliceGroups, len(groupStat))
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	out := []int{unitPolice}
	for u := unitFBI; u < len(groupStat); u++ {
		if ownedID(fmt.Sprintf("police:%d", u)) {
			out = append(out, u)
		}
	}
	return out
}

// groupUnitNow: the group that should handle the case right now.
func (g *Game) groupUnitNow() int {
	gs := policeGroups()
	if g.isIsland() && dbgRexKill { // -trexkill: the Power Rangers come for the T-Rex at once
		return unitRangers
	}
	// every death of the party counts, not only the player's (2026-11-10)
	k := maxI(g.kills, g.partyDeaths())
	return gs[policeGroupFor(k, g.total, len(gs))]
}

// groupCrew: how many officers a vehicle of this group brings.
func (g *Game) groupCrew(unit int) int {
	if unit != unitPolice && groupOK(unit) {
		return groupStat[unit].crew
	}
	found := 0
	for _, b := range g.bodies {
		if b.reported && !b.carried && !b.bagged {
			found++
		}
	}
	if found >= 2 {
		return 2
	}
	return 1
}

// updatePoliceGroup sends in the group of the current stretch once somebody is
// actually working the case (so the local police always arrive first, and nothing
// is sent after the case has been closed). Groups only go up during a call.
func (g *Game) updatePoliceGroup() {
	want := g.groupUnitNow()
	if want <= g.respKind || want == g.pendingUnit || (g.pendingUnit >= 0 && g.pendingUnit > want) {
		return
	}
	working := false
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave {
			working = true
			break
		}
	}
	if !working {
		return
	}
	g.escalateTo(want)
	g.announceUnit(want)
}
