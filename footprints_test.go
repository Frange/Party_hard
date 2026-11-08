package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func fpSetup(t *testing.T, two bool, title string) *Game {
	g := layerGame(t, two)
	on, mode := footprintsOn, g.mode
	t.Cleanup(func() { footprintsOn, g.mode = on, mode; fp = fpState{} })
	footprintsOn = true
	g.loadLevel(ambientLevelIdx(title))
	g.mode = ModePlay
	return g
}

// fpRow: a horizontal run of ground of the style, at least 200 px long.
func fpRow(t *testing.T, g *Game, st fpStyle) (x0, y float64) {
	need := 50
	if st == fpWet {
		need = 20 // the band around a pool is short
	}
	for yy := 20.0; yy < screenH-20; yy += 4 {
		run := 0
		for xx := 10.0; xx < screenW-10; xx += 4 {
			if fpGround(g.lvl, st, Vec{xx, yy}) {
				run++
				if run >= need {
					return xx - float64(need-1)*4, yy
				}
			} else {
				run = 0
			}
		}
	}
	t.Fatalf("%s: no ground for the prints", g.lvl.Def.Title)
	return
}

func fpWalk(g *Game, x0, y float64, frames int) {
	for i := 0; i < frames; i++ {
		g.p.pos = Vec{x0 + float64(i)*1.5, y}
		g.updateFootprints(1.0 / 60)
	}
}

func TestFootprintsOnlyInPlanLevelsAndWhenMoving(t *testing.T) {
	g := fpSetup(t, false, "MIAMI BEACH")
	for _, e := range footprintPlan {
		idx := ambientLevelIdx(e.Title)
		if idx < 0 {
			if e.Title == "PLAYA" { // generated for the random mode only
				continue
			}
			t.Fatalf("plan level %s does not exist", e.Title)
		}
		g.loadLevel(idx)
		g.mode = ModePlay
		g.guests, g.guards, g.cops = nil, nil, nil
		x0, y := fpRow(t, g, e.Style)
		g.updateFootprints(1.0 / 60) // follows the player
		for i := 0; i < 60; i++ {    // standing still: nothing
			g.p.pos = Vec{x0, y}
			g.updateFootprints(1.0 / 60)
		}
		if fp.n != 0 {
			t.Fatalf("%s: standing still left %d prints", e.Title, fp.n)
		}
		frames := 130
		if e.Style == fpWet {
			frames = 52 // 76 px at 1.5 px per frame
		}
		fpWalk(g, x0, y, frames)
		if fp.n < 3 {
			t.Errorf("%s: walking left only %d prints", e.Title, fp.n)
		}
		if fp.ring[0].life < 6 || fp.ring[0].life > 10 {
			t.Errorf("%s: life %.1f outside 6-10 s", e.Title, fp.ring[0].life)
		}
	}
	// a level without plan: nothing, even walking
	g.loadLevel(ambientLevelIdx("DISCOTECA"))
	g.mode = ModePlay
	fpWalk(g, 100, 200, 100)
	if fp.n != 0 {
		t.Fatalf("a level without plan left %d prints", fp.n)
	}
	// walking on the wrong ground of a plan level (the sea side / pavement): none
	g.loadLevel(ambientLevelIdx("ESTACIÓN DE ESQUÍ"))
	g.mode = ModePlay
	g.guests, g.guards, g.cops = nil, nil, nil
	for yy := 20.0; yy < screenH; yy += 8 {
		if !fpGround(g.lvl, fpSnow, Vec{20, yy}) {
			fpWalk(g, 20, yy, 40)
			if fp.n != 0 {
				t.Fatalf("prints on non-snow ground: %d", fp.n)
			}
			break
		}
	}
}

func TestFootprintsAlternateAndPointAlongTheWalk(t *testing.T) {
	g := fpSetup(t, false, "MIAMI BEACH")
	g.guests, g.guards, g.cops = nil, nil, nil
	x0, y := fpRow(t, g, fpSand)
	g.updateFootprints(0.016)
	fpWalk(g, x0, y, 130)
	for i := 1; i < fp.n; i++ {
		a, b := fp.ring[i-1], fp.ring[i]
		if !a.horiz || !b.horiz {
			t.Fatalf("walking right: the prints must point along x")
		}
		if a.y == b.y {
			t.Fatalf("print %d: left and right foot must alternate (same y %.0f)", i, a.y)
		}
		if gap := b.x - a.x; gap < 9.5 || gap > 15.5 {
			t.Fatalf("print %d: gap %.1f px, want 10-14", i, gap)
		}
	}
	if !fp.ring[0].strong {
		t.Errorf("the player's prints must be the strong ones")
	}
}

func TestFootprintsBufferCapAndFade(t *testing.T) {
	g := fpSetup(t, false, "MIAMI BEACH")
	g.guests, g.guards, g.cops = nil, nil, nil
	x0, y := fpRow(t, g, fpSand)
	g.updateFootprints(0.016)
	for lap := 0; lap < 10; lap++ {
		fpWalk(g, x0, y, 130)
		fpWalk(g, x0+200, y, 1) // teleport back: no print for the jump
	}
	if fp.n != fpMax {
		t.Fatalf("buffer holds %d prints, want exactly the cap %d", fp.n, fpMax)
	}
	// fade: alive now, gone after 10 s
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	g.drawFootprints(scr)
	if fp.drawn == 0 {
		t.Fatal("fresh prints are not drawn")
	}
	for i := 0; i < 5; i++ {
		fp.t += 2.1
	}
	g.drawFootprints(scr)
	if fp.drawn != 0 {
		t.Fatalf("after 10.5 s still %d prints drawn", fp.drawn)
	}
}

func TestFootprintsClearedOnNewLevelAndSwitch(t *testing.T) {
	g := fpSetup(t, false, "MIAMI BEACH")
	g.guests, g.guards, g.cops = nil, nil, nil
	x0, y := fpRow(t, g, fpSand)
	g.updateFootprints(0.016)
	fpWalk(g, x0, y, 100)
	if fp.n == 0 {
		t.Fatal("no prints to clear")
	}
	g.loadLevel(ambientLevelIdx("ESTACIÓN DE ESQUÍ"))
	g.mode = ModePlay
	g.updateFootprints(0.016)
	if fp.n != 0 || fp.nt > 1+len(g.guests)+len(g.guards)+len(g.cops) {
		t.Fatalf("level change must clear: n=%d tracks=%d", fp.n, fp.nt)
	}
	footprintsOn = false
	x0, y = fpRow(t, g, fpSnow)
	fpWalk(g, x0, y, 130)
	if fp.n != 0 {
		t.Fatalf("footprintsOn=false still leaves %d prints", fp.n)
	}
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	g.drawFootprints(scr)
	if fp.drawn != 0 {
		t.Fatal("footprintsOn=false still draws")
	}
}

func TestFootprintsPauseDoesNotAge(t *testing.T) {
	g := fpSetup(t, false, "MIAMI BEACH")
	g.guests, g.guards, g.cops = nil, nil, nil
	x0, y := fpRow(t, g, fpSand)
	g.updateFootprints(0.016)
	fpWalk(g, x0, y, 60)
	before, t0 := fp.n, fp.t
	// the help window pauses: Update returns before updatePlay, so the clock stays
	g.helpOv = true
	t.Cleanup(func() { g.helpOv = false })
	for i := 0; i < 600; i++ {
		_ = g.Update()
	}
	if fp.t != t0 || fp.n != before {
		t.Fatalf("paused game moved the prints: t %.2f->%.2f n %d->%d", t0, fp.t, before, fp.n)
	}
	// and a mode that is not play does nothing either
	g.mode = ModeMenu
	g.updateFootprints(5)
	if fp.t != t0 {
		t.Fatal("clock advanced outside ModePlay")
	}
}

func TestFootprints2PAndGuestsNoPanic(t *testing.T) {
	g := fpSetup(t, true, "ESTACIÓN DE ESQUÍ")
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	x0, y := fpRow(t, g, fpSnow)
	for i := 0; i < 400; i++ {
		g.p.pos = Vec{x0 + float64(i%130)*1.5, y}
		g.p2.pos = Vec{x0 + float64(i%130)*1.5, y + 12}
		for k, gu := range g.guests {
			if k < 10 {
				gu.pos = Vec{x0 + float64(i%130)*1.5, y + 20 + float64(k)*4}
			}
		}
		g.updateFootprints(1.0 / 60)
		g.drawFootprints(scr)
	}
	if fp.n == 0 || fp.n > fpMax {
		t.Fatalf("2P: %d prints", fp.n)
	}
}

func TestFootprintsAllocsPerFrame(t *testing.T) {
	g := fpSetup(t, false, "MIAMI BEACH")
	x0, y := fpRow(t, g, fpSand)
	g.updateFootprints(0.016)
	i := 0
	moving := testing.AllocsPerRun(50, func() {
		i++
		g.p.pos = Vec{x0 + float64(i%130)*1.5, y}
		g.updateFootprints(1.0 / 60)
	})
	idle := testing.AllocsPerRun(50, func() {
		g.p.pos = Vec{x0, y}
		g.updateFootprints(1.0 / 60)
	})
	t.Logf("update: %.1f allocs moving, %.1f idle", moving, idle)
	if moving > idle+0.5 {
		t.Errorf("update allocates when prints are made: %.1f vs idle %.1f", moving, idle)
	}
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	base := testing.AllocsPerRun(20, func() {
		for k := 0; k < 100; k++ {
			fpRect(scr, 1, 1, 2, 1, bgWhite(100))
		}
	}) / 100
	for k := 0; k < 130; k++ {
		g.p.pos = Vec{x0 + float64(k)*1.5, y}
		g.updateFootprints(1.0 / 60)
	}
	g.drawFootprints(scr)
	if fp.drawn == 0 {
		t.Fatal("nothing drawn")
	}
	fpRects = 0
	allocs := testing.AllocsPerRun(20, func() { g.drawFootprints(scr) })
	n := float64(fpRects) / 21
	t.Logf("draw: %.0f allocs for %v prints (bare DrawImage %.2f)", allocs, n, base)
	if allocs/n > base+0.6 {
		t.Errorf("draw allocates %.2f per print, bare DrawImage %.2f", allocs/n, base)
	}
}
