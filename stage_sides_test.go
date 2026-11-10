package main

import (
	"fmt"
	"testing"
)

// stageLevels: every level that has a band stage (the campaign and random pool, plus a few rolls of
// the generators that place one: the festival and the beach fallback).
func stageLevels() []*LevelDef {
	var out []*LevelDef
	for _, d := range builtinLevels() {
		out = append(out, d)
	}
	for s := int64(1); s <= 6; s++ {
		out = append(out, genNamed("festival", 500+s*37, fmt.Sprintf("FESTIVAL seed %d", s)))
		out = append(out, genNamed("beach", 900+s*41, fmt.Sprintf("PLAYA seed %d", s)))
	}
	var withStage []*LevelDef
	for _, d := range out {
		if d == nil {
			continue
		}
		for _, p := range d.Props {
			if p.Kind == "stage" {
				withStage = append(withStage, d)
				break
			}
		}
	}
	return withStage
}

// TestStageSidesArePassable: the user could not get past the sides of the stage. On every level
// with a stage, the player (SolidP foot box) and the guests (nav grid) must be able to walk off the
// platform sideways, and to go up and down beside it on both sides (from the front of the stage to
// behind it), all of it reachable from the start.
func TestStageSidesArePassable(t *testing.T) {
	levels := stageLevels()
	if len(levels) < 3 {
		t.Fatalf("expected several levels with a stage, got %d", len(levels))
	}
	for _, def := range levels {
		l := buildLevel(def)
		seen := reachable(l, def.Start)
		reach := func(p Vec) bool {
			for dy := -4.0; dy <= 4; dy += 4 {
				for dx := -4.0; dx <= 4; dx += 4 {
					if seen[[2]int{int(p.X+dx) / 4, int(p.Y+dy) / 4}] {
						return true
					}
				}
			}
			return false
		}
		for _, p := range l.Props {
			if p.Name != "stage" {
				continue
			}
			sx, sy := p.X, p.Y
			half := p.Foot.W / 2
			mid := sy - 30 // the middle of the platform
			for _, s := range []float64{-1, 1} {
				side := map[float64]string{-1: "left", 1: "right"}[s]
				// 1. walk straight off the platform sideways, no invisible wall
				for x := sx; s*(x-sx) <= half+24; x += s * 2 {
					if l.SolidP(x, mid) {
						t.Errorf("%s: stage at %.0f,%.0f: the %s side is blocked at x=%.0f (platform y=%.0f)", def.Title, sx, sy, side, x, mid)
						break
					}
				}
				// 2. a lane beside the stage, from in front of it to behind it, wide enough for a person
				back := sy - p.Foot.H - 14
				if back < 20 {
					back = 20
				}
				lane := false
				for x := sx + s*(half+8); s*(x-sx) <= half+64 && !lane; x += s * 2 {
					ok := true
					for y := sy + 12; y >= back; y -= 2 {
						if l.SolidP(x, y) || l.SolidP(x-4, y) || l.SolidP(x+4, y) {
							ok = false
							break
						}
					}
					if ok {
						lane = true
						// reachable by the player and walkable by the guests
						front, behind := Vec{x, sy + 12}, Vec{x, back}
						if !reach(front) || !reach(behind) || !reach(Vec{sx + s*(half-20), mid}) {
							t.Errorf("%s: stage %s side not reachable from the start (x=%.0f)", def.Title, side, x)
						}
						if path := l.Path(front, behind); path == nil {
							t.Errorf("%s: guests cannot walk the %s side of the stage (x=%.0f)", def.Title, side, x)
						} else {
							n, prev := 0.0, front
							for _, q := range path {
								n += dist(prev, q)
								prev = q
							}
							if n > 3*dist(front, behind)+40 {
								t.Errorf("%s: guests take a %.0f px detour round the %s side of the stage", def.Title, n, side)
							}
						}
					}
				}
				if !lane {
					t.Errorf("%s: stage at %.0f,%.0f: no free lane on its %s side between the front and the back", def.Title, sx, sy, side)
				}
			}
		}
	}
}

// TestNeonDanceFloorSides: the user's report (level 3, GLOW NEON: the rooftop with the dance floor in
// the middle and two helipads). The lit dance floor with its ropes must be walkable and can be left
// and rounded by both sides, top to bottom, and the DJ podium above it too; checked on the classic
// definition (-level 3) and on the campaign one (with its overrides).
func TestNeonDanceFloorSides(t *testing.T) {
	var defs []*LevelDef
	defs = append(defs, level3())
	for _, d := range builtinLevels() {
		if d.Title == "GLOW NEON" {
			defs = append(defs, d)
		}
	}
	if len(defs) < 2 {
		t.Fatalf("GLOW NEON missing from the campaign")
	}
	for _, def := range defs {
		l := buildLevel(def)
		seen := reachable(l, def.Start)
		reach := func(p Vec) bool {
			for dy := -4.0; dy <= 4; dy += 4 {
				for dx := -4.0; dx <= 4; dx += 4 {
					if seen[[2]int{int(p.X+dx) / 4, int(p.Y+dy) / 4}] {
						return true
					}
				}
			}
			return false
		}
		var fl Rect
		for _, f := range def.Floors {
			if f.Kind == "dance" {
				fl = f.R
			}
		}
		if fl.W == 0 {
			t.Fatalf("%s: no dance floor", def.Title)
		}
		c := fl.Center()
		for _, s := range []float64{-1, 1} {
			side := map[float64]string{-1: "left", 1: "right"}[s]
			for _, y := range []float64{fl.Y + 12, c.Y, fl.Y + fl.H - 8} { // off the floor sideways at three heights
				for x := c.X; s*(x-c.X) <= fl.W/2+30; x += s * 2 {
					if l.SolidP(x, y) {
						t.Errorf("%s: dance floor %s side blocked at %.0f,%.0f", def.Title, side, x, y)
						break
					}
				}
			}
			edge := c.X + s*fl.W/2
			lane := false
			for x := edge; s*(x-edge) <= 60 && !lane; x += s * 2 {
				ok := true
				for y := fl.Y - 10; y <= fl.Y+fl.H+10; y += 2 {
					if l.SolidP(x, y) || l.SolidP(x-4, y) || l.SolidP(x+4, y) {
						ok = false
						break
					}
				}
				if ok {
					lane = true
					top, bot := Vec{x, fl.Y - 10}, Vec{x, fl.Y + fl.H + 10}
					if !reach(top) || !reach(bot) {
						t.Errorf("%s: the %s side of the dance floor is not reachable from the start", def.Title, side)
					}
					if l.Path(bot, top) == nil {
						t.Errorf("%s: guests cannot walk the %s side of the dance floor", def.Title, side)
					}
				}
			}
			if !lane {
				t.Errorf("%s: no free lane along the %s side of the dance floor", def.Title, side)
			}
		}
		for _, p := range l.Props {
			if p.Name != "dj" {
				continue
			}
			// the podium: walk past it on both sides and get behind it
			for _, q := range []Vec{{p.Foot.X - 14, p.Foot.Y + p.Foot.H/2}, {p.Foot.X + p.Foot.W + 14, p.Foot.Y + p.Foot.H/2}, {p.X, p.Foot.Y - 10}, {p.X, p.Y + 20}} {
				if !reach(q) {
					t.Errorf("%s: cannot reach %v around the DJ podium", def.Title, q)
				}
			}
		}
	}
}
