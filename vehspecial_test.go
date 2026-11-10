package main

import (
	"math/rand"
	"testing"
)

// the tank, the tow truck and the horse have a front and a rear view that differ, and the free horse too
func TestSpecialVehicleViews(t *testing.T) {
	for _, n := range []string{"tank", "tow", "horse"} {
		d := vehViewDefs[n]
		if d == nil {
			t.Fatalf("%s: no views", n)
		}
		f, b := newPix(d.w, d.h), newPix(d.w, d.h)
		d.draw(f, 0, true)
		d.draw(b, 0, false)
		nf, nb, diff := 0, 0, 0
		for i := 0; i < len(f.p); i += 4 {
			if f.p[i+3] > 0 {
				nf++
			}
			if b.p[i+3] > 0 {
				nb++
			}
			if f.p[i] != b.p[i] || f.p[i+1] != b.p[i+1] || f.p[i+2] != b.p[i+2] || f.p[i+3] != b.p[i+3] {
				diff++
			}
		}
		if nf < 150 || nb < 150 || diff < 40 {
			t.Errorf("%s: front/rear views are empty or equal (%d, %d opaque, %d different)", n, nf, nb, diff)
		}
	}
	for coat := 0; coat < 3; coat++ {
		if horseFreeView(coat, true).img == nil || horseFreeView(coat, false).img == nil {
			t.Errorf("free horse, coat %d: no views", coat)
		}
	}
	if towSide().img == nil {
		t.Error("no side view of the tow truck")
	}
}

// the specials do come up as traffic, tank at most twice a party
func TestRoadSpecialPick(t *testing.T) {
	testLevels(t)
	g := &Game{rng: rand.New(rand.NewSource(11))}
	seen := map[string]int{}
	for i := 0; i < 4000; i++ {
		seen[g.roadSpecialPick("car")]++
	}
	for _, n := range []string{"horse", "tow", "tank", "car"} {
		if seen[n] == 0 {
			t.Errorf("%s never comes up", n)
		}
	}
	if seen["tank"] > 2 {
		t.Errorf("%d tanks in one party", seen["tank"])
	}
}
