package main

import (
	"math"
	"testing"
)

// hideEntrySpots: spots around a hiding prop from where the player can press E to go in (within the
// interact reach of game.go, his collision box free): below, above, left, right and the corners.
func hideEntrySpots(l *Level, pr *Prop) []Vec {
	f := pr.Foot
	cx, cy := f.X+f.W/2, f.Y+f.H/2
	cands := []Vec{
		{cx, f.Y + f.H + 6}, {cx, f.Y - 6}, {f.X - 8, cy}, {f.X + f.W + 8, cy},
		{f.X - 6, f.Y + f.H + 5}, {f.X + f.W + 6, f.Y + f.H + 5}, {f.X - 6, f.Y - 5}, {f.X + f.W + 6, f.Y - 5},
		{f.X + 3, f.Y + f.H + 10}, {f.X + f.W - 3, f.Y - 10},
	}
	var out []Vec
	for _, q := range cands {
		if q.X < 2 || q.Y < 2 || q.X > l.W-2 || q.Y > l.H-2 || l.SolidP(q.X, q.Y) || distRect(q, f) > 16 {
			continue
		}
		out = append(out, q)
	}
	return out
}

// hideWalkReach: can the player WALK from a to b (b within 3 px of a reached spot), staying inside a
// window of radius r around a? Breadth-first on a 3 px grid of spots his collision box fits on. The
// second value is false when b cannot be reached that way (another room, the far side of a wall).
func hideWalkReach(l *Level, a, b Vec, r float64) (float64, bool) {
	if dist(a, b) < 0.5 {
		return 0, true
	}
	const st = 3.0
	n := int(r/st)*2 + 1
	c := n / 2
	seen := make([]int32, n*n)
	for i := range seen {
		seen[i] = -1
	}
	type nd struct{ i, j int }
	q := []nd{{c, c}}
	seen[c*n+c] = 0
	for h := 0; h < len(q); h++ {
		cur := q[h]
		d0 := seen[cur.j*n+cur.i]
		p := Vec{a.X + float64(cur.i-c)*st, a.Y + float64(cur.j-c)*st}
		if dist(p, b) <= st {
			return float64(d0) * st, true
		}
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			ni, nj := cur.i+d[0], cur.j+d[1]
			if ni < 0 || nj < 0 || ni >= n || nj >= n || seen[nj*n+ni] >= 0 {
				continue
			}
			v := Vec{a.X + float64(ni-c)*st, a.Y + float64(nj-c)*st}
			if l.SolidP(v.X, v.Y) {
				continue
			}
			seen[nj*n+ni] = d0 + 1
			q = append(q, nd{ni, nj})
		}
	}
	return 0, false
}

// hideGameOn: a one/two player game on the level def (twoGame of twoplayers_test.go, level swapped).
func hideGameOn(t *testing.T, def *LevelDef, two bool) *Game {
	t.Helper()
	g := twoGame(two)
	g.lvl = buildLevel(def)
	// no stashes: their dice are seeded from the clock (stash.go), and about one run in six a
	// closet held an object, so the first E searched it instead of hiding (a flaky failure)
	prevStash, prevSeed := stash, stashSeed
	stash = stashState{lvl: g.lvl, item: map[*Prop]*stashItem{}}
	t.Cleanup(func() { stash, stashSeed = prevStash, prevSeed })
	g.p.pos = def.Start
	if g.p2 != nil {
		g.p2.pos = g.lvl.NearestFree(Vec{def.Start.X + 30, def.Start.Y})
	}
	return g
}

// hideKindsOf: the props of a level where the player can hide.
func hideKindsOf(l *Level) []*Prop {
	var out []*Prop
	for _, pr := range l.Props {
		if pr.K != nil && pr.K.Hide && !pr.gone {
			out = append(out, pr)
		}
	}
	return out
}

// The testers' bug: party 1, the bush at the top right on the roof edge of the house (and its twin at
// the top left). He went in from the garden and came out INSIDE the house. Now he comes out at the
// very spot he went in from, which he can walk to from where he was.
func TestHideExitPartyOneRoofBush(t *testing.T) {
	defs := testLevels(t)
	g := hideGameOn(t, defs[0], false)
	found := 0
	for _, pr := range hideKindsOf(g.lvl) {
		if pr.Name != "bush" || pr.Y > 100 {
			continue
		}
		found++
		old := g.lvl.NearestFree(Vec{pr.X, pr.Y + 16}) // the old exit
		for _, e := range hideEntrySpots(g.lvl, pr) {
			if _, ok := hideWalkReach(g.lvl, e, old, 160); ok {
				t.Fatalf("bush %.0f,%.0f: the old exit %v was reachable from %v: the test does not show the bug", pr.X, pr.Y, old, e)
			}
			g.p.pos = e
			g.interact()
			if !g.p.hidden || g.p.hideProp != pr {
				t.Fatalf("bush %.0f,%.0f from %v: he must hide in it", pr.X, pr.Y, e)
			}
			g.interact()
			if g.p.hidden || g.p.pos != e {
				t.Fatalf("bush %.0f,%.0f: went in from %v, came out at %v (hidden=%v)", pr.X, pr.Y, e, g.p.pos, g.p.hidden)
			}
		}
	}
	if found != 2 {
		t.Fatalf("party 1 has two bushes on the top edge, found %d", found)
	}
}

// Every hiding place of every party (every party of the campaign and the generated ones), from every
// side, with one and two players: in with E, out with E, at the very spot he went in from.
func TestHideExitSameSpotEveryLevel(t *testing.T) {
	defs := testLevels(t)
	for _, two := range []bool{false, true} {
		kinds := map[string]bool{}
		for li, def := range defs {
			g := hideGameOn(t, def, two)
			for _, pr := range hideKindsOf(g.lvl) {
				kinds[pr.Name] = true
				entered := 0
				for _, e := range hideEntrySpots(g.lvl, pr) {
					for _, who := range []int{0, 1} {
						if who == 1 && !two {
							continue
						}
						pl := g.p
						if who == 1 {
							pl = g.p2
						}
						pl.pos = e
						if o := g.otherPlayer(pl); o != nil {
							o.pos = Vec{e.X + 60, e.Y} // out of the way (the busy case has its own test)
						}
						g.asPlayer(who, g.interact)
						if !pl.hidden {
							continue // something closer took the E (a phone, a stash...): not a hiding case
						}
						entered++
						hp := pl.hideProp
						if pl.pos != (Vec{hp.X, hp.Y - 4}) {
							t.Fatalf("L%d %s: hidden player drawn at %v, not in the prop", li+1, hp.Name, pl.pos)
						}
						g.asPlayer(who, g.interact)
						if pl.hidden || pl.pos != e {
							t.Errorf("L%d %s (%s) %.0f,%.0f, player %d: in from %v, out at %v", li+1, def.Title, hp.Name, hp.X, hp.Y, who+1, e, pl.pos)
						}
					}
				}
				if entered == 0 && len(hideEntrySpots(g.lvl, pr)) > 0 {
					t.Errorf("L%d %s: could not hide in %s at %.0f,%.0f from any side", li+1, def.Title, pr.Name, pr.X, pr.Y)
				}
			}
		}
		for _, k := range []string{"bush", "closet", "dumpster", "portapotty"} {
			if !kinds[k] {
				t.Errorf("no level has a %s to hide in: the test lost its coverage", k)
			}
		}
	}
}

// Two players: the partner stands on the spot player 1 went in from. Player 1 comes out at the nearest
// free spot, not on top of him, not in a solid, and he can walk there from the entry spot.
func TestHideExitPartnerOnTheSpot(t *testing.T) {
	defs := testLevels(t)
	checked := 0
	for li, def := range defs[:len(campaignLevels)] {
		g := hideGameOn(t, def, true)
		for _, pr := range hideKindsOf(g.lvl) {
			es := hideEntrySpots(g.lvl, pr)
			if len(es) == 0 {
				continue
			}
			e := es[0]
			g.p.pos, g.p2.pos = e, Vec{e.X + 80, e.Y}
			g.interact()
			if !g.p.hidden {
				continue
			}
			g.p2.pos = e // the partner walks onto the spot
			g.interact()
			q := g.p.pos
			checked++
			switch {
			case g.p.hidden:
				t.Fatalf("L%d %s: he must come out", li+1, pr.Name)
			case dist(q, g.p2.pos) < hideExitBusy:
				t.Errorf("L%d %s: came out on top of the partner (%v vs %v)", li+1, pr.Name, q, g.p2.pos)
			case g.lvl.SolidP(q.X, q.Y):
				t.Errorf("L%d %s: came out inside a solid at %v", li+1, pr.Name, q)
			case dist(q, e) > 16:
				t.Errorf("L%d %s: came out %.0f px away from the entry spot", li+1, pr.Name, dist(q, e))
			}
			if d, ok := hideWalkReach(g.lvl, e, q, 60); !ok || d > 30 {
				t.Errorf("L%d %s: the exit %v is not a short walk from the entry %v (ok=%v d=%.0f)", li+1, pr.Name, q, e, ok, d)
			}
		}
	}
	if checked < 25 {
		t.Fatalf("only %d busy exits checked", checked)
	}
}

// The partner hidden in another bush does not count as standing on the spot (his position is the bush).
func TestHideExitHiddenPartnerDoesNotBlock(t *testing.T) {
	g := hideGameOn(t, testLevels(t)[0], true)
	prs := hideKindsOf(g.lvl)
	e := hideEntrySpots(g.lvl, prs[0])[0]
	g.p.pos = e
	g.interact()
	g.p2.hidden, g.p2.hideProp, g.p2.pos = true, prs[1], e // a stale position on the spot
	g.interact()
	if g.p.pos != e {
		t.Fatalf("a hidden partner must not push him off his spot: %v vs %v", g.p.pos, e)
	}
}

// The hiding place moved or vanished while he was inside: he still comes out where he went in, and a
// vanished one throws him out at once (next player update) at that spot.
func TestHideExitPropMovedOrGone(t *testing.T) {
	g := hideGameOn(t, testLevels(t)[0], false)
	var pr *Prop
	var e Vec
	for _, c := range hideKindsOf(g.lvl) {
		if es := hideEntrySpots(g.lvl, c); c.Name == "bush" && len(es) > 0 {
			pr, e = c, es[0]
			break
		}
	}
	g.p.pos = e
	g.interact()
	pr.X += 120
	pr.Foot.X += 120
	g.interact()
	if g.p.hidden || g.p.pos != e {
		t.Fatalf("moved bush: out at %v, went in from %v", g.p.pos, e)
	}
	pr.X -= 120
	pr.Foot.X -= 120
	g.p.pos = e
	g.interact()
	if !g.p.hidden {
		t.Fatal("must hide again")
	}
	pr.gone = true
	t.Cleanup(func() { pr.gone = false })
	g.updatePlayer(1.0 / 60)
	if g.p.hidden || g.p.pos != e {
		t.Fatalf("vanished bush: hidden=%v at %v, went in from %v", g.p.hidden, g.p.pos, e)
	}
}

// Something solid now covers the entry spot: out at the nearest free spot, never through a wall, never
// inside a solid, and close by.
func TestHideExitSpotCovered(t *testing.T) {
	defs := testLevels(t)
	for li, def := range defs[:len(campaignLevels)] {
		g := hideGameOn(t, def, false)
		for _, pr := range hideKindsOf(g.lvl) {
			es := hideEntrySpots(g.lvl, pr)
			if len(es) == 0 {
				continue
			}
			e := es[len(es)-1]
			g.p.pos = e
			g.interact()
			if !g.p.hidden {
				continue
			}
			saved := g.lvl.solidsP
			g.lvl.solidsP = append(append([]Rect(nil), saved...), Rect{e.X - 6, e.Y - 4, 12, 8})
			g.interact()
			q := g.p.pos
			if g.p.hidden || g.lvl.SolidP(q.X, q.Y) || dist(q, e) > 24 {
				t.Errorf("L%d %s: covered spot %v, came out at %v (hidden=%v)", li+1, pr.Name, e, q, g.p.hidden)
			}
			n := int(dist(q, e)/1.5) + 1
			for i := 0; i <= n; i++ {
				f := float64(i) / float64(n)
				if g.lvl.hideWallAt(Vec{e.X + (q.X-e.X)*f, e.Y + (q.Y-e.Y)*f}) {
					t.Errorf("L%d %s: the way from %v to the exit %v crosses a wall", li+1, pr.Name, e, q)
					break
				}
			}
			g.lvl.solidsP = saved
		}
	}
}

// Dying while hidden: he falls at the spot he went in from, not inside the bush.
func TestHideDeathFallsAtEntry(t *testing.T) {
	g := hideGameOn(t, testLevels(t)[0], true) // two players: the party goes on
	pr := hideKindsOf(g.lvl)[0]
	e := hideEntrySpots(g.lvl, pr)[0]
	g.p.pos = e
	g.interact()
	g.die("test")
	if g.p.hidden || g.p.pos != e {
		t.Fatalf("dead while hidden: hidden=%v at %v, went in from %v", g.p.hidden, g.p.pos, e)
	}
}

// hideExitPos never answers NaN or a spot off the map, also with no entry spot recorded.
func TestHideExitWithoutRecordedSpot(t *testing.T) {
	g := hideGameOn(t, testLevels(t)[0], false)
	for _, pr := range hideKindsOf(g.lvl) {
		g.p.hidden, g.p.hideProp, g.p.hideFromOK, g.p.pos = true, pr, false, Vec{pr.X, pr.Y - 4}
		q := g.hideExitPos(g.p)
		if math.IsNaN(q.X) || g.lvl.SolidP(q.X, q.Y) || q.X < 0 || q.Y < 0 || q.X > g.lvl.W || q.Y > g.lvl.H {
			t.Errorf("%s %.0f,%.0f: fallback exit %v", pr.Name, pr.X, pr.Y, q)
		}
	}
}
