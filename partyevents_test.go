package main

import (
	"image"
	"image/color"
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// partyGame: the shared game in a party of level 0, with every global of the frame restored afterwards.
func partyGame(t *testing.T) *Game {
	t.Helper()
	g := auditGame(t)
	prevReg, prevHist, prevSeed, prevRng, prevDbg := partyRegistry, partyHist, partySeed, partyRng, dbgPartyEvent
	prevFast, prevSnd, prevProg := fastEv, g.snd, progressFrozen
	prevMem := memState // the session memory of memory.go is global: what a test leaves in it never reaches the next one
	// loadLevel rolls time-seeded dice (snakes, tank, venue crowds, stash, disco, dance bother, life): the same level then differs
	// from run to run (a stash or a crowd on the spot an event wants) and the behaviour tests failed one run in ten. Pinned here.
	prevDice := [...]any{lifeRng, lifeSeed, snakeRng, snakeSeed, tankRng, tankSeed, crowdRng, crowdSeed, danceRng, danceSeed, stashSeed, discoSeed}
	lifeRng, lifeSeed, snakeRng, snakeSeed, tankRng, tankSeed = nil, 1001, nil, 1002, nil, 1003
	crowdRng, crowdSeed, danceRng, danceSeed, stashSeed, discoSeed = nil, 1004, nil, 1005, 1006, 1007
	t.Cleanup(func() {
		lifeRng, lifeSeed, snakeRng, snakeSeed = prevDice[0].(*rand.Rand), prevDice[1].(int64), prevDice[2].(*rand.Rand), prevDice[3].(int64)
		tankRng, tankSeed, crowdRng, crowdSeed = prevDice[4].(*rand.Rand), prevDice[5].(int64), prevDice[6].(*rand.Rand), prevDice[7].(int64)
		danceRng, danceSeed, stashSeed, discoSeed = prevDice[8].(*rand.Rand), prevDice[9].(int64), prevDice[10].(int64), prevDice[11].(int64)
		memState = prevMem
		partyRegistry, partyHist, partySeed, partyRng, dbgPartyEvent = prevReg, prevHist, prevSeed, prevRng, prevDbg
		fastEv, g.snd, progressFrozen = prevFast, prevSnd, prevProg
		g.pev, g.stealth, g.dc, g.blk = pevState{}, false, nil, blackoutState{}
		g.levelIdx = 0
	})
	// the drop-repeats memory of the messages (bubbles.go) runs on the wall clock: what an earlier test (or an
	// earlier -count run) said a few seconds ago was dropped here, and the parties of these tests went another way
	prevSay := sayState
	sayState.mem, sayState.last, sayState.lastAt, sayState.shownAt, sayState.queue = nil, "", 0, 0, nil
	prevClock := sayClock
	sayClock = func() float64 { return g.time } // the game clock: a slow or busy machine no longer changes the party
	t.Cleanup(func() { sayState, sayClock = prevSay, prevClock })
	partyPinDice(t)
	progressFrozen, fastEv, dbgPartyEvent = true, false, ""
	partyRegistry, partyHist, partySeed, partyRng = append([]*partyEvent(nil), partyRegistry...), nil, 77, nil
	auditPlay(g, false)
	g.snd = &Sounds{muted: true}
	partyHist = nil // loadLevel rolled a party with the real registry
	return g
}

func fakeEvent(id, group string, big bool, prob float64, prio int) *partyEvent {
	return &partyEvent{ID: id, Name: id, Group: group, Big: big, Prob: prob, AtMin: 10, AtMax: 100, Dur: 12, Prio: prio,
		Eligible: func(g *Game, t partyTraits) bool { return true },
		Start:    func(g *Game, a *partyAct) bool { return true },
		Update:   func(g *Game, a *partyAct, dt float64) bool { return g.time-a.t0 > 12 }}
}

func TestPartyRegistryIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range partyEvents() {
		if e.ID == "" || e.Name == "" || seen[e.ID] {
			t.Fatalf("bad or duplicated id %q", e.ID)
		}
		seen[e.ID] = true
		if e.External != nil || e.Reactive {
			continue
		}
		if e.Minor { // the small frequent ones (a fist fight): up to 95 %, never Big
			if e.Prob < 0.10 || e.Prob > 0.95 || e.Big {
				t.Errorf("%s: minor event with chance %.2f or Big", e.ID, e.Prob)
			}
		} else if e.Prob < 0.10 || e.Prob > 0.40 {
			t.Errorf("%s: chance %.2f outside 10-40 %%", e.ID, e.Prob)
		}
		if e.AtMin < 0 || e.AtMax <= e.AtMin || e.Dur <= 0 || e.Eligible == nil || e.Start == nil || e.Update == nil {
			t.Errorf("%s: incomplete definition", e.ID)
		}
	}
	for _, id := range []string{"toast", "emergency", "selfie", "contest", "pizza"} {
		if !seen[id] || !strings.Contains(partyList(), id) {
			t.Errorf("%s is not registered or not listed", id)
		}
	}
}

func TestPartyPlanDeterministicAndFast(t *testing.T) {
	g := partyGame(t)
	partyRegistry = []*partyEvent{fakeEvent("a", "", false, 0.4, 1), fakeEvent("b", "", false, 0.4, 1), fakeEvent("c", "", false, 0.4, 1)}
	for s := int64(1); s < 50; s++ {
		p1 := partyPlan(rand.New(rand.NewSource(s)), g, partyTraits{}, nil, false)
		p2 := partyPlan(rand.New(rand.NewSource(s)), g, partyTraits{}, nil, false)
		pf := partyPlan(rand.New(rand.NewSource(s)), g, partyTraits{}, nil, true)
		if len(p1) != len(p2) || len(p1) != len(pf) {
			t.Fatalf("seed %d: different plans", s)
		}
		for i := range p1 {
			if p1[i].ev != p2[i].ev || p1[i].at != p2[i].at {
				t.Fatalf("seed %d: not deterministic", s)
			}
		}
		for _, p := range pf {
			if p.at < 4 || p.at > 30 {
				t.Errorf("fast plan out of range: %.1f", p.at)
			}
		}
	}
}

// 4000 simulated parties: nobody happens in all of them, each frequency is near its chance (a bit lower because of the
// memory of 2 parties), never twice in a row, never more than 3 per party, and same-group events never overlap.
func TestPartyFrequencyMemoryAndExclusion(t *testing.T) {
	g := partyGame(t)
	probs := map[string]float64{"e1": 0.10, "e2": 0.15, "e3": 0.20, "e4": 0.25, "e5": 0.30, "e6": 0.35, "e7": 0.40}
	partyRegistry = nil
	i := 0
	for id, p := range probs {
		grp := ""
		if i%2 == 0 {
			grp = "dance"
		}
		partyRegistry = append(partyRegistry, fakeEvent(id, grp, i%3 == 0, p, i))
		i++
	}
	r := rand.New(rand.NewSource(99))
	const n = 4000
	count := map[string]int{}
	var hist [][]string
	var prev []string
	for k := 0; k < n; k++ {
		plan := partyPlan(r, g, partyTraits{}, hist, false)
		if len(plan) > partyMaxPlanned {
			t.Fatalf("%d events in one party", len(plan))
		}
		var ids []string
		for _, p := range plan {
			ids = append(ids, p.ev.ID)
			count[p.ev.ID]++
			for _, q := range prev {
				if q == p.ev.ID {
					t.Fatalf("%s two parties in a row", q)
				}
			}
		}
		for a := range plan {
			for b := range plan {
				pa, pb := plan[a], plan[b]
				if a >= b || !(pa.ev.Group != "" && pa.ev.Group == pb.ev.Group || pa.ev.Big && pb.ev.Big) {
					continue
				}
				lo, hi := pa, pb
				if pb.at < pa.at {
					lo, hi = pb, pa
				}
				if hi.at < lo.at+lo.ev.Dur {
					t.Fatalf("%s and %s overlap in time", pa.ev.ID, pb.ev.ID)
				}
			}
		}
		prev = ids
		hist = append(hist, ids)
		if len(hist) > partyMemory {
			hist = hist[1:]
		}
	}
	for id, p := range probs {
		c := count[id]
		if c >= n {
			t.Errorf("%s happens in every party", id)
		}
		f := float64(c) / n
		if f > p*1.1+0.01 || f < p*0.35 {
			t.Errorf("%s: frequency %.3f for chance %.2f", id, f, p)
		}
		t.Logf("%s chance %.2f -> %.3f", id, p, f)
	}
}

func TestPartyDoesNotMoveGameRng(t *testing.T) {
	g := partyGame(t)
	g.rng = rand.New(rand.NewSource(5))
	ref := rand.New(rand.NewSource(5))
	for k := 0; k < 30; k++ {
		g.partyReset()
		for s := 0; s < 40; s++ {
			g.time += 5
			g.updateParty(0.1)
		}
	}
	for k := 0; k < 5; k++ {
		if g.rng.Int63() != ref.Int63() {
			t.Fatal("the frame moved g.rng")
		}
	}
}

func TestPartyRuntimeExclusionAndMaxima(t *testing.T) {
	g := partyGame(t)
	a, b := fakeEvent("ga", "dance", false, 0.4, 1), fakeEvent("gb", "dance", false, 0.4, 1)
	c, d := fakeEvent("bc", "", true, 0.4, 1), fakeEvent("bd", "", true, 0.4, 1)
	e := fakeEvent("ee", "", false, 0.4, 1)
	partyRegistry = []*partyEvent{a, b, c, d, e}
	g.pev = pevState{ready: true}
	for _, ev := range partyRegistry {
		g.pev.plan = append(g.pev.plan, &partyPlanned{ev: ev, at: 1})
	}
	g.life.contest = nil
	started := map[string]bool{}
	for s := 0; s < 400; s++ {
		g.time = float64(s) * 0.5
		g.updateParty(0.5)
		groups, bigs := 0, 0
		for _, ac := range g.pev.act {
			started[ac.ev.ID] = true
			if ac.ev.Group == "dance" {
				groups++
			}
			if ac.ev.Big {
				bigs++
			}
		}
		if groups > 1 || bigs > 1 || len(g.pev.act) > partyMaxActive {
			t.Fatalf("t=%.1f: dance=%d big=%d active=%d", g.time, groups, bigs, len(g.pev.act))
		}
	}
	if len(started) != 5 {
		t.Errorf("they should all get their turn eventually: %v", started)
	}
	// the dance contest of partylife.go counts as a running Big/dance group
	g.pev = pevState{ready: true, plan: []*partyPlanned{{ev: a, at: 0}, {ev: c, at: 0}}}
	g.life.contest = &danceContest{}
	t.Cleanup(func() { g.life.contest = nil })
	g.time = 1
	g.updateParty(0.1)
	if len(g.pev.act) != 0 {
		t.Error("nothing of the frame starts over the contest")
	}
}

func TestPartyBlockedPlaces(t *testing.T) {
	g := partyGame(t)
	partyRegistry = []*partyEvent{fakeEvent("x", "", false, 0.4, 1)}
	partyRegistry[0].Prob = 1.01 // sure, to see only the blocks
	g.partyReset()
	if len(g.pev.plan) != 1 {
		t.Fatal("a sure event must be planned in a normal party")
	}
	partyHist = nil
	g.stealth = true
	g.partyReset()
	if len(g.pev.plan) != 0 || g.partyActive() {
		t.Error("never in STEALTH")
	}
	g.stealth = false
	idx := g.levelIdx
	g.levelIdx = specialJurassic
	g.partyReset()
	if len(g.pev.plan) != 0 {
		t.Error("never on the Jurassic island")
	}
	g.levelIdx = idx
	g.dc = &dailyChallenge{}
	partyHist = nil
	g.partyReset()
	if len(g.pev.plan) != 0 {
		t.Error("RETO DE HOY: only DailyOK events")
	}
	partyRegistry[0].DailyOK = true
	partyHist = nil
	g.partyReset()
	if len(g.pev.plan) != 1 {
		t.Error("a DailyOK event may run in the challenge")
	}
}

func TestPartyForcedEventAndHistory(t *testing.T) {
	g := partyGame(t)
	dbgPartyEvent = "toast"
	g.partyReset()
	if len(g.pev.plan) != 1 || g.pev.plan[0].ev.ID != "toast" || g.pev.plan[0].at != 3 {
		t.Fatalf("-pevent toast: plan %+v", g.pev.plan)
	}
	if len(partyHist) != 0 {
		t.Error("a forced event does not count for the memory")
	}
	dbgPartyEvent = ""
	for k := 0; k < 6; k++ {
		g.partyReset()
	}
	if len(partyHist) != partyMemory {
		t.Errorf("memory has %d parties, want %d", len(partyHist), partyMemory)
	}
	// external adapters are never planned
	for k := 0; k < 300; k++ {
		g.partyReset()
		for _, p := range g.pev.plan {
			if p.ev.External != nil || p.ev.Reactive {
				t.Fatalf("%s must not be scheduled by the frame", p.ev.ID)
			}
		}
	}
}

// Eligibility with the real levels of the campaign: traits are filled, and the toast is not forced everywhere by
// the level (the chance does that), and the adapters leave the dice of partylife.go alone.
func TestPartyEligibilityCampaignLevels(t *testing.T) {
	g := partyGame(t)
	n := len(campaignLevels)
	if n < 21 || len(levelDefs) < n {
		t.Fatalf("campaign levels %d, defs %d", n, len(levelDefs))
	}
	toast := partyByID("toast")
	in, out := 0, 0
	for i := 0; i < n; i++ {
		tr := partyTraitsOf(levelDefs[i], i)
		if tr.icon == "" || tr.indoor == tr.outdoor {
			t.Errorf("level %d (%s): venue not classified", i, levelDefs[i].Title)
		}
		if toast.Eligible(g, tr) {
			in++
		} else {
			out++
		}
	}
	t.Logf("toast eligible in %d of %d campaign levels", in, n)
	if in == 0 || out == 0 {
		t.Errorf("the toast should fit some campaign levels and not all (%d/%d)", in, out)
	}
	em := partyByID("emergency")
	g.blk = blackoutState{}
	if em.Eligible(g, partyTraits{}) {
		t.Error("emergency lights only with a blackout")
	}
	if !em.DailyOK || !em.Reactive {
		t.Error("emergency lights follow the blackout, also in the challenge")
	}
	if partyTraitsOf(nil, 0).guests != 0 {
		t.Error("nil level")
	}
}

func partyScatter(g *Game, n int) []*Guest {
	var out []*Guest
	for i := 0; i < n && i < len(g.guests); i++ {
		gu := g.guests[i]
		gu.pos = Vec{g.lvl.Def.Start.X + float64(i%3)*14, g.lvl.Def.Start.Y + 70 + float64(i/3)*14}
		gu.state, gu.timer, gu.path, gu.perf, gu.dead, gu.gone = gIdle, 1, nil, "", false, false
		out = append(out, gu)
	}
	return out
}

func TestPartyToastRunsAndReleasesTheSpeaker(t *testing.T) {
	g := partyGame(t)
	gs := partyScatter(g, 6)
	if len(gs) < 6 {
		t.Skip("level 0 has fewer than 6 guests")
	}
	for _, gu := range gs {
		if !toastGuestOK(g, gu) {
			t.Fatalf("test guest is not free")
		}
	}
	dbgPartyEvent = "toast"
	g.pev = pevState{ready: true, plan: []*partyPlanned{{ev: partyByID("toast"), at: 0}}}
	g.time = 0
	sawLine, sawHeld := false, false
	var st *toastState
	for s := 0; s < 300; s++ {
		g.time = float64(s) * 0.1
		g.updateParty(0.1)
		if a := g.partyRunning("toast"); a != nil {
			st = a.D.(*toastState)
			sawHeld = sawHeld || st.sp.timer > 1000
			if toastLine(st, g.time-a.t0) != "" {
				sawLine = true
			}
		}
	}
	if st == nil || !sawHeld || !sawLine {
		t.Fatalf("toast did not run: st=%v held=%v line=%v", st != nil, sawHeld, sawLine)
	}
	if g.partyRunning("toast") != nil {
		t.Error("toast should be over")
	}
	if st.sp.timer > 1000 {
		t.Error("the speaker was not released")
	}
	// the speaker dies: it ends at once
	partyScatter(g, 6)
	g.pev = pevState{ready: true, plan: []*partyPlanned{{ev: partyByID("toast"), at: 0}}}
	g.time = 0
	g.updateParty(0.1)
	a := g.partyRunning("toast")
	if a == nil {
		t.Fatal("did not start the second time")
	}
	a.D.(*toastState).sp.dead = true
	g.updateParty(0.1)
	if g.partyRunning("toast") != nil {
		t.Error("a dead speaker ends the toast")
	}
}

func TestPartyEmergencyLightsFollowTheBlackout(t *testing.T) {
	g := partyGame(t)
	g.blk = blackoutState{planned: true, at: 1, dur: 6}
	g.time = 0
	g.updateParty(0.1)
	if g.partyRunning("emergency") != nil {
		t.Fatal("before the cut")
	}
	g.time = 2
	g.updateBlackout(0.1)
	g.updateParty(0.1)
	if g.partyRunning("emergency") == nil {
		t.Fatal("emergency lights should be on in the blackout")
	}
	scr := ebiten.NewImage(screenW, screenH)
	g.partyOver(scr)
	g.partyItems(scr, nil)
	g.blk.t = 0
	g.blk.done = true
	g.updateParty(0.1)
	if g.partyRunning("emergency") != nil {
		t.Error("emergency lights should be off when the light returns")
	}
}

// Pixel test: the crate, glass and hands use only their exact colours (nothing blurred, nothing between).
func TestPartyDemoSpritesAreSharp(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 80))
	plot := func(x, y, w, h int, c color.RGBA) {
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				if image.Pt(xx, yy).In(img.Rect) {
					img.SetRGBA(xx, yy, c)
				}
			}
		}
	}
	for _, px := range []int{1, 2} {
		drawCratePix(plot, 40, 50, px)
		drawGlassPix(plot, 20, 20, px)
	}
	ok := map[color.RGBA]bool{{}: true, crateTop: true, crateSide: true, crateDark: true, glassCol: true, wineCol: true}
	painted := 0
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			c := img.RGBAAt(x, y)
			if !ok[c] {
				t.Fatalf("pixel %d,%d has a stray colour %v", x, y, c)
			}
			if c.A != 0 {
				painted++
			}
		}
	}
	if painted < 150 {
		t.Errorf("only %d pixels painted", painted)
	}
}

// partyPinDice: the other session dice that loadLevel and the party roll (VIP zone and its pity counter, weather,
// closing countdown, costumes, wild animals, wrong suspect, bean gang and the events of v1.3.0) start again from a
// fixed seed in every party of these tests. They were seeded from the clock and kept running from one test (or one
// -count run) to the next: a VIP zone with its guards, a storm or a gang came or not depending on what ran before,
// and the behaviour tests (conga, street kids, eating contest, sack race, mascots) failed one run in a few. A test
// that already fixed a seed (rwLoad, the bull, the sack race...) keeps it.
func partyPinDice(t *testing.T) {
	t.Helper()
	type pin struct {
		rng  **rand.Rand
		seed *int64
		v    int64
	}
	pins := []pin{{&vipRng, &vipSeed, 2001}, {&weatherRng, &weatherSeed, 2002}, {&cdRng, &cdSeed, 2003},
		{&costumeRng, &costumeSeed, 2004}, {&wildRng, &wildSeed, 2005}, {&blameRng, &blameSeed, 2006},
		{&toroRng, &toroSeed, 2007}, {&heroRng, &heroSeed, 2008}, {&fetchRng, &fetchSeed, 2009},
		{&gfRng, &gfSeed, 2010}, {&sackRng, &sackSeed, 2011}, {&petRng, &petSeed, 2012},
		{&iceSkRng, &iceSkSeed, 2013}, {&rwRng, &rwSeed, 2014}, {&fursuitRng, &fursuitSeed, 2015}}
	for _, p := range pins {
		r, s := *p.rng, *p.seed
		t.Cleanup(func() { *p.rng, *p.seed = r, s })
		if s == 0 {
			*p.rng, *p.seed = nil, p.v
		}
	}
	fg, misses, sb, sl := fgRng, vipMisses, smokeBase, smokeLoads
	t.Cleanup(func() { fgRng, vipMisses, smokeBase, smokeLoads = fg, misses, sb, sl })
	fgRng, vipMisses, smokeBase, smokeLoads = rand.New(rand.NewSource(2016)), 0, 2017, 0
}
