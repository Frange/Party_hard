package main

import (
	"math/rand"
	"testing"
)

// The "invisible wall" bug: the player got stuck in open ground. Two causes
// were found and are guarded here: spots handed out by NearestFree where the
// player's collision box did not fit (he ended up inside a prop or a wall), and
// solid footprints that reached below the drawing of their prop.

func testLevels(t *testing.T) []*LevelDef {
	t.Helper()
	snapshotPropKinds(t)
	prevN := campaignN // builtinLevels below sets it: give it back (flowGame trusts it when levelDefs is already long)
	t.Cleanup(func() { campaignN = prevN })
	if st := loadAssetStore(); st != nil {
		applyObjects(st.Objs)
	}
	fitAllFootprints()
	defs := builtinLevels()
	for _, kind := range genKinds {
		for sd := int64(1); sd < 3; sd++ {
			if d := genNamed(kind, sd*131, ""); d != nil {
				defs = append(defs, d)
			}
		}
	}
	return defs
}

func TestNearestFreeIsWalkable(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for _, def := range testLevels(t) {
		l := buildLevel(def)
		bad := 0
		for i := 0; i < 20000; i++ {
			q := l.NearestFree(Vec{rng.Float64() * l.W, rng.Float64() * l.H})
			if l.SolidP(q.X, q.Y) {
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("%s: NearestFree returned a blocked spot %d times", def.Title, bad)
		}
	}
}

func TestFootInsideOpaqueBox(t *testing.T) {
	testLevels(t)
	for name, k := range propKinds {
		if k.NoSolid || k.Ambient > 0 || k.FW <= 0 {
			continue
		}
		var pb *pixBuf
		var oy float64
		if k.Native != nil {
			pb, oy = k.Native[0], k.NAY
		} else {
			pb, oy = stylizeSprite(renderPropSprite(name, k, 0)), (float64(k.H)+1)*wk
		}
		bottom := -1
		for y := pb.h - 1; y >= 0 && bottom < 0; y-- {
			for x := 0; x < pb.w; x++ {
				if pb.get(x, y).A > 40 {
					bottom = y
					break
				}
			}
		}
		fy := -k.FH
		if k.hasFoot {
			fy = k.FootDY
		}
		if excess := oy + (fy+k.FH)*wk - float64(bottom+1); excess >= 3.5 {
			t.Errorf("%s: its solid footprint reaches %.0f px below the drawing", name, excess)
		}
	}
}

func TestStartIsWalkable(t *testing.T) {
	for _, def := range testLevels(t) {
		l := buildLevel(def)
		if l.SolidP(def.Start.X, def.Start.Y) {
			t.Errorf("%s: the player starts inside something solid at %.0f,%.0f", def.Title, def.Start.X, def.Start.Y)
		}
	}
}

// nothing that can be picked up may leave a solid behind
func TestPickupItemsAreNotSolid(t *testing.T) {
	for _, def := range testLevels(t) {
		l := buildLevel(def)
		for _, p := range l.Props {
			if p.K.Item == "" {
				continue
			}
			for _, r := range l.solidsP {
				if r == p.Foot {
					t.Errorf("%s: %s at %.0f,%.0f is solid", def.Title, p.Name, p.X, p.Y)
				}
			}
		}
	}
}
