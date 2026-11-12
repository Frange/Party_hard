package main

import (
	"flag"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the six events of this batch: suitcase (260), human statue (261), cook on fire (262), choir (263), street
// vendor (265) and drone (266), plus the treats (pe_treats.go). They use the helpers of pe_test.go (peStart, peStep...)
// with a real Game after loadLevel. No test depends on a fixed seed of partyPlan: the events are forced (peForce) or the
// plan is measured over thousands of parties.

var tandaIDs = []string{"suitcase", "statue", "cookfire", "choir", "vendor", "drone"}

// tandaStep: one frame of the party with the guests and, if asked, the fire and the firefighters.
func tandaStep(g *Game, dt float64, fire bool) {
	peStep(g, dt, true)
	if fire {
		g.updateFires(dt)
		g.updateFire(dt)
	}
}

func tandaUntil(g *Game, secs float64, fire bool, cond func() bool) bool {
	for s := 0; s < int(secs*30); s++ {
		if cond() {
			return true
		}
		tandaStep(g, 1.0/30, fire)
	}
	return cond()
}

func tandaFarPlayers(g *Game, from Vec) {
	for _, pl := range g.players() {
		best := pl.pos
		for i := 0; i < 400; i++ {
			p := g.lvl.NearestFree(g.lvl.RandomRoam(rand.New(rand.NewSource(int64(i)))))
			if dist(p, from) > 200 {
				best = p
				break
			}
		}
		pl.pos = best
	}
}

func TestPevTandaRegisteredSometimesAndWhere(t *testing.T) {
	g := partyGame(t)
	n := len(campaignLevels)
	list := partyList()
	outdoorOnly := map[string]bool{"statue": true, "vendor": true, "drone": true}
	for _, id := range tandaIDs {
		e := partyByID(id)
		if e == nil || !strings.Contains(list, id) {
			t.Fatalf("%s is not registered or not listed", id)
		}
		if e.Prob < 0.10 || e.Prob > 0.25 || e.AtMin < 30 || e.AtMax <= e.AtMin || e.Start == nil || e.Update == nil || e.Reactive || e.DailyOK {
			t.Errorf("%s: bad definition (chance %.2f, window %.0f-%.0f)", id, e.Prob, e.AtMin, e.AtMax)
		}
		if e.MaxRuns > 1 && e.Cooldown < 150 {
			t.Errorf("%s: comes back too soon (cooldown %.0f)", id, e.Cooldown)
		}
		in := peEligible(g, id)
		t.Logf("%-8s fits %2d of %d campaign levels (%.0f %% each): %v", id, len(in), n, e.Prob*100, in)
		if len(in) == 0 || len(in) == n {
			t.Errorf("%s should fit some levels, not all (%d of %d)", id, len(in), n)
		}
		if e.Eligible(g, partyTraits{}) {
			t.Errorf("%s fits an empty level", id)
		}
		for _, i := range in {
			tr := partyTraitsOf(levelDefs[i], i)
			if outdoorOnly[id] && !tr.outdoor {
				t.Errorf("%s in the indoor party %s", id, tr.icon)
			}
			if id == "cookfire" && !tr.cook {
				t.Errorf("cookfire in %s, which has no cooker", tr.icon)
			}
			if tr.icon == "station" && (id == "drone" || id == "choir" || id == "vendor") {
				t.Errorf("%s in the space station", id)
			}
		}
	}
	// "sometimes": over thousands of real plans each event is planned in a fraction of the parties where it fits, never
	// in all, and never where it does not fit
	r := rand.New(rand.NewSource(int64(len(partyRegistry))*31 + 7))
	planned, fits := map[string]int{}, map[string]int{}
	var hist [][]string
	for k := 0; k < 6000; k++ {
		idx := r.Intn(n)
		tr := partyTraitsOf(levelDefs[idx], idx)
		var ids []string
		for _, p := range partyPlan(r, g, tr, hist, false) {
			ids = append(ids, p.ev.ID)
			for _, id := range tandaIDs {
				if p.ev.ID == id {
					planned[id]++
					if !p.ev.Eligible(g, tr) {
						t.Fatalf("%s planned in %s, where it does not fit", id, tr.icon)
					}
				}
			}
		}
		for _, id := range tandaIDs {
			if partyByID(id).Eligible(g, tr) {
				fits[id]++
			}
		}
		hist = append(hist, ids)
		if len(hist) > partyMemory {
			hist = hist[1:]
		}
	}
	for _, id := range tandaIDs {
		f := float64(planned[id]) / math.Max(1, float64(fits[id]))
		t.Logf("%-8s planned in %.1f %% of the parties where it fits", id, f*100)
		if planned[id] == 0 || f > partyByID(id).Prob+0.02 {
			t.Errorf("%s: planned in %.1f %% of the parties where it fits (chance %.0f %%)", id, f*100, partyByID(id).Prob*100)
		}
	}
	// never in stealth nor on the island: the frame's rule
	g2 := peLoad(t, peEligible(g, "drone")[0], false)
	g2.stealth = true
	if partyBlocked(g2) == "" {
		t.Error("events in stealth")
	}
	g2.stealth = false
}

// -pevent id:stage: the stage is read by the event and the id still finds it; any debug flag freezes the progress.
func TestPevTandaDebugStageFlag(t *testing.T) {
	partyGame(t)
	dbgPartyEvent = "suitcase:open"
	if e := partyByID(dbgPartyEvent); e == nil || e.ID != "suitcase" {
		t.Fatal("-pevent suitcase:open does not find the event")
	}
	if pevDemo("suitcase") != "open" || pevDemo("vendor") != "" || !pevDebugOn("suitcase") || pevDebugOn("vendor") {
		t.Error("pevDemo / pevDebugOn read the stage wrong")
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.String("pevent", "", "")
	_ = fs.Parse([]string{"-pevent", "vendor:buy"})
	if !debugFlagsActive(fs) {
		t.Error("-pevent must freeze the progress")
	}
}

// ---- suitcase ------------------------------------------------------------------------------------------------

func suitcaseLeft(t *testing.T, two bool) (*Game, *suitcaseState) {
	t.Helper()
	g := peStart(t, "suitcase", two)
	a := g.partyRunning("suitcase")
	s := a.D.(*suitcaseState)
	if !tandaUntil(g, 25, false, func() bool { return s.phase != scWalk || g.partyRunning("suitcase") == nil }) || s.pr == nil {
		t.Fatal("the owner never set it down in this party")
	}
	return g, s
}

func TestSuitcaseOpensOnceAndGivesAPrize(t *testing.T) {
	for prize := 0; prize < scPrizes; prize++ {
		g, s := suitcaseLeft(t, false)
		if s.pr.K.NoSolid != true || g.lvl.SolidP(s.pr.X, s.pr.Y) {
			t.Fatal("the suitcase blocks the way")
		}
		s.prize = prize
		g.p.pos = g.lvl.NearestFree(Vec{s.pr.X, s.pr.Y + 10})
		g.p.item, g.p.spare, g.p.carrying = "", "", nil
		if lab, _ := g.useLabel(s.pr); lab != suitcasePrompt || g.useTextProp() != s.pr {
			t.Fatalf("no [E] Abrir next to the suitcase (label %q)", lab)
		}
		pts := g.points
		g.interact()
		if !s.pr.used || s.pr.Var != 1 || s.phase != scOpen {
			t.Fatalf("prize %d: E did not open it", prize)
		}
		switch s.prize {
		case scPoints:
			if g.points-pts != suitcasePts {
				t.Errorf("points: +%d", g.points-pts)
			}
		case scBills:
			if g.points-pts != suitcaseBills {
				t.Errorf("bills: +%d", g.points-pts)
			}
		case scItem:
			if g.p.item == "" {
				t.Error("no item from the suitcase")
			}
		case scHat:
			if g.time >= g.pevTreatsFor(g.p).hatT {
				t.Error("no party hat")
			}
		}
		// once: E again does nothing, the label is gone
		pts = g.points
		if lab, _ := g.useLabel(s.pr); lab != "" {
			t.Errorf("an open suitcase still says %q", lab)
		}
		g.interact()
		if g.points != pts {
			t.Error("the suitcase gave a second prize")
		}
		peDraw(g)
		if !tandaUntil(g, 10, false, func() bool { return g.partyRunning("suitcase") == nil }) {
			t.Error("the event did not end after the prize")
		}
	}
}

func TestSuitcaseOwnerComesBackIfNotOpened(t *testing.T) {
	peRetry(t, "suitcase", false, func(g *Game) string {
		s := g.partyRunning("suitcase").D.(*suitcaseState)
		if !tandaUntil(g, 25, false, func() bool { return s.phase != scWalk }) || s.pr == nil {
			return "never set down"
		}
		tandaFarPlayers(g, Vec{s.pr.X, s.pr.Y})
		touched := map[*Guest]bool{}
		for i := 0; i < 30*160 && g.partyRunning("suitcase") != nil; i++ {
			tandaStep(g, 1.0/30, false)
			peTrack(g, touched)
		}
		if g.partyRunning("suitcase") != nil {
			return "the event never ended"
		}
		if !s.pr.gone || s.pr.used {
			return "the unopened suitcase stayed"
		}
		if gu := peStale(g, touched); gu != nil {
			return "the owner is still held"
		}
		return ""
	})
}

// ---- statue -------------------------------------------------------------------------------------------------

func statuePosed(t *testing.T, g *Game) *statueState {
	t.Helper()
	s := g.partyRunning("statue").D.(*statueState)
	if !tandaUntil(g, 25, false, func() bool { return s.posed || g.partyRunning("statue") == nil }) || !s.posed {
		t.Fatal("the statue never took his place")
	}
	return s
}

func TestStatueIsAStillGuest(t *testing.T) {
	peRetry(t, "statue", false, func(g *Game) string {
		s := statuePosed(t, g)
		gu := s.mem.gu
		in := false
		for _, o := range g.guests {
			in = in || o == gu
		}
		if !in || gu.py != nil || gu.perf != "" {
			return "the statue is not a plain guest"
		}
		tandaFarPlayers(g, gu.pos)
		alive, total := g.aliveCount(), g.total
		start := gu.pos
		for i := 0; i < 30*40; i++ {
			tandaStep(g, 1.0/30, false)
			if !s.mem.mine() {
				return "something took him (fright)"
			}
			if dist(gu.pos, start) > 0.5 {
				t.Fatalf("the statue walked %.1f px", dist(gu.pos, start))
			}
		}
		if s.moves != 0 {
			t.Error("he moved with nobody near")
		}
		if g.aliveCount() != alive || g.total != total {
			t.Error("the statue changed the counts")
		}
		peDraw(g)
		// a player very close: sooner or later he jumps into another pose and somebody gets a fright
		g.p.pos = g.lvl.NearestFree(Vec{gu.pos.X + 12, gu.pos.Y + 4})
		pose := s.pose
		if !tandaUntil(g, 60, false, func() bool { return s.moves > 0 }) {
			return "he never moved with the player at his side"
		}
		if s.pose == pose || dist(gu.pos, start) > 0.5 {
			t.Error("the move must change the pose and never his place")
		}
		peDraw(g)
		// he can be killed like any guest: a body with his paint, a kill for the player
		kills := g.kills
		g.killGuest(gu, true)
		if g.kills != kills+1 || len(g.bodies) == 0 || g.bodies[len(g.bodies)-1].cs != s.painted {
			t.Error("killing the statue is not a plain kill with a painted body")
		}
		tandaUntil(g, 2, false, func() bool { return g.partyRunning("statue") == nil })
		if g.partyRunning("statue") != nil {
			t.Error("the event did not end with the statue dead")
		}
		return ""
	})
}

// ---- cook on fire ----------------------------------------------------------------------------------------------

func TestCookFireBurnsCallsTheFirefightersAndGoesOut(t *testing.T) {
	peRetry(t, "cookfire", false, func(g *Game) string {
		prevImm := g.dbg.immune
		t.Cleanup(func() { g.dbg.immune = prevImm; g.blaze = fireWorld{}; g.fireTruck, g.fireCrew = nil, nil })
		s := g.partyRunning("cookfire").D.(*cookState)
		tandaFarPlayers(g, s.spot)
		kills, wanted := g.kills, g.wanted
		if !tandaUntil(g, 40, true, func() bool { return s.fire != nil || g.partyRunning("cookfire") == nil }) || s.fire == nil {
			return "the oil never caught fire"
		}
		if g.liveFires() != 1 {
			t.Errorf("%d fires, want exactly one", g.liveFires())
		}
		// a second cook would not start a second fire
		if cookStart(g, &partyAct{ev: partyByID("cookfire")}) {
			t.Error("a second fire may start while one burns")
		}
		// it hurts: a player standing in it burns
		g.dbg.immune = true
		tandaUntil(g, 4, true, func() bool { return s.fire.intensity() >= 0.9 || s.fire.out })
		burnt := 0.0
		for i := 0; i < 30 && !s.fire.out; i++ {
			g.p.pos = Vec{s.fire.pos.X, s.fire.pos.Y - 2} // right in the flames (immune: it only measures)
			tandaStep(g, 1.0/30, true)
			burnt = math.Max(burnt, g.blaze.pBurn)
		}
		if burnt <= 0 {
			t.Error("standing in the fire does not burn")
		}
		tandaFarPlayers(g, s.spot)
		called := false
		if !tandaUntil(g, cookFireMax+20, true, func() bool {
			called = called || g.blaze.called || g.fireTruck != nil
			return s.fire.out
		}) {
			return "the fire never went out"
		}
		if !called {
			t.Error("nobody called the firefighters")
		}
		if g.kills != kills || g.wanted != wanted {
			t.Error("the kitchen fire counted kills or stars for the player")
		}
		tandaUntil(g, 3, true, func() bool { return g.partyRunning("cookfire") == nil })
		if g.partyRunning("cookfire") != nil {
			t.Error("the event did not end with the fire out")
		}
		return ""
	})
}

// ---- choir ---------------------------------------------------------------------------------------------------

func TestChoirSingsAndBreaksUp(t *testing.T) {
	peRetry(t, "choir", false, func(g *Game) string {
		s := g.partyRunning("choir").D.(*choirState)
		tandaFarPlayers(g, s.center)
		if !tandaUntil(g, 20, false, func() bool { return s.phase == chSing || g.partyRunning("choir") == nil }) || s.phase != chSing {
			return "they never got to sing"
		}
		if len(s.mem) < 3 || len(s.mem) > 6 {
			t.Errorf("%d singers", len(s.mem))
		}
		for _, m := range s.mem {
			if dist(m.gu.pos, s.center) > 60 || !peHeld(m.gu) { // they are drawn facing the audience (choirState.drawGuest)
				t.Error("a singer is not in the row facing the audience")
			}
		}
		peDraw(g)
		touched := map[*Guest]bool{}
		start := g.time
		for g.partyRunning("choir") != nil && g.time-start < 40 {
			tandaStep(g, 1.0/30, false)
			peTrack(g, touched)
		}
		if g.partyRunning("choir") != nil {
			return "the choir did not end"
		}
		if d := g.time - start; d < 9 || d > 21 {
			t.Errorf("they sang %.0f s", d)
		}
		if gu := peStale(g, touched); gu != nil {
			t.Error("a singer is still held")
		}
		return ""
	})
}

func TestChoirReactsToADeathAndScatters(t *testing.T) {
	peRetry(t, "choir", false, func(g *Game) string {
		s := g.partyRunning("choir").D.(*choirState)
		tandaFarPlayers(g, s.center)
		if !tandaUntil(g, 20, false, func() bool { return s.phase == chSing || g.partyRunning("choir") == nil }) || s.phase != chSing {
			return "they never got to sing"
		}
		rest := []*Guest{}
		for _, m := range s.mem[1:] {
			rest = append(rest, m.gu)
		}
		g.killGuest(s.mem[0].gu, false)
		tandaStep(g, 1.0/30, false)
		if s.phase != chReact {
			t.Fatal("no reaction to the death")
		}
		lines := 0
		for _, gu := range rest {
			for _, l := range choirLines[5:] {
				if gu.icon == l {
					lines++
				}
			}
		}
		if lines == 0 {
			t.Error("nobody said anything")
		}
		peDraw(g)
		if !tandaUntil(g, 4, false, func() bool { return g.partyRunning("choir") == nil }) {
			t.Fatal("they did not scatter")
		}
		for _, gu := range rest {
			if !gu.out() && peHeld(gu) {
				t.Error("a singer is still held after the scatter")
			}
		}
		return ""
	})
}

// ---- vendor --------------------------------------------------------------------------------------------------

func vendorAtStop(t *testing.T, g *Game) *vendorState {
	t.Helper()
	s := g.partyRunning("vendor").D.(*vendorState)
	if !tandaUntil(g, vendorLegMax+2, false, func() bool { return s.phase == vdStop || g.partyRunning("vendor") == nil }) || s.phase != vdStop {
		t.Fatal("the vendor never reached his stop")
	}
	return s
}

func TestVendorSellsOnceAndLeaves(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := peStart(t, "vendor", two)
		s := g.partyRunning("vendor").D.(*vendorState)
		gu := s.mem.gu
		if gu.out() || !gu.keep || g.total < 1 {
			t.Fatal("the vendor is not a guest of the party")
		}
		tandaFarPlayers(g, s.mid)
		s = vendorAtStop(t, g)
		total := g.total
		buyer := g.p
		if two {
			buyer = g.p2
		}
		buyer.pos = g.lvl.NearestFree(Vec{gu.pos.X + 14, gu.pos.Y + 6})
		var lab bool
		g.withPlayer(buyer, func() { lab = g.useTextProp() == nil })
		if !lab {
			continue // a prop label is closer here: E belongs to it (checked elsewhere)
		}
		if !s.canSell(g, buyer) {
			t.Fatal("no sale next to the vendor")
		}
		scr := ebiten.NewImage(screenW, screenH)
		g.partyOver(scr)
		stam := buyer.stamina
		buyer.stamina = 0
		g.withPlayer(buyer, func() { g.interact() })
		if !s.bought || s.buyer != buyer {
			t.Fatal("E did not buy")
		}
		tr := g.pevTreatsFor(buyer)
		switch s.buy {
		case 0:
			if g.pevSpeedMul(buyer) != treatSpeedK {
				t.Error("no speed")
			}
		case 1:
			if g.time >= tr.hatT {
				t.Error("no hat")
			}
		case 2:
			if g.time >= tr.balloonT {
				t.Error("no balloon")
			}
		case 3:
			if buyer.stamina < g.perkStamina {
				t.Error("no stamina")
			}
		}
		_ = stam
		if g.partyRunning("treats") == nil && s.buy != 3 {
			tandaStep(g, 1.0/30, false)
			if g.partyRunning("treats") == nil {
				t.Error("the treat is not drawn")
			}
		}
		peDraw(g)
		// once per visit
		if s.canSell(g, buyer) || s.canSell(g, g.first()) {
			t.Error("he sells twice")
		}
		if !tandaUntil(g, vendorLegMax+3, false, func() bool { return s.phase == vdGone || g.partyRunning("vendor") == nil }) {
			t.Fatal("he never left")
		}
		if !gu.gone || g.total != total-1 {
			t.Errorf("leaving: gone %v, total %d -> %d", gu.gone, total, g.total)
		}
		// the treats end by themselves
		tandaUntil(g, treatHatDur+2, false, func() bool { return !g.pevTreatsLive() })
		if g.pevTreatsLive() || g.pevSpeedMul(buyer) != 1 {
			t.Error("a treat never ended")
		}
	}
}

func TestVendorIsAKillableGuestAndRespectsTheCap(t *testing.T) {
	g := peStart(t, "vendor", false)
	s := g.partyRunning("vendor").D.(*vendorState)
	total, kills := g.total, g.kills
	g.killGuest(s.mem.gu, true)
	if g.kills != kills+1 || g.total != total {
		t.Error("the vendor does not die like a guest")
	}
	tandaStep(g, 1.0/30, false)
	if g.partyRunning("vendor") != nil {
		t.Error("the event goes on with the vendor dead")
	}
	// a full party: no vendor
	n, tot := len(g.guests), g.total
	g.total = g.goalMax() + 5
	if vendorStart(g, &partyAct{ev: partyByID("vendor")}) {
		t.Error("the vendor came to a full party")
	}
	if len(g.guests) != n {
		t.Error("a guest was added to a full party")
	}
	g.total = tot
	if capBreaches != 0 {
		t.Errorf("cap breaches: %d", capBreaches)
	}
}

// ---- drone ---------------------------------------------------------------------------------------------------

func TestDroneIsOnlyToLookAt(t *testing.T) {
	g := peStart(t, "drone", false)
	s := g.partyRunning("drone").D.(*droneState)
	guests, props, animals := len(g.guests), len(g.lvl.Props), len(g.animals)
	touched := map[*Guest]bool{}
	pointed := false
	for i := 0; i < 30*int(droneDur*2) && g.partyRunning("drone") != nil; i++ {
		tandaStep(g, 1.0/30, false)
		peTrack(g, touched)
		pointed = pointed || len(s.look) > 0
		sh := s.at(g.time - s.t0)
		if i%30 == 0 {
			g.p.pos = g.lvl.NearestFree(sh)
			if g.pevInteract(g.p) {
				t.Fatal("the drone answers to E")
			}
			peDraw(g)
		}
	}
	if g.partyRunning("drone") != nil {
		t.Fatal("the drone never left")
	}
	if len(g.guests) != guests || len(g.lvl.Props) != props || len(g.animals) != animals {
		t.Error("the drone added something to the world")
	}
	if !pointed {
		t.Log("nobody pointed at it in this party (it flew over empty places)")
	}
	if gu := peStale(g, touched); gu != nil {
		t.Error("a guest is still pointing at the sky")
	}
	if g.kills != 0 {
		t.Error("kills from the drone")
	}
}

// ---- all of them: two players, rng, corpses, draw ---------------------------------------------------------------

func TestPevTandaTwoPlayersAndRunToTheEnd(t *testing.T) {
	for _, id := range tandaIDs {
		for _, two := range []bool{false, true} {
			g := peStart(t, id, two)
			if two && g.p2 == nil {
				t.Fatalf("%s: not a two player game", id)
			}
			touched := map[*Guest]bool{}
			for s := 0; s < 30*260 && g.partyRunning(id) != nil; s++ {
				tandaStep(g, 1.0/30, id == "cookfire")
				if s%20 == 0 {
					peDraw(g)
				}
				peTrack(g, touched)
			}
			if g.partyRunning(id) != nil {
				t.Errorf("%s (2P %v): did not finish in 260 s", id, two)
			}
			if gu := peStale(g, touched); gu != nil {
				t.Errorf("%s (2P %v): a guest is still held (state %d)", id, two, gu.state)
			}
			if !g.pev.ran[id] {
				t.Errorf("%s: no mark for the newspaper", id)
			}
			if id != "cookfire" && g.kills != 0 {
				t.Errorf("%s: kills from an ambience event", id)
			}
			g.blaze, g.fireTruck, g.fireCrew = fireWorld{}, nil, nil
		}
	}
}

func TestPevTandaDoesNotMoveGameRng(t *testing.T) {
	for _, id := range []string{"suitcase", "statue", "choir", "vendor", "drone"} { // the kitchen fire uses the fire of the game
		g := peStart(t, id, false)
		g.rng = rand.New(rand.NewSource(99))
		for s := 0; s < 30*40; s++ {
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

func TestPevTandaTextsAndNews(t *testing.T) {
	loadLanguages()
	var mine []string
	for _, l := range [][]string{suitcaseLines, suitcaseSay, suitcaseCrowd, statueLines, cookLines, choirLines, vendorShouts, vendorLines, vendorBuys, droneLines} {
		mine = append(mine, l...)
	}
	for _, s := range append([]string{suitcasePrompt}, mine...) {
		if len(s) > 48 {
			t.Errorf("%q is longer than 48 bytes", s)
		}
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("no English for %q", s)
		}
	}
	for _, id := range tandaIDs {
		p, ok := paperEventPools[id]
		if !ok || len(p.pool) < 3 {
			t.Errorf("%s: no news for the newspaper", id)
			continue
		}
		if _, ok := paperBriefHeads[p.head]; !ok {
			t.Errorf("%s: head %q has no English", id, p.head)
		}
		for _, v := range p.pool {
			if !paperEventTextOK(v.es) || langDataEN[v.es] != v.en {
				t.Errorf("%s: bad news %q", id, v.es)
			}
		}
	}
}

// The "[E] Comprar (gratis)" of the vendor and the "[E] Abrir" of the suitcase in Spanish and English, 1 and 2
// players: inside the screen with 16 px of margin, whole pixels, not on any other text (the UI text audit).
func TestPevTandaPromptsUITextAudit(t *testing.T) {
	for _, id := range []string{"vendor", "suitcase"} {
		for _, two := range []bool{false, true} {
			g := peStart(t, id, two)
			setup := func(g *Game) { g.lvl.graded = true } // gradeBG reads pixels, which cannot be done without a window
			switch id {
			case "vendor":
				s := vendorAtStop(t, g)
				g.p.pos = g.lvl.NearestFree(Vec{s.mem.gu.pos.X + 14, s.mem.gu.pos.Y + 6})
				if two {
					g.p2.pos = g.lvl.NearestFree(Vec{s.mem.gu.pos.X - 14, s.mem.gu.pos.Y + 6})
				}
				s.mem.gu.icon = ""
				if !s.canSell(g, g.p) {
					continue
				}
			case "suitcase":
				s := g.partyRunning("suitcase").D.(*suitcaseState)
				if !tandaUntil(g, 25, false, func() bool { return s.phase != scWalk }) || s.pr == nil {
					continue
				}
				g.p.pos = g.lvl.NearestFree(Vec{s.pr.X, s.pr.Y + 10})
				if two { // the second player stands beside him (two players near two different things is not this test)
					g.p2.pos = g.lvl.NearestFree(Vec{s.pr.X + 8, s.pr.Y + 14})
				}
				if g.useTextProp() != s.pr {
					continue
				}
			}
			for _, gu := range g.guests {
				gu.icon = "" // the bubbles of the guests are audited apart
			}
			want := map[string]string{"vendor": vendorPrompt, "suitcase": suitcasePrompt}[id]
			for _, lang := range []string{"es", "en"} {
				rec, err := auditRender(g, auditShot{name: id, setup: setup}, lang)
				if err != "" {
					t.Fatal(err)
				}
				found := false
				for _, tx := range rec.texts {
					wantLang := want
					if lang == "en" {
						wantLang = langDataEN[want] // the English one, never the Spanish left untranslated
					}
					if strings.Contains(tx.s, "] "+wantLang) {
						found = true
						if tx.x < 16 || tx.x+tx.w > screenW-16 || tx.x != math.Round(tx.x) || tx.y != math.Round(tx.y) {
							t.Errorf("%s %s 2P=%v: %q at %.1f,%.1f w %.0f", id, lang, two, tx.s, tx.x, tx.y, tx.w)
						}
					}
				}
				if !found {
					t.Errorf("%s %s 2P=%v: the prompt is not drawn", id, lang, two)
				}
				for _, f := range auditCheck(id+" "+lang, rec, false, auditAllow, auditOutside) {
					if strings.Contains(f.msg, want) || strings.Contains(f.msg, langDataEN[want]) {
						t.Errorf("%s", f.msg)
						for _, pl := range g.players() {
							var tp *Prop
							g.withPlayer(pl, func() { tp = g.useTextProp() })
							t.Logf("player %d at %.0f,%.0f text prop %v 2P %v", pl.idx, pl.pos.X, pl.pos.Y, tp, g.is2P())
						}
					}
				}
			}
		}
	}
}
