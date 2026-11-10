package main

import (
	"fmt"
	"testing"
)

// Cheap quality sweep over many seeds of every generated kind (no window): overlaps, start spot,
// reachability of every point of interest, guest cap, a DJ where the party needs one.
func TestGenQualitySweep(t *testing.T) {
	snapshotPropKinds(t)
	if st := loadAssetStore(); st != nil {
		applyObjects(st.Objs)
	}
	fitAllFootprints()
	n := 0
	for _, k := range ovlKinds {
		for s := int64(1); s <= djGenSeeds; s++ {
			d := genNamed(k, s*131, "")
			if d == nil {
				t.Errorf("%s-s%d: no valid level after 40 tries", k, s)
				continue
			}
			n++
			name := fmt.Sprintf("%s-s%d", k, s)
			l := buildLevel(d)
			for _, is := range levelOverlaps(l) {
				t.Errorf("%s: overlap %s", name, is)
			}
			if l.SolidP(d.Start.X, d.Start.Y) {
				t.Errorf("%s: start (%.0f,%.0f) is blocked", name, d.Start.X, d.Start.Y)
			}
			if b := genBad(d); len(b) > 0 {
				t.Errorf("%s: %d unreachable points, first (%.0f,%.0f)", name, len(b), b[0].X, b[0].Y)
			}
			if d.Guests > 60 {
				t.Errorf("%s: guests %d above the cap of 60 (the game trims to 45-60 and fills small levels)", name, d.Guests)
			}
		}
	}
	if n < 400 {
		t.Fatalf("only %d levels checked", n)
	}
}
