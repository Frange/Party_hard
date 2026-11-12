package main

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// MEDAL CALIBRATION (medals.go). The thresholds come from a simulation that drives the REAL scoring code of a party
// (countKill -> killVal with its chains, tickClean with the "LIMPIO" bonus, previewFinish/finishScore with the
// completion, unseen and speed bonuses) with three play styles and a fixed seed:
//
//	expert ("max razonable"): every victim by the player, chains of 8 kills ~3.5 s apart with ~11 s pauses, seen 10 %
//	media  (a normal winner):  70-100 % of the victims by the player (the rest die by other causes), a kill every
//	                           2-14 s, seen 20-60 % of the times: no chains on purpose, no stealth, no rush
//	torpe  (a clumsy winner):  55-85 % by the player, a kill every 4-24 s, seen 45-85 %
//
// No police, killer, VIP or event points are counted (they only make medals easier).
// ---------------------------------------------------------------------------

type medalPolicy struct {
	name               string
	byPlayer, byDev    float64 // share of the kill target killed by the player (+-byDev from one party to another)
	gapMin, gapMax     float64 // seconds between kills (outside a chain)
	chain              int     // >0: kills in a chain of this length, chainGap s apart
	chainGap, chainDev float64
	seen, seenDev      float64 // chance that a kill is seen (+-seenDev from one party to another)
	guards             float64
}

var medalPolicies = []medalPolicy{
	{name: "expert", byPlayer: 1, gapMin: 8, gapMax: 14, chain: 8, chainGap: 3.5, chainDev: 0.35, seen: 0.10, guards: 0.06},
	{name: "media", byPlayer: 0.85, byDev: 0.15, gapMin: 2, gapMax: 14, seen: 0.40, seenDev: 0.2, guards: 0.06},
	{name: "torpe", byPlayer: 0.70, byDev: 0.15, gapMin: 4, gapMax: 24, seen: 0.65, seenDev: 0.2, guards: 0.06},
}

// medalSimParty plays one party of the policy with the real scoring functions and returns its final total.
func medalSimParty(g *Game, goal int, pol medalPolicy, rng *rand.Rand) int {
	g.run = runStats{}
	g.points, g.time, g.challenge, g.stealth, g.dc = 0, 0, false, false, nil
	advance := func(dt float64) {
		for t := 0.0; t < dt; t += 0.25 {
			g.time += 0.25
			g.tickClean(false)
		}
	}
	by := pol.byPlayer + (rng.Float64()*2-1)*pol.byDev
	seen := pol.seen + (rng.Float64()*2-1)*pol.seenDev
	kills := int(float64(goal)*math.Min(1, by) + 0.5)
	for k := 0; k < kills; k++ {
		switch {
		case pol.chain > 0 && k%pol.chain != 0:
			advance(pol.chainGap - pol.chainDev + rng.Float64()*2*pol.chainDev)
		default:
			advance(pol.gapMin + rng.Float64()*(pol.gapMax-pol.gapMin))
		}
		guard := rng.Float64() < pol.guards
		g.countKill(kcKnife, !guard)
		v := 30
		if guard {
			v = 50
		}
		g.points += g.killVal(v)
		if rng.Float64() < seen {
			g.run.seen[0]++
			g.run.cleanAt[0] = 0
			g.streakSeen(0) // as witnessAct: a seen kill breaks the RACHA and loses its extra (streak.go)
		}
	}
	g.finishScore()
	g.floats = g.floats[:0]
	return g.points
}

// medalSimRun: n parties of a policy, sorted totals.
func medalSimRun(t *testing.T, g *Game, goal int, pol medalPolicy, n int, seed int64) []int {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	out := make([]int, n)
	for i := range out {
		out[i] = medalSimParty(g, goal, pol, rng)
	}
	sort.Ints(out)
	return out
}

func medalShare(totals []int, min int) float64 {
	n := 0
	for _, v := range totals {
		if v >= min {
			n++
		}
	}
	return float64(n) / float64(len(totals))
}

func medalSimGame(t *testing.T) *Game {
	t.Helper()
	g := hudPlay(t, false)
	snapshotGlobals(t)
	progressFrozen = true
	prevRun, prevPts, prevTime, prevEnd := g.run, g.points, g.time, g.endTime
	t.Cleanup(func() { g.run, g.points, g.time, g.endTime = prevRun, prevPts, prevTime, prevEnd; g.floats = nil })
	return g
}

// The thresholds of medalThresholds are where the simulation says: a normal winner gets GOLD most of the times, a
// clumsy winner at least BRONZE (nearly always) and usually SILVER, and gold stays under the expert's typical total.
func TestMedalCalibrationSim(t *testing.T) {
	g := medalSimGame(t)
	for _, goal := range []int{goalCap, goalCap + crowdMax, goalCap + 25} {
		th := medalThresholds(goal)
		res := map[string][]int{}
		for i, pol := range medalPolicies {
			res[pol.name] = medalSimRun(t, g, goal, pol, 300, int64(77+i))
		}
		ex, me, to := res["expert"], res["media"], res["torpe"]
		t.Logf("goal %d: thresholds bronze %d silver %d gold %d", goal, th[0], th[1], th[2])
		for _, pol := range medalPolicies {
			v := res[pol.name]
			t.Logf("  %-6s median %5d (p10 %5d p90 %5d): gold %3.0f %%  silver+ %3.0f %%  bronze+ %3.0f %%", pol.name,
				v[len(v)/2], v[len(v)/10], v[len(v)*9/10], 100*medalShare(v, th[2]), 100*medalShare(v, th[1]), 100*medalShare(v, th[0]))
		}
		if s := medalShare(me, th[2]); s < 0.70 {
			t.Errorf("goal %d: a normal winner gets gold only %.0f %% of the times (want >= 70 %%)", goal, 100*s)
		}
		if s := medalShare(to, th[0]); s < 0.95 {
			t.Errorf("goal %d: a clumsy winner gets bronze only %.0f %% of the times", goal, 100*s)
		}
		if s := medalShare(to, th[2]); s > 0.5 {
			t.Errorf("goal %d: a clumsy winner gets gold %.0f %% of the times: gold means nothing", goal, 100*s)
		}
		if med := ex[len(ex)/2]; th[2] > med*60/100 || th[2] < med*40/100 {
			t.Errorf("goal %d: gold %d is not about half of the expert's median %d", goal, th[2], med)
		}
		if r := medalReasonableMax(goal); r < ex[len(ex)/2]*90/100 || r > ex[len(ex)/2]*110/100 {
			t.Errorf("goal %d: the closed-form reasonable max %d drifted from the simulated expert median %d", goal, r, ex[len(ex)/2])
		}
	}
}
