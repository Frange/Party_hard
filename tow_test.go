package main

import (
	"math"
	"math/rand"
	"testing"
)

// the tow truck used to be dead code (nothing called maybeTow). A wreck in either lane must bring a
// tow truck along that very lane, which picks the wreck up and takes it away.
func TestTowTruckCollectsWrecks(t *testing.T) {
	ran := 0
	defer func() {
		if ran == 0 {
			t.Fatal("no level was tested")
		}
	}()
	for _, mk := range []func() *LevelDef{level1, level3, level4} {
		def := mk()
		if def.Police != "car" {
			continue
		}
		for _, right := range []bool{true, false} {
			ran++
			g := &Game{rng: rand.New(rand.NewSource(3)), lvl: buildLevel(def)}
			g.ambT = 1e9
			y := def.CarFrom.Y + g.laneOff()
			if !right {
				y = def.CarFrom.Y - g.laneOff()
			}
			g.wrecks = []Vec{{400, y}}
			picked, gone := false, false
			for i := 0; i < 30*80 && !gone; i++ {
				g.updateAmbient(1.0 / 30)
				for _, a := range g.amb {
					if a.kind == ambTow {
						if math.Abs(a.y-y) > 0.01 {
							t.Fatalf("%s: tow truck in lane %.0f, wreck in lane %.0f", def.Title, a.y, y)
						}
						picked = picked || a.loaded
					}
				}
				gone = picked && len(g.amb) == 0
			}
			if len(g.wrecks) != 0 || !picked || !gone {
				t.Errorf("%s (right=%v): wrecks left %d, picked %v, gone %v", def.Title, right, len(g.wrecks), picked, gone)
			}
		}
	}
}
