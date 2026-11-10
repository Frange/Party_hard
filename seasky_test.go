package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// the sun's reflection rocks a few whole pixels, slowly, each line with its own phase
func TestSeaReflectionSwell(t *testing.T) {
	for i := 0; i < seaReflN; i++ {
		seen := map[float64]bool{}
		prev := seaReflShift(0, i)
		for k := 0; k <= 2400; k++ {
			tm := float64(k) / 60
			s := seaReflShift(tm, i)
			if s != math.Round(s) || math.Abs(s) > 4 {
				t.Fatalf("line %d at %.2f: shift %v (want whole pixels within 4)", i, tm, s)
			}
			if math.Abs(s-prev) > 1 {
				t.Fatalf("line %d at %.2f: jumped from %v to %v in one frame", i, tm, prev, s)
			}
			prev = s
			seen[s] = true
		}
		if len(seen) < 4 {
			t.Errorf("line %d barely moves: %v", i, seen)
		}
	}
	same := 0
	for i := 1; i < seaReflN; i++ {
		if seaReflShift(3.1, i) == seaReflShift(3.1, i-1) && seaReflShift(5.7, i) == seaReflShift(5.7, i-1) {
			same++
		}
	}
	if same > seaReflN/2 {
		t.Errorf("the lines move together (%d pairs in step): each needs its own phase", same)
	}
}

// the ship drifts left to right and only wraps while it is off screen
func TestSeaShipDrift(t *testing.T) {
	px, _ := seaShipPos(0)
	if px != seaShipX0 {
		t.Fatalf("at t=0 the ship should be where it used to stand: %v", px)
	}
	wraps := 0
	for k := 1; k <= 60*500; k++ {
		x, y := seaShipPos(float64(k) / 60)
		if y < seaShipY || y > seaShipY+1 {
			t.Fatalf("bobbing too much: y=%v", y)
		}
		switch {
		case x < px:
			wraps++
			if px < screenW || x+seaShipW > 0 {
				t.Fatalf("visible jump from %v to %v", px, x)
			}
		case x-px > 1:
			t.Fatalf("too fast: %v -> %v", px, x)
		}
		px = x
	}
	if wraps == 0 {
		t.Fatal("the ship never came round again")
	}
}

// the reflection and the ship are drawn every frame on the sea levels (and baked when the
// animation is off)
func TestSeaSkyDrawn(t *testing.T) {
	g := ambientSetup(t)
	i := ambientLevelIdx("MIAMI BEACH")
	if i < 0 {
		t.Fatal("no MIAMI BEACH")
	}
	ambientOn = true
	g.loadLevel(i)
	g.lvl.graded = true
	if !seaSkyMoving(g.lvl.Def) {
		t.Fatal("Miami should move its reflection")
	}
	if bot := seaWaterBottom(g.lvl.Def); bot != 90 {
		t.Errorf("the sand starts at 90, got %v", bot)
	}
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	bgCount = 0
	g.time = 7
	g.drawSeaSky(scr, g.time)
	if want := 19; bgCount != want { // 19 lines above the sand (the ship is a sprite now)
		t.Errorf("drew %d rectangles, want %d", bgCount, want)
	}
	// the horizon ship is a sprite as wide as the old hull, its hull row filled under its top-left corner + 8
	hs := genHorizonShip(seaShipCol)
	if float64(hs.w) != seaShipW {
		t.Errorf("horizon ship %d px wide, want %v", hs.w, seaShipW)
	}
	hull := 0
	for x := 0; x < hs.w; x++ {
		if hs.get(x, 10).A == 255 {
			hull++
		}
	}
	if hull < 36 {
		t.Errorf("the horizon ship has no hull (%d px)", hull)
	}
	seaSkyStill = true
	if seaSkyMoving(g.lvl.Def) {
		t.Error("a still picture must bake the reflection")
	}
	seaSkyStill = false
	ambientOn = false
	if seaSkyMoving(g.lvl.Def) {
		t.Error("ambientOn=false must bake the reflection")
	}
	ambientOn = true
}

// the new palm: four different variants, standing on the bottom row, same size and footprint
func TestPalmArt(t *testing.T) {
	// the code drawing is tested directly: it must not depend on an exported asset replacing the prop
	k := &PropKind{W: 48, H: 72, FW: 8, FH: 6, Vars: 4}
	var prev []byte
	for v := 0; v < 4; v++ {
		b := newPix(k.W+2, k.H+2)
		b.ox, b.oy = 1, 1
		drawPalm(b, v)
		if b.get(24, 71).A == 0 && b.get(23, 71).A == 0 && b.get(25, 71).A == 0 {
			t.Errorf("variant %d: the trunk does not reach the ground", v)
		}
		for x := -1; x <= k.W; x++ {
			if b.get(x, k.H).A != 0 {
				t.Errorf("variant %d: drawn below the sprite at x=%d", v, x)
				break
			}
		}
		greens := 0
		for y := 0; y < 40; y++ {
			for x := 0; x < k.W; x++ {
				if c := b.get(x, y); c.A > 0 && c.G > c.R && c.G > c.B {
					greens++
				}
			}
		}
		if greens < 350 {
			t.Errorf("variant %d: a thin crown (%d green pixels)", v, greens)
		}
		if prev != nil && string(prev) == string(b.p) {
			t.Errorf("variant %d is the same as the one before", v)
		}
		prev = append([]byte(nil), b.p...)
	}
}
