package main

import (
	"math"
	"testing"
)

// The "walking on the sea" bug (MIAMI BEACH, reported on Windows): the sea of the beach parties is only painted in
// the background, with no floor and no solid, so the player walked into it. These tests go over every party of the
// campaign, the island and the random-mode rolls (1 and 2 players) and check that the deep water is solid, that the
// real movement of the player cannot carry him into it from the shore, that the guests' grid never reaches it, and
// that every place the party needs is still reachable. Swimming pools and the island's lake are the designed
// swimming places (Pools + Swim) and the piers and the landing of ITALIA are walkable decks: none of them counts.

// waterLevels: the campaign (with the island) and the random rolls.
func waterLevels(t *testing.T) []*LevelDef {
	t.Helper()
	return append(testLevels(t), jurassicLevel())
}

// deepWater: is (x,y) in water nobody may walk into? shore is how far past the drawn shore line it must be (the
// outline of an island or a ship is followed by 8 px solid cells: its edge may be touched by a few pixels).
func deepWater(d *LevelDef, x, y, shore float64) bool {
	if d.Shape != nil && !d.Shape(x, y, shore) {
		return true // the sea around the cruise and the island, the vacuum around the station
	}
	for _, o := range [5][2]float64{{0, 0}, {-shore, 0}, {shore, 0}, {0, -shore}, {0, shore}} {
		if !waterTop(d, x+o[0], y+o[1]) {
			return false // land (a quay, a deck, the sand) within the shore distance
		}
	}
	return true
}

// waterTop: is the topmost ground at (x,y) water (a "water" floor, or the painted sea of a beach party)?
func waterTop(d *LevelDef, x, y float64) bool {
	top, any := "", false
	for _, f := range d.Floors {
		if f.R.Has(x, y) {
			top, any = f.Kind, true
		}
	}
	return top == "water" || d.Sky == "sea" && d.Shape == nil && !any && y >= 0 && y < d.H
}

func hasWater(d *LevelDef) bool {
	if d.Shape != nil || d.Sky == "sea" {
		return true
	}
	for _, f := range d.Floors {
		if f.Kind == "water" {
			return true
		}
	}
	return false
}

// playerStarts: where one and two players begin (setupPlayers in twoplayers.go).
func playerStarts(l *Level, d *LevelDef) [][]Vec {
	a := l.NearestFree(Vec{d.Start.X - playerGap/2, d.Start.Y})
	b := l.NearestFree(Vec{d.Start.X + playerGap/2, d.Start.Y})
	if dist(a, b) < 20 {
		b = l.NearestFree(Vec{a.X + playerGap, a.Y})
		if dist(a, b) < 20 {
			b = l.NearestFree(Vec{a.X, a.Y + playerGap})
		}
	}
	return [][]Vec{{d.Start}, {a, b}}
}

// playerReach flood-fills, on a 2 px grid, every spot the feet of the player can stand on (SolidP free).
func playerReach(l *Level, from []Vec) (seen []bool, w, h int) {
	const s = 2.0
	w, h = int(l.W/s), int(l.H/s)
	seen = make([]bool, w*h)
	var q []int
	for _, p := range from {
		cx, cy := int(p.X/s), int(p.Y/s)
		if cx >= 0 && cy >= 0 && cx < w && cy < h && !seen[cy*w+cx] {
			seen[cy*w+cx] = true
			q = append(q, cy*w+cx)
		}
	}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		cx, cy := c%w, c/w
		for _, dd := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := cx+dd[0], cy+dd[1]
			if nx < 0 || ny < 0 || nx >= w || ny >= h || seen[ny*w+nx] {
				continue
			}
			if l.SolidP(float64(nx)*s+1, float64(ny)*s+1) {
				continue
			}
			seen[ny*w+nx] = true
			q = append(q, ny*w+nx)
		}
	}
	return seen, w, h
}

// walkPlayer moves the feet like Game.updatePlayer does (each axis on its own, blocked by SolidP), running.
func walkPlayer(l *Level, p Vec, dx, dy float64, frames int) Vec {
	n := math.Hypot(dx, dy)
	dx, dy = dx/n, dy/n
	const dt = 1.0 / 60
	for i := 0; i < frames; i++ {
		nx := p.X + dx*74*paceK*dt
		if !l.SolidP(nx, p.Y) {
			p.X = nx
		}
		ny := p.Y + dy*74*paceK*dt
		if !l.SolidP(p.X, ny) {
			p.Y = ny
		}
	}
	return p
}

// TestDeepWaterIsSolid: every spot of deep water (a grid over all of it) is solid for the player's feet, or
// out of reach of both players.
func TestDeepWaterIsSolid(t *testing.T) {
	for _, d := range waterLevels(t) {
		if !hasWater(d) {
			continue
		}
		l := buildLevel(d)
		for np, starts := range playerStarts(l, d) {
			seen, w, _ := playerReach(l, starts)
			bad, first := 0, Vec{}
			for y := 2.0; y < l.H-2; y += 4 {
				for x := 2.0; x < l.W-2; x += 4 {
					if !deepWater(d, x, y, 4) || l.SolidP(x, y) {
						continue
					}
					if seen[int(y/2)*w+int(x/2)] {
						if bad == 0 {
							first = Vec{x, y}
						}
						bad++
					}
				}
			}
			if bad > 0 {
				t.Errorf("%s (%d players): %d spots of deep water can be walked on, the first at %.0f,%.0f", d.Title, np+1, bad, first.X, first.Y)
			}
		}
	}
}

// TestCannotWalkIntoWater: from every spot of the shore the player can reach, running straight at the water for
// two seconds (and at 45 degrees) never leaves his feet in the water; the guests' grid never reaches it either.
func TestCannotWalkIntoWater(t *testing.T) {
	dirs := [][2]float64{{0, -1}, {0, 1}, {-1, 0}, {1, 0}, {1, -1}, {-1, -1}, {1, 1}, {-1, 1}}
	for _, d := range waterLevels(t) {
		if !hasWater(d) {
			continue
		}
		l := buildLevel(d)
		for np, starts := range playerStarts(l, d) {
			seen, w, h := playerReach(l, starts)
			tried, bad := 0, 0
			var at, end Vec
			for cy := 0; cy < h; cy += 2 {
				for cx := 0; cx < w; cx += 2 {
					if !seen[cy*w+cx] {
						continue
					}
					p := Vec{float64(cx)*2 + 1, float64(cy)*2 + 1}
					if deepWater(d, p.X, p.Y, 4) {
						bad++
						at, end = p, p
						continue
					}
					for _, dv := range dirs { // only the spots by the water: is there water 24 px that way?
						q := Vec{p.X + dv[0]*24, p.Y + dv[1]*24}
						if !deepWater(d, q.X, q.Y, 4) {
							continue
						}
						tried++
						if e := walkPlayer(l, p, dv[0], dv[1], 120); deepWater(d, e.X, e.Y, 4) {
							bad++
							at, end = p, e
						}
					}
				}
			}
			if bad > 0 {
				t.Errorf("%s (%d players): the player walks into the water %d times (from %.0f,%.0f to %.0f,%.0f)", d.Title, np+1, bad, at.X, at.Y, end.X, end.Y)
			}
			if d.Title == "MIAMI BEACH" && tried < 100 {
				t.Errorf("%s: only %d walks towards the sea were tried: the shore was not found", d.Title, tried)
			}
		}
		reach := l.reachFrom(d.Start)
		for i, ok := range reach {
			if c := cellCenter(i%l.cw, i/l.cw); ok && deepWater(d, c.X, c.Y, 4) {
				t.Errorf("%s: the guests can walk to %.0f,%.0f, in the water", d.Title, c.X, c.Y)
				break
			}
		}
	}
}

// TestWaterLevelsStillReachable: closing the water leaves every place of the party reachable from the start.
func TestWaterLevelsStillReachable(t *testing.T) {
	for _, d := range waterLevels(t) {
		if !hasWater(d) {
			continue
		}
		l := buildLevel(d)
		check := func(what string, p Vec, rad float64) {
			q := l.NearestFree(p)
			if dist(p, q) > rad || l.Path(d.Start, q) == nil {
				t.Errorf("%s: %s at %.0f,%.0f cannot be reached from the start", d.Title, what, p.X, p.Y)
			}
		}
		for kind, ps := range l.POIs {
			for _, p := range ps {
				check("the "+kind+" spot", p, 40)
			}
		}
		for _, r := range d.Roam {
			check("the roaming area", r.Center(), 70)
		}
		for _, r := range d.Dance {
			check("the dance floor", r.Center(), 60)
		}
		for _, pf := range d.Performers {
			check("the "+pf.Role, pf.Pos, 60)
		}
		for _, route := range d.Guards {
			for _, p := range route {
				check("a guard's route", p, 40)
			}
		}
		for _, a := range d.Animals {
			check("an animal", a.Pos, 40)
		}
	}
}

// TestMiamiSeaClosedOnTheShore: the invisible solid of the beach parties ends just past the top of the sand.
func TestMiamiSeaClosedOnTheShore(t *testing.T) {
	testLevels(t)
	for _, d := range []*LevelDef{level4(), genNamed("beach", 131, "")} {
		l := buildLevel(d)
		sy := seaShoreY(d)
		for x := 20.0; x < d.W-20; x += 10 {
			if !l.SolidP(x, sy-2) || !l.SolidP(x, 40) {
				t.Fatalf("%s: the sea at x=%.0f is not solid", d.Title, x)
			}
		}
		// on open sand the feet stop on the beach, at most a few pixels below the shore line
		p := walkPlayer(l, Vec{620, 160}, 0, -1, 240)
		if p.Y < sy+2 || p.Y > sy+10 {
			t.Errorf("%s: walking north the player stops at y=%.1f (shore at %.0f)", d.Title, p.Y, sy)
		}
	}
}
