package main

import (
	"math/rand"
	"testing"
)

func dmgGame(t *testing.T) *Game {
	t.Helper()
	oldOn, oldSt := damageOn, dmg
	t.Cleanup(func() { damageOn, dmg = oldOn, oldSt })
	damageOn, dmg = true, dmgState{}
	g := suspectGame()
	g.time = 1
	g.updateDamage()
	return g
}

func dmgInk(x0, y0, x1, y1 int) (n int) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if dmg.buf.RGBAAt(x, y).A > 0 {
				n++
			}
		}
	}
	return n
}

func TestBlastLeavesLastingMarksAndNothingElseChanges(t *testing.T) {
	g := dmgGame(t)
	nProps, nWalls := len(g.lvl.Props), len(g.lvl.Def.Walls)
	if dmgInk(0, 0, dmg.w, dmg.h) != 0 {
		t.Fatal("a fresh party must have a clean ground")
	}
	x, y := 300.0, 260.0
	g.decals = append(g.decals, Decal{pos: Vec{x, y}, kind: 1, r: 22})
	g.updateDamage()
	cx, cy := int(x*wk), int(y*wk)
	if n := dmgInk(cx-40, cy-30, cx+40, cy+30); n < 80 {
		t.Fatalf("the blast left only %d marked pixels", n)
	}
	marks := dmg.n
	if !dmg.dirty || marks < 1 {
		t.Fatalf("a mark expected, got %d (dirty %v)", dmg.n, dmg.dirty)
	}
	g.time += 300 // marks do not fade
	g.updateDamage()
	if dmgInk(cx-40, cy-30, cx+40, cy+30) < 80 {
		t.Fatal("the marks must last until the end of the party")
	}
	if len(g.lvl.Props) != nProps || len(g.lvl.Def.Walls) != nWalls {
		t.Fatal("damage must not add or remove props or collisions")
	}
	g.updateDamage() // already seen: nothing new
	if dmg.n != marks {
		t.Fatalf("a decal was stamped twice (%d -> %d)", marks, dmg.n)
	}
}

func TestDamageIsSubtle(t *testing.T) {
	g := dmgGame(t)
	g.decals = append(g.decals, Decal{pos: Vec{300, 260}, kind: 1, r: 22})
	g.updateDamage()
	marked, strong := 0, 0
	for i := 3; i < len(dmg.buf.Pix); i += 4 {
		if a := dmg.buf.Pix[i]; a > 0 {
			marked++
			if a > 150 {
				strong++
			}
		}
	}
	if marked < 100 || strong*10 > marked {
		t.Fatalf("%d marked pixels, %d strong: must be visible but subtle (under 10%% strong)", marked, strong)
	}
}

func TestSmallFireMakesSootAndBloodMakesNothing(t *testing.T) {
	g := dmgGame(t)
	g.decals = append(g.decals, Decal{pos: Vec{200, 200}, kind: 0, r: 12}, Decal{pos: Vec{400, 300}, kind: 1, r: 8}, Decal{pos: Vec{500, 300}, kind: 1, r: 3})
	g.updateDamage()
	if dmg.n != 1 {
		t.Fatalf("only the fire spot counts, got %d marks", dmg.n)
	}
}

func TestDamageCapAndResetOnNewLevel(t *testing.T) {
	g := dmgGame(t)
	for i := 0; i < dmgMax+40; i++ {
		g.decals = append(g.decals, Decal{pos: Vec{float64(60 + i*5%800), 200}, kind: 1, r: 16})
	}
	g.updateDamage()
	if dmg.n != dmgMax {
		t.Fatalf("marks %d, cap %d", dmg.n, dmgMax)
	}
	g.lvl = buildLevel(level1()) // a new party
	g.decals = nil
	g.updateDamage()
	if dmg.n != 0 || dmgInk(0, 0, dmg.w, dmg.h) != 0 {
		t.Fatal("a new party starts with a clean ground")
	}
}

func TestBlastCracksTheBaseOfNearbyProps(t *testing.T) {
	g := dmgGame(t)
	var pr *Prop
	for _, p := range g.lvl.Props {
		if p.K != nil && !p.K.NoSolid && p.K.Trap == "" && p.Foot.W > 8 {
			pr = p
			break
		}
	}
	if pr == nil {
		t.Fatal("level1 has no solid prop")
	}
	g.decals = append(g.decals, Decal{pos: Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y + pr.Foot.H + 6}, kind: 1, r: 18})
	g.updateDamage()
	if !dmg.hurt[pr] || dmg.n < 2 {
		t.Fatalf("the prop next to the blast should have cracks (marks %d)", dmg.n)
	}
}

func TestDamageDoesNotMoveGameRngNorAllocateWhenIdle(t *testing.T) {
	g := dmgGame(t)
	g.rng = rand.New(rand.NewSource(77))
	want := rand.New(rand.NewSource(77))
	g.decals = append(g.decals, Decal{pos: Vec{300, 260}, kind: 1, r: 22})
	g.updateDamage()
	g.scar(scarScuff, 100, 100, 6)
	if g.rng.Int63() != want.Int63() {
		t.Fatal("damage used g.rng")
	}
	if a := testing.AllocsPerRun(200, func() { g.time += 0.016; g.updateDamage() }); a != 0 {
		t.Fatalf("%v allocations per idle update", a)
	}
}
