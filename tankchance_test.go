package main

import (
	"math/rand"
	"testing"
)

func tankRestore(t *testing.T) {
	t.Helper()
	sd, rg, ms, ls := tankSeed, tankRng, tankMisses, tankLast
	t.Cleanup(func() { tankSeed, tankRng, tankMisses, tankLast = sd, rg, ms, ls })
}

// Over many parties: about 1 in 5, never two in a row, never more than ~10 parties without one.
func TestTankChancePatternAndPity(t *testing.T) {
	tankRestore(t)
	for seed := int64(1); seed <= 20; seed++ {
		r := rand.New(rand.NewSource(seed))
		misses, last := 0, false
		n, yes, run, maxRun, prev := 20000, 0, 0, 0, false
		for i := 0; i < n; i++ {
			y := tankDecide(r, &misses, &last)
			if y {
				yes++
				if prev {
					t.Fatalf("seed %d: two tanks in a row", seed)
				}
				run = 0
			} else if run++; run > maxRun {
				maxRun = run
			}
			prev = y
		}
		if f := float64(yes) / float64(n); f < 0.17 || f > 0.23 {
			t.Errorf("seed %d: tank in %.3f of the parties, want about 0.20", seed, f)
		}
		if maxRun > 10 {
			t.Errorf("seed %d: %d parties without a tank", seed, maxRun)
		}
	}
	// the pity on its own: a dice that never hits
	misses, last := 0, false
	misses = tankPityAfter
	if !tankDecide(rand.New(rand.NewSource(5)), &misses, &last) {
		t.Error("after 9 parties without a tank the next one must get it")
	}
}

// A party with a tank due sends exactly ONE tank down the road, however long it lasts; one without, none.
func TestTankAtMostOnePerParty(t *testing.T) {
	for _, due := range []bool{false, true} {
		g := hudPlay(t, false)
		prevPol, prevTr := g.lvl.Def.Police, g.lvl.Def.Transit
		t.Cleanup(func() { g.lvl.Def.Police, g.lvl.Def.Transit = prevPol, prevTr })
		g.lvl.Def.Police = "car"
		g.lvl.Def.Transit = ""
		g.tankDue, g.tankN, g.amb = due, 0, nil
		seen := 0
		for i := 0; i < 400; i++ {
			g.ambT = 0
			g.amb = nil
			g.updateAmbient(0.01)
			for _, a := range g.amb {
				if a.kind == ambTank {
					seen++
				}
			}
		}
		if due && seen != 1 {
			t.Errorf("tank due: %d tanks, want exactly 1", seen)
		}
		if !due && seen != 0 {
			t.Errorf("no tank due: %d tanks", seen)
		}
	}
}
