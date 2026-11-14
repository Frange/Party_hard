package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the ambulance / police rules of 2026-11-11 (ambulance.go). Testers: "the ambulance gets
// there and goes away without doing anything; the police too". Every test runs a real party
// (loadLevel) on several levels and seeds; none depends on one lucky seed.

// ambGame: like playGame, for one or two players, with the T-Rex asleep.
func ambGame(t *testing.T, level int, seed int64, two bool) *Game {
	t.Helper()
	if len(levelDefs) < len(campaignLevels) {
		levelDefs = builtinLevels() // what newGame builds too (pirate_start_test.go does the same)
	}
	g := layerGame(t, two)
	god, rng := g.god, g.rng
	t.Cleanup(func() {
		g.god, g.rng = god, rng
		g.cops, g.veh, g.bodies = nil, nil, nil
		g.policeCalled, g.pendingCar, g.pendingUnit, g.pendingAmb = false, -1, -1, -1
		g.ambCalled, g.ambCount, g.ambTwin, g.callerGuest, g.callerGuard = false, 0, false, nil, nil
		g.wanted, g.suspect = 0, nil
		g.mode = ModeMenu
	})
	g.rng = rand.New(rand.NewSource(seed))
	g.loadLevel(level)
	g.rng = rand.New(rand.NewSource(seed))
	g.mode = ModePlay
	g.titleT = 0
	g.god = true
	g.time = 30
	// the special killer (killerreg.go, its own clock-seeded dice, 1 party in 3) and the weapons on the floor (a guest
	// who picks one up goes on a rampage) are not part of these tests: the officers ran after a killer (cSlash) and
	// their search stopped counting, TestPoliceSearch10to15sThenLeave failed one run in a few (as in peLoad)
	g.kp = killerPlan{}
	for _, w := range g.weapons {
		w.gone = true
	}
	g.weapons = nil
	if g.rex != nil {
		g.rex.state, g.rex.timer = jrSleep, 1e9
	}
	return g
}

// ambLevels: an ordinary road party, the funfair, the pirate ship, the space station and the island.
func ambLevels(t *testing.T) []int {
	return []int{0, campaignIdx("funfair"), pirateIdx, specialJurassic, len(campaignLevels) - 1}
}

// ambBodies kills n ordinary guests where they stand and has them found.
func ambBodies(g *Game, n int) []*Body {
	k := 0
	for _, gu := range g.guests {
		if k >= n {
			break
		}
		if !gu.out() && gu.perf == "" && !g.lvl.InPool(gu.pos) {
			g.killGuest(gu, false)
			k++
		}
	}
	for _, b := range g.bodies {
		b.reported, b.fallT = true, 0
	}
	return append([]*Body(nil), g.bodies...)
}

// ambWatch follows every paramedic and ambulance while the party runs and reports what broke the
// rule "a unit that gets there always works before going".
type ambWatch struct {
	last     map[*Cop]cState
	ladder   map[*Cop]int
	parked   map[*Vehicle]bool
	crew     map[*Vehicle]bool
	back     map[*Vehicle]bool
	left     int // paramedics who went back aboard
	turned   int // ambulances that turned back on the way
	carriedN int
	errs     []string
}

func newAmbWatch() *ambWatch {
	return &ambWatch{last: map[*Cop]cState{}, ladder: map[*Cop]int{}, parked: map[*Vehicle]bool{}, crew: map[*Vehicle]bool{}, back: map[*Vehicle]bool{}}
}

func (w *ambWatch) step(g *Game) {
	alive := map[*Cop]bool{}
	for _, c := range g.cops {
		alive[c] = true
		if c.medic {
			w.last[c], w.ladder[c] = c.state, c.climbPh
			if c.veh != nil {
				w.crew[c.veh] = true
			}
		}
	}
	for c, st := range w.last {
		if alive[c] {
			continue
		}
		if st == cLeave || w.ladder[c] == ladLeave { // back aboard (not killed on the way)
			w.left++
			w.carriedN += c.carried
			if c.carried == 0 && c.actT < medicMinAct {
				w.errs = append(w.errs, "a paramedic went back without having worked")
			}
		}
		delete(w.last, c)
	}
	for _, v := range g.veh {
		if !medicVeh(v) {
			continue
		}
		if v.state == 1 && v.spawned {
			w.parked[v] = true
		}
		if v.state == 2 {
			if w.parked[v] && !w.crew[v] {
				w.errs = append(w.errs, "an ambulance parked and drove away with nobody getting out")
				w.parked[v] = false
			}
			if !w.parked[v] && !w.crew[v] && !w.back[v] {
				w.turned++
				w.back[v] = true
			}
		}
	}
}

func ambRun(g *Game, w *ambWatch, secs float64, every func()) {
	for i := 0; i < int(secs*60); i++ {
		g.updatePlay(1.0 / 60)
		if g.mode != ModePlay {
			g.mode = ModePlay
		}
		w.step(g)
		if every != nil && i%60 == 0 {
			every()
		}
	}
}

// The main bug: no ambulance crew goes back without having worked, also when bodies vanish
// (simulated: now and then a body that nobody carries disappears, as a hidden one would).
func TestAmbulanceNeverLeavesWithoutActing(t *testing.T) {
	total, carried := 0, 0
	for _, lv := range ambLevels(t) {
		for seed := int64(1); seed <= 3; seed++ {
			g := ambGame(t, lv, seed, false)
			r := rand.New(rand.NewSource(seed * 7))
			ambBodies(g, 2+int(seed)*2)
			g.p.pos = g.lvl.Def.Start
			g.raisePolice(g.bodies[0].pos, false)
			w := newAmbWatch()
			ambRun(g, w, 150, func() {
				if seed == 2 && r.Intn(5) == 0 && len(g.bodies) > 0 { // seed 2: bodies vanish now and then
					b := g.bodies[r.Intn(len(g.bodies))]
					if !b.carried {
						g.removeBody(b)
					}
				}
			})
			for _, e := range w.errs {
				t.Errorf("level %d seed %d: %s", lv, seed, e)
			}
			total += w.left
			carried += w.carriedN
		}
	}
	if total == 0 || carried == 0 {
		t.Fatalf("no paramedic ever finished a job (%d left, %d bodies carried): the test saw nothing", total, carried)
	}
	t.Logf("%d paramedics went back, %d bodies taken away", total, carried)
}

// Its bodies gone while it is on its way: it takes another free one, or turns back before arriving.
func TestAmbulanceTurnsBackWhenBodiesGoneEnRoute(t *testing.T) {
	for _, lv := range ambLevels(t) {
		g := ambGame(t, lv, 4, false)
		mk := func() *Body {
			b := &Body{pos: g.lvl.RandomRoam(g.rng), reported: true, byRex: true}
			g.bodies = append(g.bodies, b)
			return b
		}
		for i := 0; i < 3; i++ {
			mk()
		}
		g.policeCalled = true
		g.maybeAmbulance()
		if !g.ambCalled || g.pendingAmb < 0 {
			t.Fatalf("level %d: 3 found bodies and no ambulance call", lv)
		}
		g.time = g.pendingAmb
		g.maybeAmbulance()
		var v *Vehicle
		for _, x := range g.veh {
			if medicVeh(x) {
				v = x
			}
		}
		if v == nil || v.state != 0 {
			t.Fatalf("level %d: no ambulance on its way", lv)
		}
		if g.ambReserved(v) != 3 {
			t.Fatalf("level %d: the ambulance must reserve its 3 bodies, got %d", lv, g.ambReserved(v))
		}
		// (1) its bodies vanish but a new free one is found: it goes for that one
		g.bodies = nil
		nb := mk()
		g.maybeAmbulance()
		if v.state != 0 || nb.resv != v {
			t.Fatalf("level %d: the ambulance must take the new body (state %d)", lv, v.state)
		}
		// (2) everything gone: it turns back before it gets there, nobody gets out
		g.bodies = nil
		w := newAmbWatch()
		ambRun(g, w, 2, nil)
		if v.state != 2 || w.crew[v] || w.parked[v] {
			t.Fatalf("level %d: with nothing to fetch it must turn back on the way (state %d parked %v)", lv, v.state, w.parked[v])
		}
		if g.ambCalled {
			t.Fatalf("level %d: nothing left on the case: the ambulance call must be over", lv)
		}
	}
}

// The police cover every body found and never carry one; they are not even reserved by them.
func TestPoliceOnlyCoverNeverCarry(t *testing.T) {
	for _, lv := range []int{0, 7, specialJurassic} {
		for seed := int64(1); seed <= 2; seed++ {
			g := ambGame(t, lv, seed, seed == 2)
			orig := ambBodies(g, 5)
			g.p.pos = g.lvl.Def.Start
			g.raisePolice(orig[0].pos, false)
			covered := map[*Body]bool{}
			police := 0
			for i := 0; i < 120*60; i++ {
				g.updatePlay(1.0 / 60)
				g.mode = ModePlay
				for _, c := range g.cops {
					if c.medic {
						continue
					}
					police++
					if c.carry != nil || c.state == cCollect || c.state == cBag || c.state == cReturn {
						t.Fatalf("level %d seed %d: an officer carries a body (state %d)", lv, seed, c.state)
					}
				}
				for _, b := range g.bodies {
					if b.bagged {
						covered[b] = true
					}
				}
			}
			if police == 0 {
				t.Fatalf("level %d seed %d: the police never came", lv, seed)
			}
			for _, b := range orig {
				if !covered[b] {
					t.Errorf("level %d seed %d: a body found was never covered", lv, seed)
				}
			}
		}
	}
}

// After covering, each officer searches 10-15 s (each towards a different place) and goes back with his crew
// when nobody is suspected.
func TestPoliceSearch10to15sThenLeave(t *testing.T) {
	for _, lv := range []int{0, 7, 12} {
		g := ambGame(t, lv, 3, false)
		for i := 0; i < 3; i++ {
			g.bodies = append(g.bodies, &Body{pos: g.lvl.RandomRoam(g.rng), reported: true, byRex: true})
		}
		orig := append([]*Body(nil), g.bodies...)
		g.raisePolice(g.bodies[0].pos, false)
		seen := map[*Cop]bool{}
		hist := map[*Cop]string{}
		leftAt := map[*Cop]float64{}
		close, pairs := 0, 0
		closed := false // a passer-by blamed and taken (wrongsuspect.go, own dice): the case closes early
		for i := 0; i < 100*60; i++ {
			g.updatePlay(1.0 / 60)
			g.mode = ModePlay
			closed = closed || g.suspect != nil || g.blame.active
			var sweep []*Cop
			for _, c := range g.cops {
				if c.medic || (!seen[c] && g.time > 75) { // only the first call (a later witness may call again)
					continue
				}
				if h := hist[c]; len(h) == 0 || h[len(h)-1] != byte('A'+int(c.state)) {
					hist[c] += string(rune('A' + int(c.state)))
				}
				if !seen[c] {
					seen[c] = true
					if c.maxSearch < 10 || c.maxSearch > 15 {
						t.Fatalf("level %d: search of %.1fs, want 10-15", lv, c.maxSearch)
					}
				}
				if c.state == cLeave {
					if _, ok := leftAt[c]; !ok {
						leftAt[c] = g.time
						if closed || g.caseHasSuspect() || strings.ContainsAny(hist[c], "JM") { // a suspect taken: the case closed
							continue
						}
						// the crew goes back complete (copcrew.go, 2026-11-12): an officer who is done may go on searching
						// until his last mate is done, so the 10-15 s hold for the last one of the crew, and they all
						// leave in the same frame
						last := c.searchT
						for _, o := range g.cops {
							if o.veh == c.veh && !o.medic {
								if o.state != cLeave {
									t.Fatalf("level %d: an officer left without his mate (mate state %d)", lv, o.state)
								}
								last = math.Min(last, o.searchT)
							}
						}
						if c.searchT < 10 || last > 15.2 {
							t.Fatalf("level %d: an officer went back after %.1fs of search (last of the crew %.1f, max %.1f, onScene %.1f, kind %d, t %.1f, states %s, resp %d)", lv, c.searchT, last, c.maxSearch, c.onSceneT, c.kind, g.time, hist[c], g.respKind)
						}
					}
				}
				if c.state == cSweep && len(c.path) > 0 {
					sweep = append(sweep, c)
				}
			}
			for a := 0; a < len(sweep); a++ {
				for b := a + 1; b < len(sweep); b++ {
					pa, pb := sweep[a].path[len(sweep[a].path)-1], sweep[b].path[len(sweep[b].path)-1]
					pairs++
					if dist(pa, pb) < 24 {
						close++
					}
				}
			}
		}
		if len(seen) == 0 {
			t.Fatalf("level %d: the police never came", lv)
		}
		if len(leftAt) != len(seen) {
			st := ""
			for c := range seen {
				if _, ok := leftAt[c]; !ok {
					st += fmt.Sprintf(" [state %d search %.1f/%.1f scene %.1f in %v kind %d suspect %v]", c.state, c.searchT, c.maxSearch, c.onSceneT, containsCop(g.cops, c), c.kind, g.caseHasSuspect())
				}
			}
			t.Fatalf("level %d: %d of %d officers never went back:%s", lv, len(seen)-len(leftAt), len(seen), st)
		}
		for _, b := range orig {
			if !b.bagged && !b.carried && containsBody(g.bodies, b) {
				t.Fatalf("level %d: a body was left uncovered %s called %v amb %v cops %d hist %v", lv, fmtB(b), g.policeCalled, g.ambCalled, len(g.cops), hist)
			}
		}
		if pairs > 0 && close*5 > pairs {
			t.Fatalf("level %d: the officers search the same spot (%d of %d)", lv, close, pairs)
		}
	}
}

// Cleaners, the stray dog and the player leave alone a body an ambulance is coming for.
func TestCleanersDogSkipReservedBody(t *testing.T) {
	g := ambGame(t, 0, 5, false)
	b := &Body{pos: g.lvl.RandomRoam(g.rng), reported: true}
	g.bodies = append(g.bodies, b)
	if !cleanBodyOK(g, b) || !dogBodyOK(g, b) || g.bodyHeld(b) {
		t.Fatal("setup: a lying body is fair game")
	}
	v := &Vehicle{kind: 1}
	g.veh = append(g.veh, v)
	b.resv = v
	if cleanBodyOK(g, b) || dogBodyOK(g, b) || !g.bodyHeld(b) {
		t.Fatal("a reserved body must be left for the ambulance")
	}
	// the player cannot pick it up either
	g.p.pos = b.pos
	g.p.carrying = nil
	g.interact()
	if g.p.carrying == b || b.carried {
		t.Fatal("the player picked up a body the ambulance is coming for")
	}
	// the ambulance goes away: the reservation goes with it
	v.state = 2
	if g.bodyHeld(b) || !cleanBodyOK(g, b) || !dogBodyOK(g, b) {
		t.Fatal("a body whose ambulance left is free again")
	}
	// a paramedic walking to it holds it too
	b.claimed = &Cop{medic: true}
	if cleanBodyOK(g, b) || dogBodyOK(g, b) {
		t.Fatal("a body a paramedic is on must be left alone")
	}
}

// 5+ free bodies: two ambulances at once, each with a crew for its share; fewer: one.
func TestTwinAmbulanceCrewMatchesFreeBodies(t *testing.T) {
	for _, tc := range []struct{ bodies, ambs int }{{2, 1}, {4, 1}, {5, 2}, {8, 2}} {
		for _, lv := range []int{0, pirateIdx, specialJurassic} {
			g := ambGame(t, lv, 6, false)
			for i := 0; i < tc.bodies; i++ {
				g.bodies = append(g.bodies, &Body{pos: g.lvl.RandomRoam(g.rng), reported: true, byRex: true})
			}
			g.policeCalled, g.ambCalled, g.pendingAmb = true, true, g.time
			g.maybeAmbulance()
			var vs []*Vehicle
			for _, v := range g.veh {
				if medicVeh(v) {
					vs = append(vs, v)
				}
			}
			if len(vs) != tc.ambs {
				t.Fatalf("level %d, %d bodies: %d ambulances, want %d", lv, tc.bodies, len(vs), tc.ambs)
			}
			sum := 0
			for _, v := range vs {
				sum += g.ambReserved(v)
			}
			if sum != tc.bodies {
				t.Fatalf("level %d: the ambulances reserve %d of %d bodies", lv, sum, tc.bodies)
			}
			want := map[*Vehicle]int{}
			for _, v := range vs {
				want[v] = clampI((g.ambReserved(v)+1)/2, 1, 3)
			}
			got := map[*Vehicle]int{}
			for i := 0; i < 30*60 && len(got) < len(vs); i++ {
				// a body that falls while it is on the way (a fight, an event of the party: one run in a few) is
				// reserved for it too and the crew at arrival grows with it (ambEnRoute / ambCrewAtArrival): the share
				// that counts is the last one before its crew gets out
				for _, v := range vs {
					if got[v] == 0 { // until its crew gets out
						want[v] = clampI((g.ambReserved(v)+1)/2, 1, 3)
					}
				}
				g.updatePlay(1.0 / 60)
				g.mode = ModePlay
				for _, v := range vs {
					if v.spawned && got[v] == 0 {
						for _, c := range g.cops {
							if c.veh == v {
								got[v]++
							}
						}
					}
				}
			}
			for _, v := range vs {
				if got[v] != want[v] {
					t.Fatalf("level %d, %d bodies: crew %d, want %d for its share", lv, tc.bodies, got[v], want[v])
				}
			}
		}
	}
}

// policeScene: a call with officers out on scene.
func policeScene(t *testing.T, lv int, seed int64) (*Game, []*Cop) {
	t.Helper()
	g := ambGame(t, lv, seed, false)
	for i := 0; i < 2; i++ {
		g.bodies = append(g.bodies, &Body{pos: g.lvl.RandomRoam(g.rng), reported: true, byRex: true})
	}
	g.raisePolice(g.bodies[0].pos, false)
	var cops []*Cop
	if !runPlay(g, 40, func() bool {
		cops = nil
		for _, c := range g.cops {
			if !c.medic {
				cops = append(cops, c)
			}
		}
		return len(cops) > 0
	}) {
		t.Fatalf("level %d: the police never came", lv)
	}
	return g, cops
}

// The case closed (suspect taken / disguise): an officer who just got out stays 20 s sealing off
// the place; one who has done his 20 s goes back at once.
func TestCloseCaseFreshCopsStay20s(t *testing.T) {
	for _, lv := range []int{0, 7} {
		g, cops := policeScene(t, lv, 7)
		old := &Cop{pos: cops[0].pos, kind: unitPolice, state: cSweep, onSceneT: 25, veh: cops[0].veh}
		g.cops = append(g.cops, old)
		g.closeCase()
		if old.state != cLeave {
			t.Fatalf("level %d: an officer with his 20 s done goes back", lv)
		}
		for _, c := range cops {
			if c.state != cCordon {
				t.Fatalf("level %d: a fresh officer (%.1fs on scene) must seal off the place, state %d", lv, c.onSceneT, c.state)
			}
		}
		for i := 0; i < 30*60; i++ {
			g.updatePlay(1.0 / 60)
			g.mode = ModePlay
			for _, c := range cops {
				if containsCop(g.cops, c) && c.state == cLeave && c.onSceneT < copMinStay-0.05 {
					t.Fatalf("level %d: an officer went back after %.1fs on scene", lv, c.onSceneT)
				}
				if c.state == cChase {
					t.Fatalf("level %d: the case is closed: nobody chases", lv)
				}
			}
		}
		for _, c := range cops {
			if containsCop(g.cops, c) && c.state != cLeave {
				t.Fatalf("level %d: after 20 s the officer goes back (state %d)", lv, c.state)
			}
		}
	}
}

// A higher group takes over: the old one stays its 20 s (sealing off, not hunting), then goes.
func TestEscalationOldTierStaysMinimum(t *testing.T) {
	g, cops := policeScene(t, 0, 8)
	for _, c := range cops {
		c.onSceneT = 3
	}
	g.changeResponse(unitFBI)
	for _, c := range cops {
		if c.state != cCordon {
			t.Fatalf("relieved after 3 s the officer must stay sealing off the place (state %d)", c.state)
		}
	}
	g.p.pos = cops[0].pos // right in front of them: they are off the case and do not chase
	hist := map[*Cop]string{}
	for i := 0; i < 20*60; i++ {
		g.updatePlay(1.0 / 60)
		g.mode = ModePlay
		for _, c := range cops {
			if h := hist[c]; len(h) == 0 || h[len(h)-1] != byte('A'+int(c.state)) {
				hist[c] += string(rune('A' + int(c.state)))
			}
			if c.state == cChase || c.state == cSweep {
				t.Fatalf("a relieved officer is back on the case (state %d)", c.state)
			}
			if c.state == cLeave && c.onSceneT < copMinStay-0.05 {
				t.Fatalf("a relieved officer went back after %.1fs", c.onSceneT)
			}
		}
	}
	for _, c := range cops {
		if containsCop(g.cops, c) && c.state != cLeave {
			t.Fatalf("after its 20 s the old group goes back (state %d, states %s, scene %.1f)", c.state, hist[c], c.onSceneT)
		}
	}
}

// A paramedic with a body walks 40 % faster than the old 32 px/s.
func TestAmbulanceSpeedBoostCarrying(t *testing.T) {
	if math.Abs(medicCarrySpeed-32*1.4) > 1e-9 || medicCarrySpeed <= medicSpeed {
		t.Fatalf("carry speed %.1f, want 44.8 (and faster than walking empty, %.1f)", medicCarrySpeed, medicSpeed)
	}
	g := ambGame(t, 0, 9, false)
	v := &Vehicle{kind: 1, x: g.lvl.Def.CarTo.X, y: g.lvl.Def.CarFrom.Y, state: 1, spawned: true}
	g.veh = append(g.veh, v)
	best := 0.0
	for k := 0; k < 6 && best == 0; k++ {
		start := g.lvl.RandomRoam(g.rng)
		b := &Body{pos: start, reported: true, carried: true, bagged: true}
		g.bodies = append(g.bodies, b)
		c := &Cop{pos: start, medic: true, veh: v, state: cReturn, carry: b}
		c.path = g.lvl.Path(c.pos, g.vehDoor(c))
		if len(c.path) < 6 {
			continue
		}
		g.cops = []*Cop{c}
		walked, p0 := 0.0, c.pos
		for i := 0; i < 30; i++ { // half a second
			g.updateCop(c, 1.0/60)
			walked += dist(p0, c.pos)
			p0 = c.pos
		}
		best = walked * 2
	}
	if best == 0 {
		t.Fatal("no long path to test on")
	}
	if want := 32 * 1.4 * paceK * 0.9; best < want {
		t.Fatalf("a paramedic carrying walks %.1f px/s, want >= %.1f", best, want)
	}
}

// The special arrivals: the medical helicopter of the island, the rescue launch with its rope
// ladder of the pirate ship, the little ship with a cross at the station. Each one gets there,
// its crew takes the bodies away, and the whole thing draws without a panic, 1 and 2 players.
func TestAmbulanceSpecialArrivals(t *testing.T) {
	st := len(campaignLevels) - 1
	for _, two := range []bool{false, true} {
		for _, lv := range []int{specialJurassic, pirateIdx, st} {
			g := ambGame(t, lv, 10, two)
			orig := ambBodies(g, 3)
			g.policeCalled, g.ambCalled, g.pendingAmb = true, true, g.time
			g.maybeAmbulance()
			var v *Vehicle
			for _, x := range g.veh {
				if medicVeh(x) {
					v = x
				}
			}
			if v == nil {
				t.Fatalf("level %d: no ambulance", lv)
			}
			switch lv {
			case specialJurassic:
				if v.kind != 3 || !v.isHeli() || v.ty != jpPads[0].Y {
					t.Fatalf("island: the ambulance must be the medical helicopter on the helipad (kind %d)", v.kind)
				}
			case pirateIdx:
				if !v.ladder || v.kind != 1 || v.x > 0 {
					t.Fatalf("pirate ship: the ambulance must be the rescue launch coming along the hull")
				}
			default:
				if g.lvl.Def.Transit != "space" || transitCraft(v) != crMedic {
					t.Fatalf("station: the ambulance must be a little ship with the cross")
				}
			}
			scr := ebiten.NewImage(screenW, screenH)
			climbed, ladderSeen, delivered := false, false, false
			w := newAmbWatch()
			for i := 0; i < 120*60; i++ {
				g.updatePlay(1.0 / 60)
				g.mode = ModePlay
				w.step(g)
				for _, c := range g.cops {
					if c.climbPh == ladUp {
						climbed = true
					}
					if c.climbPh == ladDeliver && c.carry != nil {
						delivered = true
					}
				}
				if v.ladder && v.state == 1 {
					ladderSeen = true
				}
				if i%600 == 0 || (v.state == 1 && i%120 == 0) {
					ambDrawScene(g, scr) // no panic: helicopter, launch with the ladder, ship, the climbing paramedic
				}
			}
			for _, e := range w.errs {
				t.Errorf("level %d two=%v: %s", lv, two, e)
			}
			gone := 0
			for _, b := range orig {
				in := false
				for _, x := range g.bodies {
					in = in || x == b
				}
				if !in {
					gone++
				}
			}
			if gone == 0 {
				t.Fatalf("level %d two=%v: the paramedics took no body away (%d left)", lv, two, w.left)
			}
			if lv == pirateIdx && (!climbed || !ladderSeen || !delivered) {
				t.Fatalf("pirate ship: up the rope ladder %v, ladder hanging %v, down with the body %v", climbed, ladderSeen, delivered)
			}
			scr.Deallocate()
		}
	}
}

func containsBody(bs []*Body, b *Body) bool {
	for _, x := range bs {
		if x == b {
			return true
		}
	}
	return false
}

// ambDrawScene paints what the world drawing paints of the emergency services (the full Draw needs a
// running game: it reads pixels): every vehicle and every officer / paramedic, the way draw.go does.
func ambDrawScene(g *Game, scr *ebiten.Image) {
	for _, v := range g.veh {
		switch {
		case v.isHeli():
			g.drawHeli(scr, v)
		case g.lvl.Def.Transit != "":
			g.drawTransit(scr, v)
		}
	}
	for _, c := range g.cops {
		anim, phase := g.copPose(c)
		cp := g.copDrawPos(c)
		g.drawActor(scr, g.copCSFor(c), anim, phase, cp, c.dir, 0, false)
		if c.carry != nil && c.carry.cs != nil {
			bi := g.bodyLyingImg(c.carry)
			g.drawImg(scr, bi, g.wx(cp.X), g.wy(cp.Y)-32, float64(bi.Bounds().Dx())/2, float64(bi.Bounds().Dy()))
		}
	}
}

// 2P revive: the partner gets up some other way while the paramedics are already out: they stand up
// and walk back to the ambulance (before, they vanished on the spot and it drove off at once).
func TestReviveCrewWalksBackWhenPartnerUp(t *testing.T) {
	g := reviveGame()
	kill(g, g.p2, "Te ha atropellado un coche")
	g.callAmbulance(g.p, nil)
	const dt = 1.0 / 60
	var j *reviveJob
	for i := 0; i < 60*90; i++ {
		g.time += dt
		g.updateVehicles(dt)
		g.updateTwoPlayers(dt)
		if j = g.rev.job; j != nil && j.phase == rvCheck {
			break
		}
	}
	if j == nil || j.phase != rvCheck || len(j.crew) == 0 {
		t.Fatal("setup: the paramedics never reached the body")
	}
	g.p2.dead = false // up by other means
	v := j.v
	g.time += dt
	g.updateTwoPlayers(dt)
	if g.rev.job != j || j.phase != rvUp {
		t.Fatalf("the crew must stand up and walk back, not vanish (job %v phase %d)", g.rev.job != nil, j.phase)
	}
	if v != nil && v.state == 2 {
		t.Fatal("the ambulance must wait for its crew")
	}
}
