package main

import "testing"

// The Eiffel Tower: only the legs are solid, the arch between them can be walked under.
func checkEiffelArch(t *testing.T, label string) {
	t.Helper()
	l := buildLevel(genPlaza(newRand(3), true))
	var tw *Prop
	for _, p := range l.Props {
		if p.Name == "eiffel" {
			tw = p
		}
	}
	if tw == nil {
		t.Fatal("no eiffel in Paris")
	}
	y := tw.Y - 7
	if l.SolidP(tw.X, y) || l.SolidP(tw.X-20, y) || l.SolidP(tw.X+20, y) || l.SolidP(tw.X, tw.Y-1) {
		t.Errorf("%s: the arch is blocked", label)
	}
	if !l.SolidP(tw.X-50, y) || !l.SolidP(tw.X+50, y) {
		t.Errorf("%s: the legs are not solid", label)
	}
	for k, ps := range l.POIs {
		for _, p := range ps {
			if p.X > tw.X-60 && p.X < tw.X+60 && p.Y > tw.Y-16 && p.Y < tw.Y+2 && (p.X < tw.X-40 || p.X > tw.X+40) {
				t.Errorf("%s: POI %s inside a leg at %v", label, k, p)
			}
		}
	}
}

func TestEiffelArchWalkable(t *testing.T) { checkEiffelArch(t, "code") }

func TestEiffelArchWalkableExported(t *testing.T) {
	// only the exported eiffel (read from the real assets, never written), restored afterwards
	snapshotPropKinds(t)
	oa, err := loadObjAsset(spriteDirOf("assets", "objetos", "eiffel"))
	if err != nil {
		t.Skip("eiffel is not exported:", err)
	}
	applyObjects([]*ObjAsset{oa})
	if k := propKinds["eiffel"]; k == nil || len(k.Native) == 0 {
		t.Fatal("setup: eiffel should come from its PNG")
	}
	checkEiffelArch(t, "exported")
}
