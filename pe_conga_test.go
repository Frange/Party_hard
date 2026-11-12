package main

import (
	"math/rand"
	"testing"
)

// ---- the conga in (almost) every party (pe_conga.go, 2026-11-11, owner: "nunca la he visto") ----

// The conga fits 25 of the 26 campaign parties (all but the space station), comes in about a third of the parties
// where it fits (it was 22 % of 9 levels) and the planner keeps it in the plan: the measure is the real partyPlan with
// the memory of the last parties, as in pe_freq_test.
func TestPevCongaFitsAndComesOften(t *testing.T) {
	g := partyGame(t)
	in := peEligible(g, "conga")
	t.Logf("conga fits %d of %d campaign parties: %v", len(in), len(campaignLevels), in)
	if len(in) != len(campaignLevels)-1 {
		t.Errorf("the conga should fit every party but the station: %d of %d", len(in), len(campaignLevels))
	}
	for _, i := range in {
		if levelDefs[i].Icon == "station" {
			t.Error("conga in the space station")
		}
	}
	e := partyByID("conga")
	if e.Prob < 0.33 || e.Prob > 0.37 || !e.Big || e.Group != "dance" {
		t.Errorf("conga: chance %.2f big %v group %q", e.Prob, e.Big, e.Group)
	}
	r := rand.New(rand.NewSource(99))
	var hist [][]string
	with, fit := 0, 0
	for k := 0; k < 4000; k++ {
		idx := r.Intn(len(campaignLevels))
		tr := partyTraitsOf(levelDefs[idx], idx)
		plan := partyPlan(r, g, tr, hist, false)
		var ids []string
		for _, p := range plan {
			ids = append(ids, p.ev.ID)
		}
		hist = append(hist, ids)
		if len(hist) > partyMemory {
			hist = hist[1:]
		}
		if !e.Eligible(g, tr) {
			continue
		}
		fit++
		for _, id := range ids {
			if id == "conga" {
				with++
				break
			}
		}
	}
	f := float64(with) / float64(fit)
	t.Logf("parties with a conga where it fits: %.0f %%", f*100)
	if f < 0.20 || f > 0.40 {
		t.Errorf("conga in %.0f %% of the parties where it fits, want about a third", f*100)
	}
}

// Forced in every party where it fits (1P and 2P): it starts, keeps <= its maximum, nobody stays inside a solid,
// it breaks up by itself and lets everybody go, and it never adds guests (the caps are untouched).
func TestPevCongaEveryVenue(t *testing.T) {
	probe := partyGame(t)
	defer func() { dbgPartyEvent = "" }()
	for _, two := range []bool{false, true} {
		for _, idx := range peEligible(probe, "conga") {
			started := false
			for seed := int64(1); seed <= 4 && !started; seed++ {
				peLoadSeed = seed
				g := peLoad(t, idx, two)
				peLoadSeed = 0
				g.rng = rand.New(rand.NewSource(seed*131 + int64(idx)))
				partyRng = rand.New(rand.NewSource(seed*7919 + int64(idx)))
				peForce(g, "conga")
				for s := 0; s < 30*40 && g.partyRunning("conga") == nil && len(g.pev.plan) > 0; s++ {
					peStep(g, 1.0/30, true)
				}
				a := g.partyRunning("conga")
				if a == nil {
					continue
				}
				started = true
				st := a.D.(*congaState)
				n0, gm := len(g.guests), g.goalMax()
				touched := map[*Guest]bool{}
				maxN, bad, frames := 0, 0, 0
				for s := 0; s < 30*90 && g.partyRunning("conga") != nil; s++ {
					peStep(g, 1.0/30, true)
					peTrack(g, touched)
					if len(st.line) > maxN {
						maxN = len(st.line)
					}
					for _, m := range st.line[1:] {
						frames++
						if m.next == 0 && g.lvl.SolidP(m.gu.pos.X, m.gu.pos.Y) {
							bad++
						}
					}
				}
				name := levelDefs[idx].Icon
				if g.partyRunning("conga") != nil {
					t.Errorf("%s 2P=%v: the conga never breaks up", name, two)
				}
				if maxN > st.max || st.max != congaMax && st.max != congaShortMax {
					t.Errorf("%s: %d dancers (max %d)", name, maxN, st.max)
				}
				if maxN < 3 {
					t.Errorf("%s 2P=%v: only %d dancers", name, two, maxN)
				}
				// the line walks the leader's exact steps, so a dancer brushes a corner now and then (the old
				// TestPevCongaGrowsAndStaysOnItsPath logs the same); never for long
				if bad*100 > frames*8 { // (measured: up to ~5.5 %, casino 2P; the house already 67 frames before this change)
					t.Errorf("%s: %d of %d dancer frames inside a solid", name, bad, frames)
				}
				if len(g.guests) > n0 || g.goalMax() != gm || g.aliveCount() > g.guestMax()+g.twoExtra+g.crowdExtra+5 {
					t.Errorf("%s: the conga changed the crowd: %d -> %d guests", name, n0, len(g.guests))
				}
				for k := 0; k < 30; k++ {
					peStep(g, 1.0/30, true)
				}
				if gu := peStale(g, touched); gu != nil {
					var acts []string
					for _, a := range g.pev.act {
						acts = append(acts, a.ev.ID)
					}
					t.Errorf("%s 2P=%v: a dancer is still held after the conga: %s state %d timer %.0f fun %d dur %.0f next %d taken %v, events running %v",
						name, two, guestSlug(gu), gu.state, gu.timer, gu.fun, gu.dur, gu.next, g.pevTaken(gu), acts)
				}
				if !two {
					t.Logf("%-9s up to %d dancers (max %d)", name, maxN, st.max)
				}
			}
			if !started {
				t.Errorf("%s 2P=%v: the forced conga never started", levelDefs[idx].Icon, two)
			}
		}
	}
}

// The short conga: when the roaming area is too small for the 520 px route, a line of 4-6 dancers still runs.
func TestPevCongaShortVersion(t *testing.T) {
	defer func() { dbgPartyEvent = "" }()
	ok := false
	for try := 0; try < 24 && !ok; try++ {
		seed := int64(try%12 + 1)
		bw := []float64{170, 220}[try/12]
		peLoadSeed = seed
		g := peLoad(t, 0, false) // the house
		peLoadSeed = 0
		// a copy of the definition with ONE small roaming box around the dance floor (never levelDefs itself)
		def := *g.lvl.Def
		c := congaFloor(&def)
		def.Roam = []Rect{{c.X - bw/2, c.Y - bw*0.4, bw, bw * 0.8}}
		old := g.lvl.Def
		g.lvl.Def = &def
		partyRng = rand.New(rand.NewSource(seed * 17))
		a := &partyAct{ev: partyByID("conga"), t0: g.time}
		if !congaSmall(&def) {
			t.Fatalf("a box of %.0f px should be small", bw)
		}
		if congaStart(g, a) {
			st := a.D.(*congaState)
			if st.max != congaShortMax {
				t.Fatalf("small area, but a conga of %d", st.max)
			}
			{
				ok = true
				for s := 0; s < 30*90; s++ {
					g.time += 1.0 / 30
					if congaUpdate(g, a, 1.0/30) {
						break
					}
					if len(st.line) > congaShortMax {
						t.Fatalf("short conga with %d dancers", len(st.line))
					}
				}
			}
		}
		g.lvl.Def = old
	}
	if !ok {
		t.Error("no short conga on a small roaming area")
	}
}
