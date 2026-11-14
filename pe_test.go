package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- helpers ---------------------------------------------------------------------------------------------

var peIDs = []string{"conga", "chairs", "drinks", "bbq", "bachelor", "mime", "juggler", "busker"}

// peLoad: the shared party game with campaign level idx loaded (1 or 2 players), every global put back afterwards.
func peLoad(t *testing.T, idx int, two bool) *Game {
	t.Helper()
	g := partyGame(t)
	// g.rng of the shared game is seeded from the process clock, and loadLevel draws the guests (who comes, the skaters on the
	// rink...) from it: the behaviour tests failed one run in fifteen. Pinned per level and per party (peLoadSeed).
	prevRng := g.rng
	t.Cleanup(func() { g.rng = prevRng })
	g.rng = rand.New(rand.NewSource(peLoadSeed*104729 + int64(idx)*977 + 5))
	prevPace, prevSc, prevTables := paceK, g.charSc, tablesOff
	t.Cleanup(func() { paceK, g.charSc, jgGiven, tablesOff = prevPace, prevSc, nil, prevTables })
	tablesOff = !peWithTables // no cloth tables unless the test wants them: the reactive tablehide would join in (pe_tables.go)
	twoPlayerMode, dbgTwoPlayers = two, two
	// The session memory (memory.go) is global: loadLevel notes the witnesses of the previous party, and any test that killed
	// before leaves its notes: the reactive "memory" event would start in this party (or not) depending on which test ran
	// before. Every party of these tests starts with a clean memory, except in the test of the memory itself (peKeepMemory).
	keep := map[int]bool{}
	if peKeepMemory {
		for k, v := range memState.seen {
			keep[k] = v
		}
	}
	// The special killer (killerreg.go, its own time-seeded dice, 1 party in 3) and the mascot suits (fursuit.go, a reactive
	// event with its own dice, 1 party in 4, that lasts the whole party) are not part of the event under test: a killer
	// killed guests in the middle of a test and the mascots stayed in g.pev.act after the tested event ended, one run in a
	// few. Neither comes in these parties unless the test asks for it (fursuitGame clears fursuitLvl to roll its suits).
	prevFur := [...]any{fursuitRng, fursuitSeed, fursuitLvl, fursuitWant, fursuitAt, fursuitDone}
	t.Cleanup(func() {
		fursuitRng, fursuitSeed, fursuitLvl = prevFur[0].(*rand.Rand), prevFur[1].(int64), prevFur[2].(*Level)
		fursuitWant, fursuitAt, fursuitDone = prevFur[3].(bool), prevFur[4].(float64), prevFur[5].(bool)
	})
	// The VIP zone dice (vipchance.go, vipshape.go) kept running from one load to the next and the style dice could be
	// seeded from the clock: the birthday party got its zone (and its shape) or not depending on what ran before, and the
	// magician test failed one run in two. Pinned per level and per party, like g.rng.
	prevVIP := [...]any{vipRng, vipSeed, vipStyleRng, vipStyleDecision, vipMisses}
	t.Cleanup(func() {
		vipRng, vipSeed, vipStyleRng = prevVIP[0].(*rand.Rand), prevVIP[1].(int64), prevVIP[2].(*rand.Rand)
		vipStyleDecision, vipMisses = prevVIP[3].(vipStyle), prevVIP[4].(int)
	})
	vipSeed = peLoadSeed*7907 + int64(idx)*131 + 3001
	vipRng = rand.New(rand.NewSource(vipSeed))
	vipStyleRng = rand.New(rand.NewSource(vipSeed*7919 + 17))
	vipStyleDecision, vipMisses = vipStyle{}, 0
	g.loadLevel(idx)
	g.kp = killerPlan{}
	fursuitLvl, fursuitWant = g.lvl, false
	// nor the weapons on the floor (presets3.go): a guest who picks one up goes on a rampage and kills two or three
	// (it broke TestFursuitPartyDanceAndBonus, "the suits changed the party", one run in a few)
	for _, w := range g.weapons {
		w.gone = true
	}
	g.weapons = nil
	memState.seen, memState.lvl, memState.pick, memState.spoke = keep, nil, nil, nil
	g.mode = ModePlay
	g.time = 0
	g.pev = pevState{ready: true}
	g.life.contestAt, g.life.pizzaAt, g.life.selfieT = 0, 0, 1e9
	g.life.mascot = nil
	return g
}

// peKeepMemory: the memory test of the guests (extras_test.go) loads the same level twice and needs what it noted in between.
var peKeepMemory bool

// peLoadSeed: the party number of peStartSeed, so each party of peRetry gets its own guests (0 elsewhere).
var peLoadSeed int64

// peEligible: the campaign levels where the event fits.
func peEligible(g *Game, id string) []int {
	var out []int
	e := partyByID(id)
	for i := range campaignLevels {
		if i < len(levelDefs) && e.Eligible(g, partyTraitsOf(levelDefs[i], i)) {
			out = append(out, i)
		}
	}
	return out
}

func peForce(g *Game, id string) {
	dbgPartyEvent = id
	g.pev = pevState{ready: true, traits: partyTraitsOf(g.lvl.Def, g.levelIdx), plan: []*partyPlanned{{ev: partyByID(id), at: g.time}}}
}

// peStep: one frame of the party (and, if asked, of the guests).
func peStep(g *Game, dt float64, guests bool) {
	g.time += dt
	g.updateParty(dt)
	if guests {
		for _, gu := range g.guests {
			g.updateGuest(gu, dt)
		}
	}
}

// peStart loads the first eligible level where the event really starts (guests with the dice of seed 1).
func peStart(t *testing.T, id string, two bool) *Game { return peStartSeed(t, id, two, 1) }

// peRetry runs a behaviour test with up to six different parties: a guest can always be frightened away by something of the
// party itself (a star's act, a fright) and then the event ends early, as it should; it must work in at least one.
func peRetry(t *testing.T, id string, two bool, body func(g *Game) string) {
	t.Helper()
	last := ""
	for seed := int64(1); seed <= 6; seed++ {
		g := peStartSeed(t, id, two, seed)
		if last = body(g); last == "" {
			return
		}
		t.Logf("%s, party %d: %s", id, seed, last)
	}
	t.Error(id + ": " + last)
}

func peStartSeed(t *testing.T, id string, two bool, seed int64) *Game {
	t.Helper()
	probe := partyGame(t)
	idxs := peEligible(probe, id)
	if len(idxs) == 0 {
		t.Fatalf("%s: no campaign level fits it", id)
	}
	// A party where the event cannot start (it declines by design when, say, fewer than 3 skaters stand on the ice at that moment)
	// is not a failure of the event: the next party with other guests is tried (up to 5 in all, each one deterministic).
	for try := int64(0); try < 5; try++ {
		for _, idx := range idxs {
			peLoadSeed = seed + 6*try
			g := peLoad(t, idx, two)
			peLoadSeed = 0
			g.rng = rand.New(rand.NewSource((seed+6*try)*131 + int64(idx)))
			partyRng = rand.New(rand.NewSource((seed+6*try)*7919 + int64(idx))) // the party's own dice vary with the party too (partyGame restores partyRng)
			peForce(g, id)
			for s := 0; s < 30*40 && g.partyRunning(id) == nil && len(g.pev.plan) > 0; s++ {
				peStep(g, 1.0/30, true)
			}
			if g.partyRunning(id) != nil {
				return g
			}
		}
	}
	t.Fatalf("%s: it never started in the %d levels that fit it %v", id, len(idxs), idxs)
	return nil
}

// peTrack notes the guests an event has taken at this moment (the stars with acts of their own use the same "fun" state).
func peTrack(g *Game, touched map[*Guest]bool) {
	for _, gu := range g.guests {
		if g.pevTaken(gu) {
			touched[gu] = true
		}
	}
}

// peStale: a guest the events took who is still held (nil = all free). A guest that another event still running owns
// right now (the reactive tablehide hid a released dancer under a cloth table: TestPevCongaEveryVenue, funfair 2P, one
// run in a few) is not stale: that event lets him go when it ends.
func peStale(g *Game, touched map[*Guest]bool) *Guest {
	for gu := range touched {
		if !gu.out() && peHeld(gu) && !g.pevTaken(gu) {
			return gu
		}
	}
	return nil
}

// peHeld: a guest is still held by an event (the marks every event leaves on the ones it owns).
func peHeld(gu *Guest) bool {
	return gu.state == gIdle && gu.timer > 1000 || gu.state == gFun && gu.fun == 18 || gu.state == gDrink && gu.dur >= 1000 ||
		gu.state == gGoto && gu.next == gIdle && gu.dur == 9999
}

func peDraw(g *Game) {
	scr := ebiten.NewImage(screenW, screenH)
	for _, it := range g.partyItems(scr, nil) {
		it.fn()
	}
	g.partyOver(scr)
}

// ---- texts and registry ----------------------------------------------------------------------------------

func TestPevTexts(t *testing.T) {
	if len(pevTextES) < 60 {
		t.Fatalf("only %d texts registered", len(pevTextES))
	}
	for _, s := range pevTextES {
		if len(s) > 48 {
			t.Errorf("%q is longer than 48 bytes", s)
		}
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("no English for %q", s)
		}
	}
}

func TestPevRegisteredAndListed(t *testing.T) {
	list := partyList()
	for _, id := range peIDs {
		e := partyByID(id)
		if e == nil {
			t.Fatalf("%s is not registered", id)
		}
		if e.Prob < 0.10 || e.Prob > 0.40 || e.Cooldown < 0 || e.AtMax <= e.AtMin {
			t.Errorf("%s: bad chance or window", id)
		}
		if !strings.Contains(list, id) {
			t.Errorf("%s is not listed in -pevents", id)
		}
	}
	for _, id := range []string{"conga", "chairs"} { // the dancing ones never run together with the contest
		if e := partyByID(id); e.Group != "dance" || !e.Big {
			t.Errorf("%s must be Big and of the dance group", id)
		}
	}
	if !partyByID("bachelor").Big {
		t.Error("the bachelor party is big")
	}
	for _, id := range []string{"mime", "juggler", "busker"} {
		if partyByID(id).Group != "street" {
			t.Errorf("%s: the street artists share a group", id)
		}
	}
}

// Eligibility: each one fits some campaign levels and not all; the outdoor ones never an indoor venue; the barbecue only
// where there is a barbecue and the sky is open.
func TestPevEligibilityByLevel(t *testing.T) {
	g := partyGame(t)
	n := len(campaignLevels)
	for _, id := range peIDs {
		e := partyByID(id)
		in := peEligible(g, id)
		t.Logf("%-8s fits %2d of %d campaign levels: %v", id, len(in), n, in)
		// (the conga fits every party but the station since 2026-11-11, owner: pe_conga_test.go checks it)
		if len(in) == 0 || id != "conga" && len(in) > n*2/3 {
			t.Errorf("%s should fit some campaign levels, not all (%d of %d)", id, len(in), n)
		}
		for _, i := range in {
			tr := partyTraitsOf(levelDefs[i], i)
			switch id {
			case "bbq":
				if !tr.bbq || !(tr.outdoor || tr.icon == "house") {
					t.Errorf("bbq fits level %d (%s) without barbecue or outdoors", i, tr.icon)
				}
			case "mime", "juggler", "busker":
				if tr.indoor && !(id == "juggler" && tr.icon == "cake") {
					t.Errorf("%s fits the indoor level %d (%s)", id, i, tr.icon)
				}
			case "drinks":
				if !tr.bar {
					t.Errorf("drinks fits level %d without a bar", i)
				}
			case "conga", "chairs":
				if !tr.dance {
					t.Errorf("%s fits level %d without a dance floor", id, i)
				}
			}
		}
		if e.Eligible(g, partyTraits{}) { // random mode (unknown venue) and no traits: none of them
			t.Errorf("%s fits an empty level", id)
		}
	}
}

// ---- running ---------------------------------------------------------------------------------------------

// Every event starts in a level that fits it, runs to its end and leaves the guests free: nobody stays held, and most of the
// party has not even been touched.
func TestPevRunsAndReleasesTheGuests(t *testing.T) {
	for _, id := range peIDs {
		id := id
		t.Run(id, func(t *testing.T) {
			g := peStart(t, id, false)
			a := g.partyRunning(id)
			if a == nil {
				t.Fatal("not running")
			}
			sawHeld := false
			touched := map[*Guest]bool{}
			for s := 0; s < 30*200 && g.partyRunning(id) != nil; s++ {
				peStep(g, 1.0/30, true)
				if s%10 == 0 {
					peDraw(g)
				}
				peTrack(g, touched)
				for gu := range touched {
					if peHeld(gu) {
						sawHeld = true
					}
				}
			}
			if g.partyRunning(id) != nil {
				t.Fatal("it did not finish in 200 s")
			}
			if !sawHeld && id != "mime" && id != "juggler" && id != "busker" && id != "conga" && id != "chairs" {
				t.Error("it never held a guest")
			}
			if gu := peStale(g, touched); gu != nil {
				t.Errorf("a guest is still held at the end (state %d timer %.0f dur %.0f fun %d)", gu.state, gu.timer, gu.dur, gu.fun)
			}
			if len(touched) > len(g.guests)/2 {
				t.Errorf("%d of %d guests were taken: too many", len(touched), len(g.guests))
			}
			if len(g.pev.act) != 0 {
				ids := ""
				for _, a := range g.pev.act {
					ids += a.ev.ID + " "
				}
				t.Errorf("%d events still running: %s", len(g.pev.act), ids)
			}
		})
	}
}

// The events never move g.rng (the dice of the rest of the party): the same value comes out as if they had not run.
func TestPevDoesNotMoveGameRng(t *testing.T) {
	for _, id := range peIDs {
		g := peStart(t, id, false)
		g.rng = rand.New(rand.NewSource(99))
		for s := 0; s < 30*45; s++ { // the guests are not updated (they use g.rng): only the event
			peStep(g, 1.0/30, false)
			if s%30 == 0 {
				peDraw(g)
			}
		}
		if got, want := g.rng.Int63(), rand.New(rand.NewSource(99)).Int63(); got != want {
			t.Errorf("%s moved g.rng", id)
		}
	}
}

// Two players, with the second one near: it runs and draws without trouble and leaves nobody held.
func TestPevTwoPlayers(t *testing.T) {
	for _, id := range peIDs {
		g := peStart(t, id, true)
		if !g.is2P() || g.p2 == nil {
			t.Fatalf("%s: not a two player game", id)
		}
		g.p2.pos = Vec{g.p.pos.X + 20, g.p.pos.Y}
		touched := map[*Guest]bool{}
		for s := 0; s < 30*100; s++ {
			peStep(g, 1.0/30, true)
			if s%15 == 0 {
				peDraw(g)
			}
			peTrack(g, touched)
		}
		if gu := peStale(g, touched); gu != nil && g.partyRunning(id) == nil {
			t.Errorf("%s: a guest is held after the end (state %d timer %.0f dur %.0f next %d fun %d)", id, gu.state, gu.timer, gu.dur, gu.next, gu.fun)
		}
	}
}

// With many corpses around the guests panic and leave the events: nothing explodes and nobody is left held.
func TestPevWithManyCorpses(t *testing.T) {
	for _, id := range peIDs {
		g := peStart(t, id, false)
		for i := 0; i < 24; i++ {
			g.killByPlayer = true
			g.spawnBody(Vec{g.p.pos.X + float64(i%6)*12 - 30, g.p.pos.Y + 20 + float64(i/6)*10}, g.guests[0].cs, DirDown)
			g.killByPlayer = false
		}
		touched := map[*Guest]bool{}
		for s := 0; s < 30*120; s++ {
			peStep(g, 1.0/30, true)
			if s%20 == 0 {
				peDraw(g)
			}
			peTrack(g, touched)
		}
		if gu := peStale(g, touched); gu != nil {
			t.Errorf("%s: a guest is held after 120 s with corpses", id)
		}
	}
}

// If the main character of an event dies it ends at once and lets the others go.
func TestPevEndsWhenItsStarIsKilled(t *testing.T) {
	g := peStart(t, "bachelor", false)
	st := g.partyRunning("bachelor").D.(*bachelorState)
	touched := map[*Guest]bool{}
	peTrack(g, touched)
	st.honor[0].gu.dead = true
	peStep(g, 1.0/30, true)
	if g.partyRunning("bachelor") != nil {
		t.Fatal("the party goes on without the honoree")
	}
	if peStale(g, touched) != nil {
		t.Error("a guest is still held")
	}
}

// ---- each event ------------------------------------------------------------------------------------------

func TestPevCongaGrowsAndStaysOnItsPath(t *testing.T) {
	peRetry(t, "conga", false, func(g *Game) string {
		st := g.partyRunning("conga").D.(*congaState)
		maxN, bad := 0, 0
		for s := 0; s < 30*80 && g.partyRunning("conga") != nil; s++ {
			peStep(g, 1.0/30, true)
			if n := len(st.line); n > maxN {
				maxN = n
			}
			if len(st.line) > congaMax {
				return "the line has too many dancers"
			}
			for _, m := range st.line[1:] {
				if m.next == 0 && g.lvl.SolidP(m.gu.pos.X, m.gu.pos.Y) {
					bad++
				}
			}
		}
		t.Logf("conga: up to %d dancers, %d frames inside a solid", maxN, bad)
		switch {
		case maxN < 3:
			return "the conga never had 3 dancers"
		case g.partyRunning("conga") != nil:
			return "the conga does not break up"
		}
		return ""
	})
}

func TestPevChairsRoundsAndALoser(t *testing.T) {
	peRetry(t, "chairs", false, func(g *Game) string {
		st := g.partyRunning("chairs").D.(*chairsState)
		n0 := len(st.mem)
		maxChairs, stops := 0, 0
		lastPhase := -1
		for s := 0; s < 30*150 && g.partyRunning("chairs") != nil; s++ {
			peStep(g, 1.0/30, true)
			if st.phase != lastPhase {
				if st.phase == chRushP {
					stops++
				}
				lastPhase = st.phase
			}
			if c := st.aliveN(); c > maxChairs {
				maxChairs = c
			}
			for _, m := range st.mem { // nobody walks through a wall
				if m.mine() && g.lvl.SolidP(m.gu.pos.X, m.gu.pos.Y) && st.phase == chMusic {
					return "a player is inside a solid"
				}
			}
			peDraw(g)
		}
		t.Logf("chairs: %d players, up to %d chairs, %d rounds", n0, maxChairs, stops)
		switch {
		case stops < 2 || stops > 3:
			return "wrong number of rounds"
		case maxChairs != n0-1:
			return "one chair fewer than players wanted"
		case st.aliveN() != 0:
			return "the chairs are still there at the end"
		case len(g.partyItems(ebiten.NewImage(screenW, screenH), nil)) != 0:
			return "something is still drawn after the end"
		}
		return ""
	})
}

func TestPevDrinksHasAWinnerAndADizzyOne(t *testing.T) {
	peRetry(t, "drinks", false, func(g *Game) string {
		st := g.partyRunning("drinks").D.(*drinksState)
		sawWin, sawCount := false, false
		for s := 0; s < 30*100 && g.partyRunning("drinks") != nil; s++ {
			peStep(g, 1.0/30, true)
			if st.phase == drCompete && (st.cnt[0] > 0 || st.cnt[1] > 0) {
				sawCount = true
			}
			if st.phase == drResult {
				sawWin = true
				if st.cnt[st.win] != drGoal || st.cnt[1-st.win] >= drGoal {
					return "bad score"
				}
				if w, l := st.c[st.win].gu, st.c[1-st.win].gu; w.fun != 7 || l.fun != 6 {
					return "bad poses"
				}
			}
			peDraw(g)
		}
		if !sawWin || !sawCount {
			return "no contest happened"
		}
		return ""
	})
}

func TestPevBBQBurnsAndLeavesTheGrillAlone(t *testing.T) {
	peRetry(t, "bbq", false, func(g *Game) string {
		st := g.partyRunning("bbq").D.(*bbqState)
		st.burn = true
		pr := st.pr
		sawFlare, sawCall := false, false
		for s := 0; s < 30*100 && g.partyRunning("bbq") != nil; s++ {
			peStep(g, 1.0/30, true)
			sawFlare = sawFlare || st.phase == bqFlare
			sawCall = sawCall || len(st.crowd.mem) > 0
			if pr.lit || pr.used || pr.gone || pr.wreck || pr.armed {
				return "the barbecue was touched"
			}
			peDraw(g)
		}
		if !sawFlare || !sawCall {
			return "no flare-up or no watchers"
		}
		// the player arms the barbecue: the event just ends
		peForce(g, "bbq")
		for s := 0; s < 30*40 && g.partyRunning("bbq") == nil; s++ {
			peStep(g, 1.0/30, true)
		}
		if a := g.partyRunning("bbq"); a != nil {
			a.D.(*bbqState).pr.armT = 3
			peStep(g, 1.0/30, true)
			a.D.(*bbqState).pr.armT = 0
			if g.partyRunning("bbq") != nil {
				return "it goes on after the player used the barbecue"
			}
		}
		return ""
	})
}

func TestPevBachelorGroupSingsAndToasts(t *testing.T) {
	peRetry(t, "bachelor", false, func(g *Game) string {
		st := g.partyRunning("bachelor").D.(*bachelorState)
		if n := len(st.gang) + 1; n < 4 || n > 6 {
			return "the group is not of 4-6 people"
		}
		sawWalk, sawToast, dared := false, false, false
		for s := 0; s < 30*150 && g.partyRunning("bachelor") != nil; s++ {
			peStep(g, 1.0/30, true)
			sawWalk = sawWalk || st.phase == baWalk
			sawToast = sawToast || st.phase == baToast && st.toastSent
			if st.phase == baWalk && !dared && st.honor[0].mine() {
				h := st.honor[0].gu
				g.p.pos = Vec{h.pos.X + 12, h.pos.Y}
				before := g.points
				h.icon = ""
				if !g.pevInteract(g.p) || h.icon == "" {
					return "E on the honoree gives no dare"
				}
				if g.points != before {
					return "the dare gives points"
				}
				dared = true
			}
			peDraw(g)
		}
		if !sawWalk || !sawToast || !dared {
			return "it did not walk, toast and dare"
		}
		return ""
	})
}

func TestPevMimeAndBuskersGetCoins(t *testing.T) {
	for _, id := range []string{"mime", "busker"} {
		id := id
		peRetry(t, id, false, func(g *Game) string {
			var show *pevShow
			switch d := g.partyRunning(id).D.(type) {
			case *mimeState:
				show = &d.show
			case *buskerState:
				show = &d.show
			}
			maxCrowd, maxTips := 0, 0
			for s := 0; s < 30*120 && g.partyRunning(id) != nil; s++ {
				peStep(g, 1.0/30, true)
				if n := len(show.crowd.mem); n > maxCrowd {
					maxCrowd = n
				}
				if show.tips > maxTips {
					maxTips = show.tips
				}
				if show.phase == showAct && s%7 == 0 && g.pevInteract(g.p) { // E on a mime or a musician does nothing
					return "it answers to E"
				}
				peDraw(g)
			}
			t.Logf("%s: audience up to %d, %d coins", id, maxCrowd, maxTips)
			if maxCrowd < 1 || maxTips < 1 {
				return "no audience or no coins"
			}
			return ""
		})
	}
}

func TestPevJugglerBallBonusOnce(t *testing.T) {
	peRetry(t, "juggler", false, func(g *Game) string {
		st := g.partyRunning("juggler").D.(*jugglerState)
		st.drops = []float64{0.5, 3} // two falls, soon
		took := 0
		for s := 0; s < 30*80 && g.partyRunning("juggler") != nil; s++ {
			peStep(g, 1.0/30, true)
			if st.jphase == jgDropped {
				g.p.pos = Vec{st.ball.X + 6, st.ball.Y}
				before := g.points
				if !g.pevInteract(g.p) {
					return "E near the fallen ball does nothing"
				}
				if g.points > before {
					took += g.points - before
				}
			}
			peDraw(g)
		}
		if took != jgBonus {
			return "the player did not get exactly one bonus"
		}
		if st.didDrop < 1 {
			return "no ball fell"
		}
		g.p.pos = Vec{-500, -500} // far from everything E does nothing
		if g.pevInteract(g.p) {
			return "E far from everything answers"
		}
		return ""
	})
}

// ---- pixels ----------------------------------------------------------------------------------------------

// Everything the events paint with plotFn stays in a box around its anchor, with exact colours (opaque or a valid
// premultiplied one): no stray pixels, nothing blurred.
func TestPevSpritesAreSharpAndInsideTheirBox(t *testing.T) {
	const W, H = 260, 220
	ax, ay := 130, 160
	for _, px := range []int{1, 2} {
		k := func(v float64) int { return int(math.Round(v * float64(px))) }
		draws := map[string]func(plot plotFn){
			"coin":      func(p plotFn) { drawCoinPix(p, ax, ay, px) },
			"notes":     func(p plotFn) { drawNotesPix(p, ax, ay-k(48), px, 1.3, 4) },
			"clap":      func(p plotFn) { drawClapPix(p, ax, ay-k(34), px, 0.4, 1) },
			"stars":     func(p plotFn) { drawStarsPix(p, ax, ay-k(41), px, 0.7) },
			"count":     func(p plotFn) { drawCountPix(p, ax, ay-k(50), px, 7) },
			"chef":      func(p plotFn) { drawChefHatPix(p, ax, ay-k(33), px) },
			"apron":     func(p plotFn) { drawApronPix(p, ax, ay, px, k) },
			"smoke":     func(p plotFn) { drawSmokePix(p, ax, ay-k(24), px, 0.9, 1) },
			"flame":     func(p plotFn) { drawFlamePix(p, ax, ay-k(24), px, 0.3, 0.5) },
			"chair0":    func(p plotFn) { drawChairPix(p, ax, ay, px, 0) },
			"chair1":    func(p plotFn) { drawChairPix(p, ax, ay, px, 1) },
			"table":     func(p plotFn) { drawTablePix(p, ax, ay, px, 2) },
			"glove":     func(p plotFn) { drawGlovePix(p, ax, ay-k(24), px) },
			"ball":      func(p plotFn) { drawBallPix(p, ax, ay-k(30), px, jgCols[0]) },
			"blush":     func(p plotFn) { drawBlushPix(p, ax, ay-k(30), px) },
			"hat":       func(p plotFn) { drawGroundHatPix(p, ax, ay+k(9), px, 9) },
			"coinpile":  func(p plotFn) { drawCoinPilePix(p, ax, ay, px, 8) },
			"mimeback":  func(p plotFn) { drawMimeBoxPix(p, ax, ay, px, k, true, 0.5) },
			"mimefront": func(p plotFn) { drawMimeBoxPix(p, ax, ay, px, k, false, 0.5) },
			"glasses":   func(p plotFn) { drawStarGlassesPix(p, ax, ay-k(28), px, pevGold) },
			"cook":      func(p plotFn) { drawCookKitPix(p, ax, ay, px, k) },
		}
		for kind := 0; kind < 4; kind++ {
			for _, fem := range []bool{false, true} {
				kind, fem := kind, fem
				draws["kit"+string(rune('0'+kind))+map[bool]string{false: "m", true: "f"}[fem]] = func(p plotFn) {
					drawBachelorKitPix(p, ax, ay, px, k, kind, fem, baPink)
				}
			}
		}
		for d := 0; d < 10; d++ {
			d := d
			draws["digit"+string(rune('0'+d))] = func(p plotFn) { drawDigitPix(p, ax, ay-k(50), px, d, clapCol) }
		}
		for tt := 0; tt < 14; tt++ {
			tt := float64(tt) * 0.5
			draws["mimehands"+string(rune('a'+int(tt*2)))] = func(p plotFn) { drawMimeFrontPix(p, ax, ay, px, k, tt) }
		}
		for name, fn := range draws {
			painted := 0
			fn(func(x, y, w, h int, c color.RGBA) {
				if c.R > c.A || c.G > c.A || c.B > c.A {
					t.Fatalf("%s: colour %v is not premultiplied", name, c)
				}
				for yy := y; yy < y+h; yy++ {
					for xx := x; xx < x+w; xx++ {
						if !image.Pt(xx, yy).In(image.Rect(0, 0, W, H)) || absI(xx-ax) > 40*px || yy-ay < -70*px || yy-ay > 20*px {
							t.Fatalf("%s (px %d): pixel %d,%d outside the box", name, px, xx, yy)
						}
						if c.A != 0 {
							painted++
						}
					}
				}
			})
			if painted == 0 && name != "smoke" && name != "flame" && !strings.HasPrefix(name, "mimehands") && name != "notes" && name != "stars" {
				t.Errorf("%s paints nothing", name)
			}
		}
		// the juggler's balls stay within reach of the hands
		for i := 0; i < 3; i++ {
			for tt := 0.0; tt < 3; tt += 0.05 {
				x, y := jugglerBall(i, tt)
				if math.Abs(x) > 14 || y > -20 || y < -62 {
					t.Fatalf("ball %d at %.2f is at %.1f,%.1f", i, tt, x, y)
				}
			}
		}
	}
}

func TestPevPureHelpers(t *testing.T) {
	var tr pevTrail
	for i := 0; i < 50; i++ {
		tr.add(Vec{float64(i) * 2, 0})
	}
	if p := tr.at(10); math.Abs(p.X-(98-10)) > 0.01 {
		t.Errorf("trail.at(10) = %v", p)
	}
	if p := tr.at(1000); p.X != 0 {
		t.Errorf("the oldest point is the limit: %v", p)
	}
	c := pevCoin{from: Vec{0, 0}, to: Vec{10, 4}, t0: 1}
	if _, _, landed := pevCoinAt(c, 1.2); landed {
		t.Error("the coin has not landed yet")
	}
	if p, h, landed := pevCoinAt(c, 1+pevCoinDur+0.01); !landed || p != c.to || h != 0 {
		t.Error("the coin lands on its target")
	}
	cs := []pevCoin{c, {t0: 5}}
	left, n := pevSettleCoins(cs, 1.8)
	if n != 1 || len(left) != 1 {
		t.Errorf("settle: %d fell, %d left", n, len(left))
	}
}
