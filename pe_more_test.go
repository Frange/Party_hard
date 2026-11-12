package main

import (
	"math"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests of the five events of ideas 267-274: poker table, explosive cake (a trap), photographer, painter on a ladder and the
// sleeping guard. Real games (partyGame + loadLevel through peLoad), several levels and several parties; nothing depends
// on a fixed seed of the plan (other events change partyPlan): every event is forced with peForce where it fits.
// ---------------------------------------------------------------------------

var pmIDs = []string{"poker", "tarta", "fotografo", "pintor", "guardia"}

// pmStartIn: the event forced in campaign level idx (party seed s); nil if it could not start there.
func pmStartIn(t *testing.T, id string, idx int, s int64) *Game {
	t.Helper()
	peLoadSeed = s
	g := peLoad(t, idx, false)
	peLoadSeed = 0
	partyRng = rand.New(rand.NewSource(s*7919 + int64(idx)))
	prevDbg := dbgPartyEvent
	t.Cleanup(func() { dbgPartyEvent = prevDbg })
	peForce(g, id)
	for st := 0; st < 30*40 && g.partyRunning(id) == nil && len(g.pev.plan) > 0; st++ {
		peStep(g, 1.0/30, true)
	}
	if g.partyRunning(id) == nil && id != "tarta" && id != "guardia" {
		return nil
	}
	return g
}

// Where each one fits and where it can really start (the painter needs a high wall, the guard a door).
func TestPmEventsFitSomeLevelsAndStart(t *testing.T) {
	g := partyGame(t)
	n := len(campaignLevels)
	for _, id := range pmIDs {
		e := partyByID(id)
		if e == nil {
			t.Fatalf("%s not registered", id)
		}
		if e.Prob < 0.10 || e.Prob > 0.40 || e.Reactive || e.Minor {
			t.Errorf("%s: chance %.2f: it must happen sometimes, not always", id, e.Prob)
		}
		in := peEligible(g, id)
		if len(in) == 0 || len(in) > n/2 {
			t.Errorf("%s fits %d of %d campaign levels: some, not all", id, len(in), n)
		}
		if e.Eligible(g, partyTraits{}) {
			t.Errorf("%s fits an unknown level", id)
		}
		started := 0
		var where []int
		for _, idx := range in {
			gg := pmStartIn(t, id, idx, 1)
			if gg == nil {
				continue
			}
			ok := true
			switch id {
			case "tarta":
				ok = countKind(gg.lvl, "bombcake") == 1
			case "guardia":
				ok = len(sleepGuards) > 0 && pmSleeper(gg) != nil
			}
			if ok {
				started++
				where = append(where, idx)
			}
		}
		t.Logf("%-9s fits %v, starts in %v", id, in, where)
		if id == "pintor" && (started < 2 || started > 4) {
			t.Errorf("the painter must start in 2-4 levels with a facade, got %d", started)
		} else if started == 0 {
			t.Errorf("%s never starts", id)
		}
	}
}

func pmSleeper(g *Game) *Guard {
	for _, gd := range g.guards {
		if g.guardAsleep(gd) {
			return gd
		}
	}
	return nil
}

// pmRun: frames of the party with the guests, the guards and the traps (what updatePlay does for these events).
func pmRun(g *Game, secs float64, each func() bool) {
	dt := 1.0 / 30
	for s := 0; s < int(secs*30); s++ {
		peStep(g, dt, true)
		for _, gd := range g.guards {
			g.updateGuard(gd, dt)
		}
		for _, pr := range g.lvl.Props {
			if trapProp(pr) {
				g.updateTrapMine(pr, dt)
			}
		}
		if s%10 == 0 {
			peDraw(g)
		}
		if each != nil && each() {
			return
		}
	}
}

// pmAway puts the players far from p (so nothing of theirs gets in the way).
func pmAway(g *Game, p Vec) {
	g.p.pos = g.lvl.NearestFree(Vec{math.Mod(p.X+300, g.lvl.W-80) + 40, p.Y})
	if dist(g.p.pos, p) < 120 {
		g.p.pos = g.lvl.NearestFree(Vec{math.Mod(p.X+500, g.lvl.W-80) + 40, math.Mod(p.Y+200, g.lvl.H-80) + 40})
	}
}

// ---- 267 poker -----------------------------------------------------------------------------------------------

// The poker table: guests sit, are dealt cards and bet; the player has nothing to do there (no label, no E); at the end the
// table is taken away and nobody is held.
func TestPmPokerPlaysWithoutThePlayerAndCleansUp(t *testing.T) {
	peRetry(t, "poker", false, func(g *Game) string {
		st := g.partyRunning("poker").D.(*pkrState)
		tbl := st.tbl
		if countKind(g.lvl, "pokertable") != 1 || !g.lvl.SolidP(tbl.X, tbl.Y-3) {
			return "no solid table in the level"
		}
		pmAway(g, Vec{tbl.X, tbl.Y})
		touched := map[*Guest]bool{}
		dealt, checked := false, false
		pmRun(g, 200, func() bool {
			peTrack(g, touched)
			if g.partyRunning("poker") == nil {
				return true
			}
			if st.phase == pkBet {
				dealt = true
			}
			if dealt && !checked && len(st.seats) >= 2 {
				checked = true
				keep := g.p.pos
				g.p.pos = g.lvl.NearestFree(Vec{tbl.X + 26, tbl.Y + 14})
				if lab, _ := g.useLabel(tbl); lab != "" {
					t.Errorf("the poker table has a label %q: the player does nothing there", lab)
				}
				if g.pevInteract(g.p) {
					t.Error("E next to the poker table did something")
				}
				g.p.pos = keep
			}
			return false
		})
		if !dealt || st.hand < 1 {
			return "no hand was dealt"
		}
		if g.partyRunning("poker") != nil {
			return "it never ended"
		}
		if countKind(g.lvl, "pokertable") != 0 || g.lvl.SolidP(tbl.X, tbl.Y-3) {
			t.Error("the table (or its solid) is still there after the game")
		}
		if gu := peStale(g, touched); gu != nil {
			t.Errorf("a player is still held after the game (state %d)", gu.state)
		}
		return ""
	})
}

// Killing one of the players breaks the game: the others jump up (frightened, not held) and the table goes.
func TestPmPokerBreaksWhenAPlayerIsKilled(t *testing.T) {
	peRetry(t, "poker", false, func(g *Game) string {
		st := g.partyRunning("poker").D.(*pkrState)
		pmAway(g, Vec{st.tbl.X, st.tbl.Y})
		pmRun(g, 40, func() bool { return st.phase == pkBet || g.partyRunning("poker") == nil })
		if st.phase != pkBet || len(st.seats) < 3 {
			return "the game did not get to the bets"
		}
		victim := st.seats[1].m.gu
		others := []*Guest{}
		for _, s := range st.seats {
			if s.m.gu != victim {
				others = append(others, s.m.gu)
			}
		}
		g.p.pos = g.lvl.NearestFree(Vec{victim.pos.X - 12, victim.pos.Y})
		kills := g.kills
		g.killGuest(victim, true)
		if g.kills != kills+1 {
			return "the kill did not count"
		}
		pmRun(g, 0.2, nil)
		if g.partyRunning("poker") != nil {
			t.Error("the game goes on with a dead player")
		}
		for _, o := range others {
			if !o.out() && (peHeld(o) || o.state == gIdle && o.timer > 100) {
				t.Errorf("a player is still sitting after the murder (state %d)", o.state)
			}
		}
		if countKind(g.lvl, "pokertable") != 0 {
			t.Error("the table stays after the game broke")
		}
		return ""
	})
}

// ---- 268 explosive cake ------------------------------------------------------------------------------------------

// The cake is a trap: E arms it, it waits the load (candles), then blows up by itself: the close ones die (player's trap
// kills, "traps" statistic), the middle ones are stunned, the far ones are fine; the cream makes people slip; once only.
func TestPmCakeTrapDelayRadiusAndStats(t *testing.T) {
	g := peStart(t, "tarta", false)
	var cake *Prop
	for _, pr := range g.lvl.Props {
		if pr.Name == "bombcake" {
			cake = pr
		}
	}
	if cake == nil || !trapProp(cake) {
		t.Fatal("no cake trap in the level")
	}
	if lab, _ := g.useLabel(cake); lab != "Armar trampa" {
		t.Fatalf("the cake's label is %q", lab)
	}
	c := trapCentre(cake)
	// the guests: one right at it, one in the cream radius, one far; the rest sent away
	var near, mid, far *Guest
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		switch {
		case near == nil:
			near = gu
		case mid == nil:
			mid = gu
		case far == nil:
			far = gu
		default:
			gu.pos = g.lvl.NearestFree(Vec{math.Mod(c.X+400, g.lvl.W-60) + 30, math.Mod(c.Y+260, g.lvl.H-60) + 30})
			if dist(gu.pos, c) < 200 {
				gu.gone = true
			}
		}
	}
	place := func() {
		near.pos, mid.pos, far.pos = Vec{c.X + 12, c.Y + 4}, Vec{c.X - 50, c.Y + 6}, Vec{c.X, c.Y + 160}
		for _, gu := range []*Guest{near, mid, far} {
			gu.state, gu.timer, gu.path = gIdle, 5, nil
		}
	}
	for _, gd := range g.guards {
		if dist(gd.pos, c) < 200 {
			gd.dead = true
		}
	}
	g.p.pos = g.lvl.NearestFree(Vec{c.X + 6, c.Y + 50})
	place()
	trapsBefore, killsBefore := g.progress.Stats["traps"], g.run.kills[0][kcTrap]
	g.armTrap(cake, g.p)
	if lab, _ := g.useLabel(cake); lab != "" {
		t.Errorf("an armed cake still offers %q", lab)
	}
	// during the load nothing happens, even with the guest right at it
	for s := 0; s < int((trapArmTime-0.3)*30); s++ {
		g.updateTrapMine(cake, 1.0/30)
		place()
	}
	if cake.used || near.out() {
		t.Fatal("the cake went off before its delay")
	}
	for s := 0; s < 30 && !cake.used; s++ {
		g.updateTrapMine(cake, 1.0/30)
		place()
	}
	if !cake.used {
		t.Fatal("the cake never went off by itself after the load")
	}
	if !near.dead {
		t.Error("the guest at the cake survived")
	}
	if mid.dead || mid.vipStun <= 0 {
		t.Errorf("the guest in the cream radius: dead %v stun %.1f (want stunned, alive)", mid.dead, mid.vipStun)
	}
	if far.dead || far.vipStun > 0 {
		t.Error("the far guest was touched")
	}
	if g.p.dead || g.p.stunT <= 0 {
		t.Errorf("the player in the cream radius: dead %v stun %.1f (want a short stun)", g.p.dead, g.p.stunT)
	}
	if g.progress.Stats["traps"] != trapsBefore+1 || g.run.kills[0][kcTrap] != killsBefore+1 {
		t.Errorf("trap statistics %d->%d, kills %d->%d", trapsBefore, g.progress.Stats["traps"], killsBefore, g.run.kills[0][kcTrap])
	}
	// once only: a used cake offers nothing and does nothing
	if lab, _ := g.useLabel(cake); lab != "" {
		t.Errorf("an exploded cake still offers %q", lab)
	}
	creamDecal := false
	for _, d := range g.decals {
		if d.kind == 5 {
			creamDecal = true
		}
	}
	if !creamDecal {
		t.Error("no cream on the floor")
	}
	// the cream makes a walking guest slip (the reactive "nata" event)
	far.vipStun = 0
	far.pos, far.moving, far.state = Vec{cake.X + 4, cake.Y + 6}, true, gIdle
	slipped := false
	pmRun(g, 1, func() bool {
		if far.vipStun > 0 {
			slipped = true
		}
		far.moving = true
		return slipped
	})
	if !slipped {
		t.Error("nobody slips on the cream")
	}
}

// ---- 272 photographer --------------------------------------------------------------------------------------------

// The photographer lines a group up and flashes: a player in front of the camera gets the short white flash (never stunned),
// one behind it does not; he is a normal guest witness.
func TestPmPhotographerFlashDazzlesAndWitnesses(t *testing.T) {
	peRetry(t, "fotografo", false, func(g *Game) string {
		st := g.partyRunning("fotografo").D.(*photoState)
		pmAway(g, st.spot)
		pmRun(g, 30, func() bool { return st.phase == phPose || g.partyRunning("fotografo") == nil })
		if st.phase != phPose {
			return "no photo was posed"
		}
		// in front of the camera (between him and the group, a little to the side), in sight of it
		cam := Vec{st.spot.X, st.spot.Y - 26}
		front := false
		for _, o := range []Vec{{18, 22}, {-18, 22}, {26, 12}, {-26, 12}, {0, 24}, {30, 0}, {-30, 0}} {
			q := g.lvl.NearestFree(Vec{st.centre.X + o.X, st.centre.Y + o.Y})
			if ds := dist(st.spot, q); ds > 1 && (st.spot.Y-q.Y)/ds >= 0.35 && dist(cam, q) < photoDazzleR-10 && g.lvl.LOS(cam, q) {
				g.p.pos, front = q, true
				break
			}
		}
		if !front {
			return "no place in sight in front of the camera"
		}
		g.flash = 0
		pmRun(g, 3, func() bool { return st.phase == phFlash })
		if st.phase != phFlash {
			return "no flash"
		}
		if st.dazzled == 0 || g.flash <= 0 || g.flash > photoDazzle+0.01 {
			t.Errorf("the player in front of the camera was not dazzled (dazzled %d flash %.2f)", st.dazzled, g.flash)
		}
		if g.p.stunT > 0 || g.p.knockT > 0 {
			t.Error("the flash blocked the player")
		}
		// the next photo, with the player behind the photographer: no flash for him
		pmRun(g, 30, func() bool { return st.phase == phPose || g.partyRunning("fotografo") == nil })
		if st.phase == phPose {
			g.p.pos = g.lvl.NearestFree(Vec{st.spot.X, st.spot.Y + 40})
			before := st.dazzled
			pmRun(g, 3, func() bool { return st.phase == phFlash })
			if st.dazzled != before {
				t.Error("a player behind the camera was dazzled")
			}
		}
		// a witness like anybody: he sees the player kill right in front of him
		if g.partyRunning("fotografo") != nil && st.ph.mine() {
			ph := st.ph.gu
			ph.dir = DirUp
			var victim *Guest
			for _, gu := range g.guests {
				if !gu.out() && gu != ph && !st.pevHas(gu) && gu.perf == "" {
					victim = gu
					break
				}
			}
			g.p.pos = g.lvl.NearestFree(Vec{ph.pos.X, ph.pos.Y - 40})
			victim.pos = Vec{g.p.pos.X + 6, g.p.pos.Y}
			if g.lvl.LOS(ph.pos, g.p.pos) {
				g.killGuest(victim, true)
				if !ph.sawKiller {
					t.Error("the photographer did not witness a kill in front of him")
				}
			}
		}
		return ""
	})
}

// Killing the photographer ends the photos and lets the group go.
func TestPmPhotographerCanBeKilled(t *testing.T) {
	peRetry(t, "fotografo", false, func(g *Game) string {
		st := g.partyRunning("fotografo").D.(*photoState)
		pmAway(g, st.spot)
		pmRun(g, 30, func() bool { return st.phase == phPose || g.partyRunning("fotografo") == nil })
		if st.phase != phPose {
			return "no photo was posed"
		}
		grp := append([]*pevMem(nil), st.group...)
		g.p.pos = g.lvl.NearestFree(Vec{st.ph.gu.pos.X + 10, st.ph.gu.pos.Y})
		g.killGuest(st.ph.gu, true)
		pmRun(g, 0.3, nil)
		if g.partyRunning("fotografo") != nil {
			t.Error("the photos go on with the photographer dead")
		}
		for _, m := range grp {
			if !m.gu.out() && peHeld(m.gu) {
				t.Error("a posing guest is still held")
			}
		}
		return ""
	})
}

// ---- 273 painter -------------------------------------------------------------------------------------------------

// The painter up his ladder: the knife does not reach him, the ladder does ("[E] Tirar la escalera", once); he falls and dies
// (the player's kill); no label before he is up or after.
func TestPmPainterOnlyTheLadderKillsHim(t *testing.T) {
	peRetry(t, "pintor", false, func(g *Game) string {
		st := g.partyRunning("pintor").D.(*painterState)
		pmAway(g, st.base)
		if lab, _ := g.useLabel(st.ladder); lab != "" && st.phase == pnWalk {
			t.Errorf("label %q before the painter is up", lab)
		}
		if st.ladder.Foot.W > 14 || st.ladder.Foot.H > 4 || !g.lvl.SolidP(st.base.X, st.base.Y-1) {
			t.Errorf("the ladder's solid is not its feet: %+v", st.ladder.Foot)
		}
		pmRun(g, 40, func() bool { return st.phase == pnPaint || g.partyRunning("pintor") == nil })
		if st.phase != pnPaint {
			return "he never got up the ladder"
		}
		pt := st.m.gu
		if !g.pevAloft(pt) {
			t.Error("the painter is not aloft")
		}
		// the knife: right under him, nobody else around
		for _, gu := range g.guests {
			if gu != pt && !gu.out() && dist(gu.pos, pt.pos) < 60 {
				gu.pos = g.lvl.NearestFree(Vec{gu.pos.X, gu.pos.Y + 90})
			}
		}
		g.p.pos = g.lvl.NearestFree(Vec{pt.pos.X + 8, pt.pos.Y + 6})
		g.p.cooldown = 0
		g.stab()
		if pt.out() {
			t.Fatal("the knife reached the painter up the ladder")
		}
		// far from the ladder E does nothing; next to it, it knocks it over
		keep := g.p.pos
		g.p.pos = g.lvl.NearestFree(Vec{st.base.X + 80, st.base.Y + 40})
		if g.pevInteract(g.p) {
			t.Error("E far from the ladder knocked it over")
		}
		g.p.pos = keep
		if distRect(g.p.pos, st.ladder.Foot) >= pntrReach {
			g.p.pos = g.lvl.NearestFree(Vec{st.base.X + 12, st.base.Y + 6})
		}
		if lab, _ := g.useLabel(st.ladder); lab != pntrLabel {
			t.Errorf("the ladder's label is %q", lab)
		}
		kills := g.kills
		if !g.pevInteract(g.p) || st.phase != pnFall {
			t.Fatal("E next to the ladder did not knock it over")
		}
		if g.pevInteract(g.p) {
			t.Error("the ladder can be knocked over twice")
		}
		pmRun(g, 1.2, nil)
		if !pt.dead || g.kills != kills+1 {
			t.Errorf("the painter did not die of the fall as the player's kill (dead %v kills %d->%d)", pt.dead, kills, g.kills)
		}
		if countKind(g.lvl, "paintladder") != 0 || countKind(g.lvl, "ladderdown") != 1 {
			t.Error("the ladder is not lying on the floor")
		}
		paint := false
		for _, d := range g.decals {
			if d.kind >= 6 {
				paint = true
			}
		}
		if !paint {
			t.Error("no paint on the floor")
		}
		return ""
	})
}

// Left alone he paints, climbs down, goes back to the party and the ladder is taken away; he counts as a guest all along.
func TestPmPainterClimbsDownAndCounts(t *testing.T) {
	peRetry(t, "pintor", false, func(g *Game) string {
		st := g.partyRunning("pintor").D.(*painterState)
		pmAway(g, st.base)
		pt := st.m.gu
		counted := func() bool { // he is one of the guests of the count (aliveCount, the BAJAS x/N)
			for _, gu := range g.guests {
				if gu == pt {
					return !gu.gone && gu.py == nil
				}
			}
			return false
		}
		pmRun(g, 200, func() bool {
			if !pt.out() && !counted() {
				t.Error("the painter is not counted as a guest")
				return true
			}
			return g.partyRunning("pintor") == nil
		})
		if g.partyRunning("pintor") != nil {
			return "he never finished"
		}
		if st.m.gu.out() {
			return "he died of something else"
		}
		if peHeld(st.m.gu) || countKind(g.lvl, "paintladder") != 0 {
			t.Error("the painter is still held or the ladder stayed")
		}
		return ""
	})
}

// ---- 274 sleeping guard ------------------------------------------------------------------------------------------

func pmGuard(t *testing.T) (*Game, *Guard) {
	t.Helper()
	g := peStart(t, "guardia", false)
	gd := pmSleeper(g)
	if gd == nil {
		pmRun(g, 2, func() bool { return pmSleeper(g) != nil })
		gd = pmSleeper(g)
	}
	if gd == nil {
		t.Fatal("no sleeping guard")
	}
	for _, gu := range g.guests { // nobody around him (a kill of somebody else must not wake him by chance)
		if !gu.out() && dist(gu.pos, gd.pos) < 160 {
			gu.pos = g.lvl.NearestFree(Vec{math.Mod(gd.pos.X+400, g.lvl.W-60) + 30, math.Mod(gd.pos.Y+250, g.lvl.H-60) + 30})
			gu.path = nil
		}
	}
	return g, gd
}

// Walking past him slowly does not wake him; running past does: "¡Eh!" and he reacts like a guard who saw you.
func TestPmSleepingGuardWalkPastOrRun(t *testing.T) {
	g, gd := pmGuard(t)
	if len(g.guards) == 0 || g.total <= 0 {
		t.Fatal("bad game")
	}
	// walk slowly right past him
	for s := 0; s < 90; s++ {
		g.p.pos = g.lvl.NearestFree(Vec{gd.pos.X - 30 + float64(s)*0.7, gd.pos.Y + 14})
		g.p.moving, g.p.running = true, false
		g.updateGuard(gd, 1.0/30)
	}
	if !g.guardAsleep(gd) {
		t.Fatal("walking past slowly woke him")
	}
	// asleep he sees nothing: a kill in front of him far enough is not seen by him
	gd.sight = 0
	g.witnessAct(true)
	if gd.sight != 0 {
		t.Error("a sleeping guard saw something")
	}
	// run past
	g.p.pos = g.lvl.NearestFree(Vec{gd.pos.X + 30, gd.pos.Y + 16})
	g.p.moving, g.p.running = true, true
	g.updateGuard(gd, 1.0/30)
	if g.guardAsleep(gd) {
		t.Fatal("running right past him did not wake him")
	}
	ok := false
	for _, l := range sleepWakeLines {
		if gd.icon == l {
			ok = true
		}
	}
	if !ok || gd.state != gdReact && gd.state != gdChase {
		t.Errorf("he woke without the alarm: icon %q state %d", gd.icon, gd.state)
	}
	if sleepGuards[gd].why != "run" {
		t.Errorf("woke because of %q", sleepGuards[gd].why)
	}
}

// A kill close to him wakes him and he goes to the body to call the police, as any guard who finds a body.
func TestPmSleepingGuardWakesOnAKillNearby(t *testing.T) {
	g, gd := pmGuard(t)
	var v *Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" {
			v = gu
			break
		}
	}
	v.pos = g.lvl.NearestFree(Vec{gd.pos.X + 60, gd.pos.Y + 30})
	g.p.pos = g.lvl.NearestFree(Vec{v.pos.X + 10, v.pos.Y})
	g.killGuest(v, true)
	g.updateGuard(gd, 1.0/30)
	if g.guardAsleep(gd) {
		t.Fatal("a murder 60 px away did not wake him")
	}
	if sleepGuards[gd].why != "kill" || gd.state != gdInvestigate && gd.state != gdReact && gd.state != gdChase {
		t.Errorf("woke by %q, state %d", sleepGuards[gd].why, gd.state)
	}
}

// He can be killed asleep (a guard's kill) and then he wakes nobody.
func TestPmSleepingGuardCanBeKilledAsleep(t *testing.T) {
	g, gd := pmGuard(t)
	before := g.progress.Stats["guards"]
	for _, gu := range g.guests {
		if !gu.out() && dist(gu.pos, gd.pos) < 40 {
			gu.pos.Y += 120
		}
	}
	g.p.pos = g.lvl.NearestFree(Vec{gd.pos.X + 10, gd.pos.Y + 4})
	g.p.cooldown = 0
	g.stab()
	if !gd.dead || g.progress.Stats["guards"] != before+1 {
		t.Fatalf("the sleeping guard could not be killed (dead %v)", gd.dead)
	}
	if g.guardAsleep(gd) {
		t.Error("a dead guard still counts as asleep")
	}
}

// ---- all of them -------------------------------------------------------------------------------------------------

// Two players: they run, draw and leave nobody held; the count of the party (caps) is only touched by the guard, by one, and
// only when there is room.
func TestPmTwoPlayersAndCaps(t *testing.T) {
	for _, id := range pmIDs {
		g := peStart(t, id, true)
		if !g.is2P() {
			t.Fatalf("%s: not two players", id)
		}
		g.p2.pos = Vec{g.p.pos.X + 20, g.p.pos.Y}
		touched := map[*Guest]bool{}
		guests, breaches := len(g.guests), capBreaches
		pmRun(g, 90, func() bool { peTrack(g, touched); return false })
		if len(g.guests) != guests {
			t.Errorf("%s added guests (%d -> %d)", id, guests, len(g.guests))
		}
		if gu := peStale(g, touched); gu != nil && g.partyRunning(id) == nil {
			t.Errorf("%s: a guest is held after the end", id)
		}
		if capBreaches != breaches {
			t.Errorf("%s: the guest cap was broken", id)
		}
	}
}

// They never move g.rng while nobody does anything (the dice of the rest of the party).
func TestPmDoesNotMoveGameRng(t *testing.T) {
	for _, id := range []string{"poker", "fotografo", "pintor", "guardia"} {
		g := peStart(t, id, false)
		pmAway(g, g.lvl.Def.Start)
		g.rng = rand.New(rand.NewSource(99))
		for s := 0; s < 30*45; s++ { // guests not updated (they use g.rng): only the events
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

// Every text of these events has its English and fits a bubble; the prompt of the ladder too.
func TestPmTextsBothLanguages(t *testing.T) {
	for _, s := range []string{pntrLabel, pntrTooHigh, "¡La tarta ha explotado!", "Te ha explotado una tarta en la cara", "Armar trampa"} {
		en, ok := langDataEN[s]
		if !ok || en == "" {
			t.Errorf("no English for %q", s)
		}
		if len(s) > 48 || len(en) > 48 {
			t.Errorf("%q / %q longer than 48 bytes", s, en)
		}
	}
	all := append(append(append(append(append([]string{}, pkrBetLines...), cakeLines...), photoCheese...), pntrLines...), sleepWakeLines...)
	for _, s := range all {
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("no English for %q", s)
		}
	}
}
