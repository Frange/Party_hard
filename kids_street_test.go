package main

import "testing"

// ---- Diego, César and Sam-sam ride outside every other party (kids_street.go, 2026-11-11) ----

// The list of parties with room for the three outside (kidStreetFit) is what the built levels really have: open
// ground connected to the party, big enough (kidStreetArea).
func TestKidStreetFitMatchesTheLevels(t *testing.T) {
	for idx := range campaignLevels {
		g := peLoad(t, idx, false)
		comp, span := kidStreetArea(g.lvl)
		icon := campaignLevels[idx].icon
		fit := len(comp) > 0 && !kidStreetNo[icon]
		t.Logf("%2d %-9s open ground %4d cells, span %4.0f px -> fit %v", idx+1, icon, len(comp), span, fit)
		if icon == "skate" {
			continue // the bowls decide there
		}
		if fit != kidStreetFit[icon] {
			t.Errorf("level %d (%s): measured fit %v, kidStreetFit says %v", idx+1, icon, fit, kidStreetFit[icon])
		}
	}
}

// kidStreetWant: the campaign parties (1-based) where the three ride outside today (12, the skate park, has its
// bowls instead).
var kidStreetWant = map[int]bool{1: true, 3: true, 5: true, 7: true, 9: true, 11: true, 14: true, 16: true, 18: true,
	20: true, 22: true, 24: true, 26: true} // (v1.4.0: ALCALÁ DE HENARES is party 7 and GUADALAJARA 16: the ones after them moved)

var kidStreetDebug func(string, ...any)

// what the last kidStreetRun saw: the distance each kid rode and the tricks shouted
var (
	kidStreetTravel     map[*Guest]float64
	kidStreetTricksSeen int
)

func kidStreetAlive(g *Game) []*Guest {
	var out []*Guest
	for _, gu := range g.guests {
		if !gu.out() && isKid(gu) && gu.cs != nil && gu.cs.Spec != nil && isKidSlug(gu.cs.Spec.Name) {
			out = append(out, gu)
		}
	}
	return out
}

// kidStreetRun: s seconds of the party (guests, kids' arrivals) at 30 frames per second; it reports the longest time
// a riding kid stood still and the frames a kid spent off his open ground once inside it.
func kidStreetRun(g *Game, s float64) (stuck float64, outside int) {
	kidStreetTravel, kidStreetTricksSeen = map[*Guest]float64{}, 0
	trick := map[string]bool{}
	for _, v := range skaterTrickShout {
		trick[v[0]] = true
	}
	lastIcon := map[*Guest]string{}
	still := map[*Guest]float64{}
	last := map[*Guest]Vec{}
	inside := map[*Guest]bool{}
	for k := 0; k < int(s*30); k++ {
		peStep(g, 1.0/30, true)
		g.updateKidArrivals()
		g.updateElevators(1.0 / 30) // the roof: they come up in the lift
		g.updateVehicles(1.0 / 30)  // the cruise: in their little boat
		for _, gu := range kidStreetAlive(g) {
			if gu.state == gGoto && len(gu.path) > 0 && dist(gu.pos, last[gu]) < 0.5 {
				still[gu] += 1.0 / 30
				if still[gu] > stuck {
					stuck = still[gu]
				}
			} else {
				still[gu] = 0
			}
			if p, ok := last[gu]; ok {
				kidStreetTravel[gu] += dist(p, gu.pos)
			}
			if gu.icon != lastIcon[gu] && trick[gu.icon] {
				kidStreetTricksSeen++
			}
			lastIcon[gu] = gu.icon
			last[gu] = gu.pos
			if !g.kidStreetActive() {
				continue
			}
			cx, cy := g.lvl.cellOf(gu.pos)
			in := cx >= 0 && cy >= 0 && cx < g.lvl.cw && cy < g.lvl.ch && kidStreet.in[cy*g.lvl.cw+cx]
			if in {
				inside[gu] = true
			} else if inside[gu] && !g.lvl.Outdoor(gu.pos) {
				if outside%40 == 0 && kidStreetDebug != nil {
					end := Vec{}
					if len(gu.path) > 0 {
						end = gu.path[len(gu.path)-1]
					}
					ex, ey := g.lvl.cellOf(end)
					kidStreetDebug("off ground: %s at %.0f,%.0f state %d next %d path %d end %.0f,%.0f endIn %v pev %v scared %v", gu.cs.Spec.Name, gu.pos.X, gu.pos.Y, gu.state, gu.next, len(gu.path), end.X, end.Y, kidStreet.in[ey*g.lvl.cw+ex], g.pevTaken(gu), gu.state == gScared)
				}
				outside++
			}
		}
	}
	return
}

// With a real Game after loadLevel, in each of the 27 parties: the three come exactly where the rule says (3 kids,
// riding outside, never into a building, never stuck), nowhere else (not even dressed-up guests), and the caps hold.
func TestKidStreetExactlyEveryOtherParty(t *testing.T) {
	got := map[int]bool{}
	tricksAll := 0
	for idx := range campaignLevels {
		g := peLoad(t, idx, false)
		kidStreetDebug = t.Logf
		t.Cleanup(func() { kidStreetDebug = nil })
		n := idx + 1
		icon := campaignLevels[idx].icon
		if g.kidStreetActive() {
			got[n] = true
		}
		if icon == "skate" {
			if g.kidStreetActive() || len(g.lvl.bowls) == 0 {
				t.Errorf("skate park: street %v bowls %d", g.kidStreetActive(), len(g.lvl.bowls))
			}
			continue
		}
		planned := len(g.kidPlan)
		stuck, outside := kidStreetRun(g, 40)
		kids := kidStreetAlive(g)
		t.Logf("%2d %-9s street %-5v planned %d, kids now %d, still at most %.1f s, off-ground frames %d, total %d / goal %d",
			n, icon, g.kidStreetActive(), planned, len(kids), stuck, outside, g.total, g.goalMax())
		if kidStreetWant[n] {
			if planned > 3 || len(kids) != 3 {
				t.Errorf("party %d (%s): %d planned, %d kids riding, want 3", n, icon, planned, len(kids))
			}
			seen := map[string]bool{}
			for _, gu := range kids {
				seen[gu.cs.Spec.Name] = true
			}
			if len(seen) != len(kids) {
				t.Errorf("party %d: a kid twice", n)
			}
			for _, gu := range kids {
				if kidStreetTravel[gu] < 150 {
					t.Errorf("party %d: %s rode only %.0f px in 40 s", n, gu.cs.Spec.Name, kidStreetTravel[gu])
				}
			}
			tricksAll += kidStreetTricksSeen
		} else if planned != 0 || len(kids) != 0 {
			t.Errorf("party %d (%s): %d planned, %d kids, want none", n, icon, planned, len(kids))
		}
		if stuck > kidStreetStuckT+1.5 {
			t.Errorf("party %d: a kid stood still %.1f s while riding", n, stuck)
		}
		if outside > 0 {
			t.Errorf("party %d: %d frames of a kid inside a building", n, outside)
		}
		if g.total > g.goalMax() || g.aliveCount() > g.goalMax() {
			t.Errorf("party %d: caps broken: total %d alive %d goal %d", n, g.total, g.aliveCount(), g.goalMax())
		}
	}
	t.Logf("tricks shouted on the flat in all the parties: %d", tricksAll)
	if tricksAll < 12 {
		t.Errorf("only %d tricks on the flat in %d parties", tricksAll, len(kidStreetWant))
	}
	for n := 1; n <= len(campaignLevels); n++ {
		if got[n] != kidStreetWant[n] {
			t.Errorf("party %d: street kids %v, want %v", n, got[n], kidStreetWant[n])
		}
	}
	// every other party: never two parties of the rule in a row, never three without (the skate park apart)
	gap := 0
	for n := 1; n <= len(campaignLevels); n++ {
		if got[n] {
			gap = 0
			if got[n+1] {
				t.Errorf("parties %d and %d both have them", n, n+1)
			}
			continue
		}
		if gap++; gap > 2 {
			t.Errorf("three parties in a row without them, up to %d", n)
		}
	}
}

// Two players: the kids come all the same, the 2P caps hold; a dead kid never comes back.
func TestKidStreetTwoPlayersCapsAndNoRespawn(t *testing.T) {
	for _, idx := range []int{0, 2, 10} {
		g := peLoad(t, idx, true)
		kidStreetRun(g, 20)
		kids := kidStreetAlive(g)
		if len(kids) != 3 {
			t.Fatalf("2P party %d: %d kids", idx+1, len(kids))
		}
		if g.total > g.goalMax() || g.aliveCount() > g.goalMax() {
			t.Errorf("2P party %d: total %d alive %d goal %d", idx+1, g.total, g.aliveCount(), g.goalMax())
		}
		victim := kids[0].cs.Spec.Name
		g.killGuest(kids[0], true)
		kidStreetRun(g, 40)
		if g.kidPresent(victim) {
			t.Errorf("2P party %d: %s came back after dying", idx+1, victim)
		}
		// the other two ride on; one of them may die too meanwhile (a star of the party smashing the floor next to
		// him, one run in a few): what counts is that nobody new comes in his place
		left := map[*Guest]bool{kids[1]: true, kids[2]: true}
		now := kidStreetAlive(g)
		if len(now) > 2 {
			t.Errorf("2P party %d: %d kids after one died", idx+1, len(now))
		}
		for _, gu := range now {
			if !left[gu] {
				t.Errorf("2P party %d: a new kid (%s) came after one died", idx+1, gu.cs.Spec.Name)
			}
		}
	}
}

// Random mode: the same rule by title. A random roll of a campaign venue answers like its campaign party; the
// space station, the ship and the island never.
func TestKidStreetRandomModeSameRule(t *testing.T) {
	g := partyGame(t)
	_ = g
	camp := kidStreetCampaign()
	n := 0
	for i := campaignN; i < len(levelDefs); i++ {
		d := levelDefs[i]
		for c, e := range campaignLevels {
			if e.title != "" && e.title == d.Title {
				n++
				if got := kidStreetOnFor(i, d, nil); got != camp[c] {
					t.Errorf("random %q: %v, campaign party %d says %v", d.Title, got, c+1, camp[c])
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no random roll of a campaign venue")
	}
	if kidStreetOnFor(specialJurassic, &LevelDef{Title: "X"}, nil) || kidStreetOnFor(5000, &LevelDef{Icon: "station", Title: "Y"}, nil) {
		t.Error("street kids on the island or the station")
	}
}
