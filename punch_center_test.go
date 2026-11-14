package main

import (
	"math"
	"testing"
)

// punchSkipIcons: parties whose punch is not checked here. The BIRTHDAY, the HOCKEY RINK and the 70s DISCO are checked
// again (v1.4.0: the bowls of the birthday and the disco moved out of their corners); the SPACE STATION keeps its bowl
// by the bar of the module (179 px from the dance floor), a ship that is checked by its own tests.
var punchSkipIcons = map[string]bool{"station": true}

// punchSpot: how central a punch bowl is: the share of the floor round it (a disc of 96 px) that can be walked TO from the
// bowl without crossing a wall (a flood fill kept inside the disc: a corner of a room gives about a quarter, the middle of a
// room nearly all of it), and how far it is from the dance floor (or from the middle of the party when there is none).
func punchSpot(l *Level, pr *Prop, reach []bool) (open, toDance float64) {
	use := l.NearestFree(Vec{pr.X, pr.Y + 14})
	const R = 96.0
	in := func(cx, cy int) bool {
		dx, dy := (float64(cx)+0.5)*navCell-use.X, (float64(cy)+0.5)*navCell-use.Y
		return dx*dx+dy*dy <= R*R
	}
	sx, sy := l.cellOf(use)
	n := 0
	for cy := sy - 13; cy <= sy+13; cy++ {
		for cx := sx - 13; cx <= sx+13; cx++ {
			if in(cx, cy) {
				n++
			}
		}
	}
	seen := map[[2]int]bool{{sx, sy}: true}
	q := [][2]int{{sx, sy}}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nc := [2]int{c[0] + d[0], c[1] + d[1]}
			if seen[nc] || !in(nc[0], nc[1]) || !l.cellFree(nc[0], nc[1]) || !reach[nc[1]*l.cw+nc[0]] {
				continue
			}
			seen[nc] = true
			q = append(q, nc)
		}
	}
	toDance = math.MaxFloat64
	for _, dr := range l.Def.Dance {
		toDance = math.Min(toDance, dist(use, dr.Center()))
	}
	if len(l.Def.Dance) == 0 {
		toDance = dist(use, Vec{l.W / 2, l.H / 2})
	}
	return float64(len(seen)) / float64(n), toDance
}

func checkPunches(t *testing.T, what string, l *Level, minOpen float64) int {
	reach := l.reachFrom(l.NearestFree(l.Def.Start))
	found := 0
	for _, pr := range l.Props {
		if pr.Name != "punch" {
			continue
		}
		found++
		open, toDance := punchSpot(l, pr, reach)
		if open < minOpen || toDance > 150 {
			t.Errorf("%s: the punch at %.0f,%.0f is in a corner (%.0f%% open floor round it, %.0f px from the dance floor)",
				what, pr.X, pr.Y, open*100, toDance)
		}
	}
	return found
}

// The punch bowls stand where the party is (open floor all round, near the dance floor), not in a forgotten corner.
func TestPunchBowlsAreCentral(t *testing.T) {
	n := killTargetLevels(t)
	total := 0
	for i := 0; i < n; i++ {
		if punchSkipIcons[campaignLevels[i].icon] {
			continue
		}
		g := playGame(t, i, int64(21+i))
		total += checkPunches(t, campaignLevels[i].icon, g.lvl, 0.6)
	}
	if total < 3 {
		t.Errorf("only %d punch bowls in the campaign", total)
	}
	// the random generators that put one (the house's living room, the casino's room): their rooms can be small (a living
	// room 120 px tall), so the open share asked is lower; the bowl must still stand by the dance floor
	for _, kind := range []string{"house", "casino", "yacht"} {
		for seed := int64(1); seed <= 6; seed++ {
			d := genNamed(kind, seed, "")
			if d == nil {
				continue
			}
			checkPunches(t, kind, buildLevel(d), 0.5)
		}
	}
}
