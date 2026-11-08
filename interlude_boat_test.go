package main

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// boatInside: is a pair of feet (x, y) inside the hull of a boat whose stern is at bx and waterline at by?
func boatInside(fx, fy, bx, by float64) bool {
	const m = 4.0
	return fx >= bx+m && fx <= bx+interBoatW-m && fy >= by-interBoatTop && fy <= by+m
}

// The passengers must stay inside the hull for the whole scene (bobbing, sinking, the bump, hiding, the push).
func TestBoatPassengersStayInHull(t *testing.T) {
	for _, two := range []bool{false, true} {
		for tt := 0.0; tt <= boatLen; tt += 0.05 {
			by := boatY(tt)
			var xs [2]float64
			for k := 0; k < 2; k++ {
				x, y, _, _, _, _, _ := boatPose(tt, k, two)
				xs[k] = x
				if !boatInside(x, y, boatBX, by) {
					t.Fatalf("2P=%v t=%.2f: passenger %d at (%.0f,%.0f) outside the hull at (%.0f,%.0f)", two, tt, k, x, y, boatBX, by)
				}
			}
			if two && math.Abs(xs[0]-xs[1]) < 30 {
				t.Fatalf("t=%.2f: both on the same seat", tt)
			}
		}
	}
}

// The test would catch the old bug: a passenger with an absolute position (not following the boat).
func TestBoatDetectsAbsoluteActor(t *testing.T) {
	absX, absY := boatBX+40, boatBaseY-44
	bad := false
	for tt := 0.0; tt <= boatLen; tt += 0.05 {
		if !boatInside(absX, absY, boatBX, boatY(tt)) {
			bad = true
		}
	}
	if !bad {
		t.Fatal("an actor fixed on the screen should leave the hull when the boat sinks or jumps")
	}
}

// the story has its beats in order, it is short, and it is not the street of the car
func TestBoatTimelineAndStory(t *testing.T) {
	if !(boatLeak < boatBail0 && boatBail0 < boatFin && boatFin < boatBump && boatBump < boatJump && boatJump < boatApex &&
		boatApex < boatLand && boatLand < boatTurn && boatTurn < boatPush && boatPush < boatFade && boatFade < boatLen) {
		t.Fatal("the boat timeline is out of order")
	}
	if boatLen > 20 || boatLen < 10 {
		t.Fatalf("the boat lasts %.1f s: it must be between 10 and 20", boatLen)
	}
	if len(interludePlaces(interBoat)) != 0 {
		t.Error("the boat has its own scene: no street memories")
	}
	// the water rises with the leak, never goes negative, and has run out over the stern at the end
	if boatWater(boatLeak-0.1) != 0 || boatWater(boatBump-0.1) < 12 || boatWater(boatLen) > 0.01 {
		t.Errorf("water: %.1f %.1f %.1f", boatWater(boatLeak-0.1), boatWater(boatBump-0.1), boatWater(boatLen))
	}
	for tt := 0.0; tt <= boatLen; tt += 0.05 {
		if w := boatWater(tt); w < 0 || w > 24.001 {
			t.Fatalf("t=%.2f: water %v", tt, w)
		}
		if boatSpeed(tt) < 0 {
			t.Fatalf("t=%.2f: negative speed", tt)
		}
	}
	// the sea goes by faster once the shark pushes and never goes backwards
	prev := -1.0
	for tt := 0.0; tt <= boatLen; tt += 0.05 {
		d := boatDist(tt)
		if d < prev {
			t.Fatalf("the sea goes backwards at %.2f", tt)
		}
		prev = d
	}
	if boatSpeed(boatPush+2) < 10*boatSpeed(1) {
		t.Error("the push must be a lot faster than the rowing")
	}
	// the shark: leaps over the boat (above the heads at the top), lands on the right, then pushes from the stern
	x, y, _, mode, _ := boatSharkPos(boatApex)
	if mode != 1 || y > 200 || x < boatBX || x > boatBX+boatSeatDX[1]+100 {
		t.Errorf("at the top of the leap the shark is at (%.0f,%.0f) mode %d", x, y, mode)
	}
	if x, _, _, mode, _ := boatSharkPos(boatLen); mode != 4 || x > boatBX {
		t.Errorf("at the end the shark pushes from the stern (x %.0f mode %d)", x, mode)
	}
	// different things happen in the phases, for both passengers (2P)
	for k := 0; k < 2; k++ {
		seen := map[int]bool{}
		for _, tt := range []float64{0.5, boatLeak + 1, boatBail0 + 0.5, boatFin + 0.5, boatApex, boatPush + 1} {
			_, _, a, _, _, _, _ := boatPose(tt, k, true)
			seen[a] = true
		}
		if len(seen) < 3 {
			t.Fatalf("passenger %d (2P): only %d different actions", k, len(seen))
		}
	}
}

// Drawing the boat interlude, 1P and 2P, from the start to the end must not panic. (Not checked: how it looks.)
func TestBoatInterludeDraws(t *testing.T) {
	g := auditGame(t) // fonts and px, no window
	oldPx := px
	if px == nil {
		px = ebiten.NewImage(1, 1)
		px.Fill(color.White)
	}
	t.Cleanup(func() { px = oldPx; g.inter = nil })
	for _, two := range []bool{false, true} {
		in := &interState{kind: interBoat, two: two, did: map[string]bool{}}
		g.inter = in
		dst := ebiten.NewImage(screenW, screenH)
		for tt := 0.0; tt < boatLen; tt += 0.3 {
			in.t = tt
			g.drawInterlude(dst)
		}
	}
}
