package main

import "testing"

// The record trophy (t_pts8k, "Noche de récord"; the id stays, the goal went from 8000 to 6500 points in one party on
// 2026-11-11, owner): a clean full party WITHOUT the RETO reaches it (it made 7946 with the RACHA, streak.go), the
// RETO too, and a weak party (a few kills, slow, no chain) stays far below, so the trophy is not given away.
func TestTrophyPts8kStillReachable(t *testing.T) {
	var tr *Trophy
	for i := range trophyList {
		if trophyList[i].id == "t_pts8k" {
			tr = &trophyList[i]
		}
	}
	if tr == nil || tr.stat != "best" || tr.need != 6500 {
		t.Fatalf("t_pts8k: %+v", tr)
	}
	for _, challenge := range []bool{false, true} {
		g := scoreGame(t)
		g.challenge = challenge
		n := 0
		for gu := liveGuest(g, 0); gu != nil && n < 200; gu = liveGuest(g, 0) {
			g.time += 1.5 // one every second and a half: a chain, nobody sees it
			g.killGuest(gu, true)
			n++
		}
		g.finishScore()
		f := g.run.fin
		t.Logf("RETO %v: %d kills, total %d (base %d, chain %d, clean %d, streak %d best x%d, unseen %d, speed %d, x5 extra %d)",
			challenge, n, f.total, f.base, f.chain, f.clean, f.streak, f.streakBest, f.unseen, f.speed, f.finalExtra)
		if f.streak < 0 {
			t.Errorf("the streak took points away: %d", f.streak)
		}
		if f.total < tr.need {
			t.Errorf("a perfect party (RETO %v) makes %d points: the record trophy (%d) cannot be reached", challenge, f.total, tr.need)
		}
	}
}

// TestTrophyPts8kNotGivenAway: a mediocre party (a third of the guests, one every 20 s so no chain, no RETO) is far
// below the 6500 of the record trophy, and the trophy is not granted from its best.
func TestTrophyPts8kNotGivenAway(t *testing.T) {
	g := scoreGame(t)
	progressFrozen = true // checkTrophies saves: never the real progress (scoreGame puts the old value back)
	so := shopOwned
	t.Cleanup(func() { shopOwned = so })
	all := 0
	for _, gu := range g.guests {
		if !gu.out() {
			all++
		}
	}
	n := 0
	for gu := liveGuest(g, 0); gu != nil && n < all/3; gu = liveGuest(g, 0) {
		g.time += 20
		g.killGuest(gu, true)
		n++
	}
	g.finishScore()
	f := g.run.fin
	t.Logf("mediocre: %d of %d kills, total %d", n, all, f.total)
	if f.total*2 > 6500 {
		t.Errorf("a mediocre party makes %d points: too close to the record trophy (6500)", f.total)
	}
	g.progress.Stats = map[string]int{"best": f.total}
	g.trophyDirty = true
	g.checkTrophies()
	if g.progress.Trophies["t_pts8k"] {
		t.Errorf("record trophy granted with a best of %d", f.total)
	}
	g.progress.Stats["best"] = 6500
	g.trophyDirty = true
	g.checkTrophies()
	if !g.progress.Trophies["t_pts8k"] {
		t.Errorf("record trophy not granted at 6500")
	}
}
