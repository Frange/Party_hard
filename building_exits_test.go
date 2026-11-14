package main

import (
	"fmt"
	"testing"
)

// buildingExits finds the buildings of a level and how many ways out each one has.
//
// The doors of a level (Def.Doors: every gap left in a wall) are closed for a moment and the walkable cells are split
// into areas (flood fill). An area is indoor when most of its floor is a covered floor (Level.Outdoor false). The indoor
// areas joined by doors between them make one building (a house and its rooms); its EXITS are the doors from it to an
// outdoor area (only gaps in building walls are doors: fences, hedges and rails are left open). Doors stopped by
// furniture do not count. It returns, for every building, its exits and a short
// description (for the error message).
func buildingExits(l *Level) (exits []int, what []string) {
	reach := l.reachFrom(l.NearestFree(l.Def.Start))
	n := l.cw * l.ch
	shut := make([]bool, n)
	copy(shut, l.blocked)
	type doorCells struct {
		d     Door
		cells []int
	}
	var doors []doorCells
	for _, dr := range l.Def.Doors {
		if !buildingWall(dr.Style) { // a gap in a fence, a hedge or a rail is no door of a building
			continue
		}
		var dc doorCells
		dc.d = dr
		for cy := maxI(0, int(dr.R.Y/navCell)); cy < l.ch && float64(cy)*navCell < dr.R.Y+dr.R.H; cy++ {
			for cx := maxI(0, int(dr.R.X/navCell)); cx < l.cw && float64(cx)*navCell < dr.R.X+dr.R.W; cx++ {
				i := cy*l.cw + cx
				if !l.blocked[i] {
					dc.cells = append(dc.cells, i)
				}
				shut[i] = true
			}
		}
		doors = append(doors, dc)
	}
	// the areas
	area := make([]int, n)
	for i := range area {
		area[i] = -1
	}
	var indoor []bool
	var size []int
	for s := 0; s < n; s++ {
		if shut[s] || area[s] >= 0 || !reach[s] {
			continue
		}
		id := len(size)
		in, cnt := 0, 0
		q := []int{s}
		area[s] = id
		for len(q) > 0 {
			c := q[0]
			q = q[1:]
			cnt++
			if !l.Outdoor(Vec{(float64(c%l.cw) + 0.5) * navCell, (float64(c/l.cw) + 0.5) * navCell}) {
				in++
			}
			cx, cy := c%l.cw, c/l.cw
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := cx+d[0], cy+d[1]
				if nx < 0 || ny < 0 || nx >= l.cw || ny >= l.ch {
					continue
				}
				j := ny*l.cw + nx
				if !shut[j] && area[j] < 0 && reach[j] {
					area[j] = id
					q = append(q, j)
				}
			}
		}
		size = append(size, cnt)
		indoor = append(indoor, in*10 >= cnt*6)
	}
	// which areas every open door joins (the areas of the cells next to its open cells)
	link := make([]map[int]bool, len(doors))
	for k, dc := range doors {
		link[k] = map[int]bool{}
		// cells of the door joined to each other, then the areas round them
		inDoor := map[int]bool{}
		for _, c := range dc.cells {
			inDoor[c] = true
		}
		for _, c := range dc.cells {
			if !reach[c] {
				continue
			}
			cx, cy := c%l.cw, c/l.cw
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := cx+d[0], cy+d[1]
				if nx < 0 || ny < 0 || nx >= l.cw || ny >= l.ch {
					continue
				}
				if a := area[ny*l.cw+nx]; a >= 0 && !inDoor[ny*l.cw+nx] {
					link[k][a] = true
				}
			}
		}
	}
	// buildings: indoor areas joined by doors that link only indoor areas
	bld := make([]int, len(size))
	for i := range bld {
		bld[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for bld[i] != i {
			i = bld[i]
		}
		return i
	}
	for _, lk := range link {
		var ins []int
		for a := range lk {
			if indoor[a] {
				ins = append(ins, a)
			}
		}
		for _, a := range ins[min(1, len(ins)):] {
			bld[find(a)] = find(ins[0])
		}
	}
	type info struct {
		exits, cells int
		at           Vec
	}
	got := map[int]*info{}
	for a := range size {
		if !indoor[a] || size[a] < 30 {
			continue
		}
		r := find(a)
		if got[r] == nil {
			got[r] = &info{}
		}
		got[r].cells += size[a]
	}
	for k, lk := range link {
		seen := map[int]bool{}
		hasOut := false
		for a := range lk {
			if !indoor[a] {
				hasOut = true
			}
		}
		if !hasOut {
			continue
		}
		for a := range lk {
			if indoor[a] {
				r := find(a)
				if got[r] != nil && !seen[r] {
					seen[r] = true
					got[r].exits++
					got[r].at = doors[k].d.R.Center()
				}
			}
		}
	}
	for r, in := range got {
		if in.exits == 0 {
			continue // no door of a building wall leads out of it: an open lot (a skate park, a rink), not a building
		}
		exits = append(exits, in.exits)
		what = append(what, fmt.Sprintf("building of %d cells (area %d, last exit at %.0f,%.0f)", in.cells, r, in.at.X, in.at.Y))
	}
	return exits, what
}

// buildingWall: the styles of the walls of a building (not fences, hedges, rails or the void round a ship or a station).
func buildingWall(style int) bool {
	switch style {
	case WallFence, WallHedge, WallRail:
		return false
	}
	return style >= WallPlaster && style <= WallRail
}

// No building of a party is a closed place with a single way in and out (a trap for the player with the police at the
// door): every one has two exits at least, or it is open.
func TestBuildingsHaveTwoExits(t *testing.T) {
	n := killTargetLevels(t)
	for i := 0; i < n; i++ {
		g := playGame(t, i, int64(31+i)) // every party, the BIRTHDAY, HOCKEY and 70s DISCO too (v1.4.0)
		ex, what := buildingExits(g.lvl)
		for k, e := range ex {
			if e < 2 {
				t.Errorf("L%d (%s): %s has %d exit(s)", i+1, campaignLevels[i].icon, what[k], e)
			}
		}
	}
	// the buildings of the random mode
	for _, kind := range []string{"house", "club", "pub", "cinema", "mall", "market", "casino"} {
		for seed := int64(1); seed <= 8; seed++ {
			d := genNamed(kind, seed, "")
			if d == nil {
				continue
			}
			ex, what := buildingExits(buildLevel(d))
			for k, e := range ex {
				if e < 2 {
					t.Errorf("random %s seed %d: %s has %d exit(s)", kind, seed, what[k], e)
				}
			}
		}
	}
}

// streetGates: the open gates of the street fence of genStreet (y 444) of a level: gaps with a walkable, reachable cell.
func streetGates(l *Level) int {
	reach := l.reachFrom(l.NearestFree(l.Def.Start))
	n := 0
	for _, dr := range l.Def.Doors {
		if dr.R.Y != 444 || !dr.Horiz {
			continue
		}
		open := false
		for x := dr.R.X + 4; x < dr.R.X+dr.R.W-4 && !open; x += navCell / 2 {
			for _, y := range []float64{dr.R.Y - 10, dr.R.Y + dr.R.H + 10} {
				cx, cy := l.cellOf(Vec{x, y})
				if l.cellFree(cx, cy) && reach[cy*l.cw+cx] {
					open = true
				}
			}
		}
		if open {
			n++
		}
	}
	return n
}

// The HOCKEY RINK and the 70s DISCO are closed grounds (wall, fences and the street fence): they had one single way in
// from the street, and the disco hall one single door (its side walls also stopped 14 px short of the front wall). Now
// both grounds have two gates in the street fence and the hall is a building with two exits, in every layout of the dice.
func TestClosedVenuesHaveTwoWaysIn(t *testing.T) {
	for _, icon := range []string{"hockey", "disco70"} {
		i := campaignIdx(icon)
		for seed := int64(1); seed <= 6; seed++ {
			d := genNamed(campaignLevels[i].kind, seed*97, "")
			if d == nil {
				t.Fatalf("%s seed %d: no level", icon, seed)
			}
			l := buildLevel(d)
			if n := streetGates(l); n < 2 {
				t.Errorf("%s seed %d: %d open gate(s) in the street fence", icon, seed, n)
			}
			if icon == "disco70" {
				ex, what := buildingExits(l)
				if len(ex) != 1 || ex[0] < 2 {
					t.Errorf("disco70 seed %d: the hall should be one building with 2 exits: %v %v", seed, ex, what)
				}
			}
		}
		g := playGame(t, i, int64(31+i))
		if n := streetGates(g.lvl); n < 2 {
			t.Errorf("%s (campaign): %d open gate(s) in the street fence", icon, n)
		}
	}
}
