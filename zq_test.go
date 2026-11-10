package main

import (
	"math/rand"
	"testing"
)

func TestZQ(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	n, z := 20000, 0
	for i := 0; i < n; i++ {
		L := 300 + r.Float64()*180
		t0 := 45 + r.Float64()*45
		big, sat, god := 0, false, false
		has := false
		for tt := t0; tt < L; tt += rareGap(r, tt) {
			k := rareKind(r, tt, big, sat, god, func(int) bool { return false })
			sat = sat || k == evSatan
			god = god || k == evGod
			if rareBigKinds[k] {
				big++
			}
			if k == evZombie {
				has = true
			}
		}
		if has {
			z++
		}
	}
	t.Logf("ZOMBIE %.1f%%", 100*float64(z)/float64(n))
}
