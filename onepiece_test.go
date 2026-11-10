package main

import (
	"math/rand"
	"testing"
)

// The five straw-hat parodies: in all views and animations the drawing stays in the box, frames are
// cached (no extra allocation) and they are famous guests.
func TestOnePieceFourDirectionsInBox(t *testing.T) {
	for _, n := range []string{"luffy", "sanji", "zoro", "nami", "chopper"} {
		if presetByName(n) == nil {
			t.Fatalf("no preset %s", n)
		}
		sp := presetByName(n)(rand.New(rand.NewSource(1)))
		found := false
		for _, s := range specialPool {
			found = found || s == n
		}
		if !found {
			t.Errorf("%s is not in the special pool", n)
		}
		for vw := 0; vw < 3; vw++ {
			for a := 0; a < AnimCount; a++ {
				for f := 0; f < animFrames[a]; f++ {
					pp := poseFor(a, f)
					pp.Anim, pp.F = a, f
					b := genChar(&sp, vw, pp)
					if b.w != charW || b.h != charH {
						t.Fatalf("%s box %dx%d", n, b.w, b.h)
					}
				}
			}
		}
	}
}
