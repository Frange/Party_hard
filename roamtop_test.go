package main

import (
	"math/rand"
	"testing"
)

// The guests do not wander to (nor appear at) the very top of the screen, where their heads are cut by the border:
// every place picked by RandomRoam has its feet at roamTopY or lower, in every party, and each party keeps most of its
// roaming area (the cut is a thin band, nothing is closed).
func TestGuestsKeepAwayFromTheTopBorder(t *testing.T) {
	if roamTopY < 44 || roamTopY > 90 {
		t.Fatalf("roamTopY %.0f: a person is about 40 px tall; the band must show him whole and stay thin", roamTopY)
	}
	n := killTargetLevels(t)
	for i := 0; i < n; i++ {
		g := playGame(t, i, int64(41+i))
		l := g.lvl
		all, kept := 0.0, 0.0
		for _, r := range l.Def.Roam {
			all += r.W * r.H
		}
		for _, r := range l.roamRects() {
			kept += r.W * r.H
		}
		if kept < all*0.8 {
			t.Errorf("L%d (%s): only %.0f%% of the roaming area is left", i+1, campaignLevels[i].icon, kept/all*100)
		}
		rng := rand.New(rand.NewSource(int64(i)))
		low := 0
		for k := 0; k < 400; k++ {
			p := l.RandomRoam(rng)
			if p.Y < roamTopY {
				low++
			}
			if !l.Free(p) {
				t.Fatalf("L%d: RandomRoam gave a blocked place %v", i+1, p)
			}
		}
		if low > 0 {
			t.Errorf("L%d (%s): %d of 400 places to wander to are in the top band (y < %.0f)", i+1, campaignLevels[i].icon, low, roamTopY)
		}
	}
}
