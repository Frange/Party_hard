package main

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

// Tests of the petting, the ball thrower and the figure skater (pe_petting.go, pe_fetch.go, pe_icelake.go), on the
// real game after loadLevel, in several levels and with dice that change from run to run (no fixed seed decides).

// paGlobals: every global of the three events, put back after the test.
func paGlobals(t *testing.T) {
	t.Helper()
	pr, ps, pin, plv := petRng, petSeed, petInTests, petLv
	fr, fs, fin, ff, flv := fetchRng, fetchSeed, fetchInTests, fetchForce, fetchLv
	ir, is, iin, iff, ilv := iceSkRng, iceSkSeed, iceSkInTests, iceSkForce, iceSkLv
	holds, hl := pevAnimalHolds, pevAnimalHoldLvl
	t.Cleanup(func() {
		petRng, petSeed, petInTests, petLv = pr, ps, pin, plv
		fetchRng, fetchSeed, fetchInTests, fetchForce, fetchLv = fr, fs, fin, ff, flv
		iceSkRng, iceSkSeed, iceSkInTests, iceSkForce, iceSkLv = ir, is, iin, iff, ilv
		pevAnimalHolds, pevAnimalHoldLvl = holds, hl
	})
	sd := time.Now().UnixNano()
	petRng, fetchRng, iceSkRng = rand.New(rand.NewSource(sd)), rand.New(rand.NewSource(sd+1)), rand.New(rand.NewSource(sd+2))
	pevAnimalHolds, pevAnimalHoldLvl = map[*Animal]func(*Game, *Animal, float64) bool{}, nil
}

// paStep: one frame of the party, the guests and the animals.
func paStep(g *Game, dt float64) {
	g.time += dt
	g.updateParty(dt)
	for _, gu := range g.guests {
		g.updateGuest(gu, dt)
	}
	g.updateAnimals(dt)
}

// paLoad: level idx with at least one live animal of the kind (several parties are tried; nil = none had it).
func paLoad(t *testing.T, idx int, kind int, two bool) (*Game, *Animal) {
	t.Helper()
	for try := int64(1); try <= 8; try++ {
		peLoadSeed = try*31 + time.Now().UnixNano()%1000
		g := peLoad(t, idx, two)
		peLoadSeed = 0
		for _, a := range g.animals {
			if a.kind == kind && !a.dead {
				return g, a
			}
		}
	}
	return nil, nil
}

// ---- petting -------------------------------------------------------------------------------------------------

// A free run of the real dice: petting happens only with the animal near, is rare, the animal stays still and nobody
// is left held.
func TestPetOnlyNearRareAndNoStuck(t *testing.T) {
	paGlobals(t)
	petInTests = true
	starts, frames, looks := 0, 0, 0
	for _, lv := range []struct{ idx, kind int }{{0, animDog}, {13, animCat}, {3, animHorse}, {1, animDog}} {
		g, _ := paLoad(t, lv.idx, lv.kind, false)
		if g == nil {
			continue
		}
		var cur *petState
		var held Vec
		var heldAn *Animal
		for s := 0; s < 30*300; s++ {
			paStep(g, 1.0/30)
			frames++
			a := g.partyRunning("caricia")
			if a == nil {
				if cur != nil {
					if gu := cur.m.gu; !gu.out() && peHeld(gu) {
						t.Errorf("level %d: the guest is still held after the petting", lv.idx)
					}
					if pevAnimalHeld(cur.a) {
						t.Errorf("level %d: the animal is still held after the petting", lv.idx)
					}
				}
				cur, heldAn = nil, nil
				continue
			}
			ps := a.D.(*petState)
			if ps != cur {
				cur, heldAn, held = ps, ps.a, ps.a.pos
				starts++
				if d := dist(ps.m.gu.pos, ps.a.pos); d > petNearR+2 {
					t.Errorf("level %d: a guest %.0f px away went to pet the animal", lv.idx, d)
				}
				if !petKindOK(ps.a) {
					t.Errorf("level %d: petting an animal of kind %d", lv.idx, ps.a.kind)
				}
			}
			if heldAn != nil && pevAnimalHeld(heldAn) {
				if dist(heldAn.pos, held) > 0.01 || heldAn.state == aFlee {
					t.Fatalf("level %d: the petted animal moved (%.1f px) or fled", lv.idx, dist(heldAn.pos, held))
				}
			}
			if g.time-a.t0 > 30 {
				t.Fatalf("level %d: a petting lasted more than 30 s", lv.idx)
			}
		}
		looks += petLv.looks
	}
	mins := float64(frames) / 30 / 60
	t.Logf("pettings: %d in %.1f min of parties (%d looks with a pair near)", starts, mins, looks)
	if mins > 0 && float64(starts)/mins > 1.6 {
		t.Errorf("%d pettings in %.1f minutes: too often", starts, mins)
	}
	if looks > 30 && float64(starts) > float64(looks)*0.3 {
		t.Errorf("%d pettings out of %d chances: not rare", starts, looks)
	}
}

// Forced (-pevent caricia:...): for a dog, a cat and a horse the guest walks up, pets for 3-6 s, the animal does not
// move, and both are let go with their cooldowns.
func TestPetForcedEachKind(t *testing.T) {
	for _, lv := range []struct {
		idx, kind int
		name      string
	}{{0, animDog, "perro"}, {13, animCat, "gato"}, {3, animHorse, "caballo"}} {
		t.Run(lv.name, func(t *testing.T) {
			paGlobals(t)
			g, _ := paLoad(t, lv.idx, lv.kind, false)
			if g == nil {
				t.Skipf("no %s in level %d in 8 parties", lv.name, lv.idx)
			}
			dbgPartyEvent = "caricia:" + lv.name
			// the player far away from the animals (a cat runs from him)
			g.p.pos = g.lvl.NearestFree(Vec{8, 8})
			petLv.lvl = nil
			var ps *petState
			var at Vec
			petT := 0.0
			started, full := 0, 0
			check := func() {
				if ps.a.kind != lv.kind {
					t.Errorf("petted kind %d, want %d", ps.a.kind, lv.kind)
				}
				if petT > petMaxT+0.2 {
					t.Errorf("petted for %.1f s, want at most %.0f", petT, petMaxT)
				}
				if petT >= petMinT-0.2 {
					full++
				}
				if pevAnimalHeld(ps.a) || peHeld(ps.m.gu) && !ps.m.gu.out() {
					t.Error("the animal or the guest is still held")
				}
				if petLv.aCool[ps.a] <= g.time || ps.m.gu != nil && petLv.gCool[ps.m.gu] <= g.time {
					t.Error("no cooldown for the guest or the animal")
				}
			}
			for s := 0; s < 30*90 && full == 0; s++ {
				paStep(g, 1.0/30)
				a := g.partyRunning("caricia")
				if a == nil || ps != nil && a.D != ps {
					if ps != nil {
						check()
						ps = nil
					}
					if a == nil {
						continue
					}
				}
				x := a.D.(*petState)
				if ps == nil {
					ps, at, petT = x, x.a.pos, 0
					started++
				}
				if x.phase == ptPet {
					petT += 1.0 / 30
					if dist(x.a.pos, at) > 0.01 || x.a.state == aFlee {
						t.Fatalf("the %s moved while petted", lv.name)
					}
				}
			}
			t.Logf("%d pettings started, %d complete", started, full)
			if started == 0 {
				t.Skip("nobody could reach an animal in this party")
			}
			if full == 0 {
				t.Error("no petting was completed")
			}
		})
	}
}

// The fear of the cat wins: the player comes near the petted cat and it runs away (its own AI), the petting ends.
func TestPetCatStillRunsFromThePlayer(t *testing.T) {
	// It used to load only party 13 and SKIP when that one came without a cat (or the petting did not start): the
	// test proved nothing then. Now it tries the parties with a cat in their own list of animals (13 and the HOUSE
	// PARTY, whose cat is fixed in levels.go) and FAILS if none of them gets a petted cat.
	prevEv := dbgPartyEvent
	t.Cleanup(func() { dbgPartyEvent = prevEv })
	var g *Game
	var ps *petState
	for _, idx := range []int{13, 0, 13, 0} {
		paGlobals(t)
		if g, _ = paLoad(t, idx, animCat, false); g == nil {
			continue
		}
		dbgPartyEvent = "caricia:gato"
		g.p.pos = g.lvl.NearestFree(Vec{8, 8})
		petLv.lvl = nil
		ps = nil
		for s := 0; s < 30*30 && (ps == nil || ps.phase != ptPet); s++ {
			paStep(g, 1.0/30)
			if a := g.partyRunning("caricia"); a != nil {
				ps = a.D.(*petState)
			}
		}
		if ps != nil && ps.phase == ptPet {
			break
		}
	}
	if g == nil || ps == nil || ps.phase != ptPet {
		t.Fatal("no party with a cat got a petting going (parties 13 and 0)")
	}
	g.p.dead, g.p.hidden = false, false
	g.p.pos = Vec{ps.a.pos.X + 10, ps.a.pos.Y}
	fled := false
	for s := 0; s < 30*3; s++ {
		paStep(g, 1.0/30)
		if ps.a.state == aFlee {
			fled = true
		}
	}
	if !fled || pevAnimalHeld(ps.a) {
		t.Errorf("the cat did not run from the player (fled %v, held %v)", fled, pevAnimalHeld(ps.a))
	}
}

// ---- the ball thrower ------------------------------------------------------------------------------------------

func paFetch(t *testing.T, two bool, body func(g *Game, s *fetchState) string) {
	t.Helper()
	last := ""
	for _, idx := range []int{0, 5, campaignIdx("festival"), campaignIdx("cake"), campaignIdx("funfair")} {
		paGlobals(t)
		g, _ := paLoad(t, idx, animDog, two)
		if g == nil {
			last = "no dog"
			continue
		}
		if !fetchIcons[g.lvl.Def.Icon] {
			t.Fatalf("level %d (%s) is not a park venue", idx, g.lvl.Def.Icon)
		}
		fetchForce, fetchLv.lvl = true, nil
		var s *fetchState
		for i := 0; i < 30*300 && s == nil; i++ {
			paStep(g, 1.0/30)
			if a := g.partyRunning("pelota"); a != nil {
				s = a.D.(*fetchState)
			}
		}
		if s == nil {
			last = "it never started"
			continue
		}
		if last = body(g, s); last == "" {
			return
		}
		t.Logf("level %d: %s", idx, last)
	}
	t.Error(last)
}

// The dog runs after the ball, picks it up and brings it back; 2-4 throws, no stuck phase; nothing is left held and
// the animals are as many as before (one of the level's own).
func TestFetchDogBringsTheBallBack(t *testing.T) {
	for _, two := range []bool{false, true} {
		paFetch(t, two, func(g *Game, s *fetchState) string {
			n0 := len(g.animals)
			picked, t0 := false, g.time
			for i := 0; i < 30*90 && g.partyRunning("pelota") != nil; i++ {
				paStep(g, 1.0/30)
				if s.inMouth && s.phase == fcBack {
					picked = true
				}
				if s.phase == fcChase && s.throws > 0 && s.t > fetchFlyMin+fetchFlyVar+1 && !s.onGround && !s.inMouth {
					return "the ball never came down"
				}
			}
			if g.partyRunning("pelota") != nil {
				return "it did not end in 90 s"
			}
			if s.throws == 0 || !picked || s.backs == 0 {
				return "no round trip"
			}
			if s.throws > 4 || g.time-t0 > fetchMaxT+25 {
				t.Errorf("%d throws in %.0f s", s.throws, g.time-t0)
			}
			if pevAnimalHeld(s.dog) || !s.m.gu.out() && peHeld(s.m.gu) {
				t.Error("the dog or the thrower is still held")
			}
			if len(g.animals) != n0 || len(g.animals) > animalMax {
				t.Errorf("animals %d -> %d (max %d)", n0, len(g.animals), animalMax)
			}
			for _, gu := range g.guests {
				if gu == nil {
					t.Fatal("nil guest")
				}
			}
			return ""
		})
	}
}

// The thrower is killed while the dog runs: the dog goes to the body with the ball and waits there, then it is free.
func TestFetchThrowerKilledDogWaits(t *testing.T) {
	paFetch(t, false, func(g *Game, s *fetchState) string {
		for i := 0; i < 30*30 && s.phase != fcChase; i++ {
			paStep(g, 1.0/30)
			if g.partyRunning("pelota") == nil {
				return "ended before the throw"
			}
		}
		if s.phase != fcChase {
			return "no throw"
		}
		g.killGuest(s.m.gu, false)
		body := s.m.gu.pos
		waited, near := false, false
		for i := 0; i < 30*40 && g.partyRunning("pelota") != nil; i++ {
			paStep(g, 1.0/30)
			if s.phase == fcMourn && s.inMouth {
				waited = true
				if s.mourned && dist(s.dog.pos, body) < 30 {
					near = true
				}
			}
		}
		if g.partyRunning("pelota") != nil {
			return "the dog waited for ever"
		}
		if !waited || !near {
			return "the dog did not wait by the body with the ball"
		}
		if pevAnimalHeld(s.dog) {
			t.Error("the dog is still held")
		}
		return ""
	})
}

// The dog is killed: the event ends at once and the thrower goes back to the party.
func TestFetchDogKilledEnds(t *testing.T) {
	paFetch(t, false, func(g *Game, s *fetchState) string {
		for i := 0; i < 30*30 && s.phase != fcChase; i++ {
			paStep(g, 1.0/30)
		}
		if g.partyRunning("pelota") == nil {
			return "ended before the throw"
		}
		g.killAnimal(s.dog)
		paStep(g, 1.0/30)
		paStep(g, 1.0/30)
		if g.partyRunning("pelota") != nil {
			return "still running with a dead dog"
		}
		if gu := s.m.gu; !gu.out() && peHeld(gu) {
			return "the thrower is still held"
		}
		return ""
	})
}

// The dice: a party of a park venue gets it about fetchChance of the time; never outside those venues.
func TestFetchSometimesAndOnlyInParks(t *testing.T) {
	paGlobals(t)
	if fetchChance < 0.10 || fetchChance > 0.12 {
		t.Errorf("fetchChance %.2f", fetchChance)
	}
	g := partyGame(t)
	fetchInTests = true
	t.Cleanup(func() { fetchInTests = false })
	for idx := range campaignLevels {
		g.loadLevel(idx)
		yes := 0
		for k := 0; k < 1500; k++ {
			fetchReset(g)
			if fetchLv.want {
				yes++
			}
		}
		f := float64(yes) / 1500
		if fetchIcons[g.lvl.Def.Icon] {
			if f < 0.07 || f > 0.17 {
				t.Errorf("level %d (%s): fetch in %.0f %% of the parties", idx, g.lvl.Def.Icon, f*100)
			}
		} else if yes > 0 {
			t.Errorf("level %d (%s) is not a park but got the ball thrower", idx, g.lvl.Def.Icon)
		}
	}
}

// ---- the figure skater -------------------------------------------------------------------------------------------

// Only the snow and the hockey parties have ice for the skater; the others never get it.
func TestIceSkaterOnlyOnIceLevels(t *testing.T) {
	paGlobals(t)
	g := partyGame(t)
	iceSkInTests = true
	t.Cleanup(func() { iceSkInTests = false })
	n := 0
	for idx := range campaignLevels {
		g.loadLevel(idx)
		fits := iceSkFits(g)
		if fits != iceSkIcons[g.lvl.Def.Icon] {
			t.Errorf("level %d (%s): fits %v", idx, g.lvl.Def.Icon, fits)
		}
		yes := 0
		for k := 0; k < 600; k++ {
			iceSkReset(g)
			if iceSkLv.want {
				yes++
			}
		}
		if !fits && yes > 0 {
			t.Errorf("level %d (%s) got the figure skater", idx, g.lvl.Def.Icon)
		}
		if fits {
			n++
			if f := float64(yes) / 600; f < 0.18 || f > 0.42 {
				t.Errorf("level %d: the skater in %.0f %% of the parties", idx, f*100)
			}
			z := iceSkZone(g.lvl)
			free := 0
			for y := z.Y; y < z.Y+z.H; y += 8 {
				for x := z.X; x < z.X+z.W; x += 8 {
					if g.iceSkOK(Vec{x, y}) {
						free++
					}
				}
			}
			if free < 100 {
				t.Errorf("level %d: only %d free points of ice", idx, free)
			}
		}
	}
	if n != 2 {
		t.Errorf("%d ice levels, want 2 (ski resort and hockey rink)", n)
	}
}

// The show: he gets onto the ice, does tricks (one funny fall and getting up), stays on the ice, bows and is let go;
// killed in the middle it ends at once. In the ski resort and on the hockey rink, one and two players.
func TestIceSkaterShow(t *testing.T) {
	for _, idx := range []int{campaignIdx("snow"), campaignIdx("hockey")} {
		for _, two := range []bool{false, true} {
			whys := ""
			ok := false
			for try := 0; try < 8 && !ok; try++ {
				paGlobals(t)
				peLoadSeed = int64(try)*17 + time.Now().UnixNano()%1000
				g := peLoad(t, idx, two)
				peLoadSeed = 0
				if !iceSkFits(g) {
					t.Fatalf("level %d does not fit", idx)
				}
				dbgPartyEvent = "hielo"
				iceSkLv.lvl = nil
				var s *iceSkState
				t0 := 0.0
				for i := 0; i < 30*60 && s == nil; i++ {
					paStep(g, 1.0/30)
					if a := g.partyRunning("hielo"); a != nil {
						s, t0 = a.D.(*iceSkState), g.time
					}
				}
				if s == nil {
					t.Fatalf("level %d: the skater never came", idx)
				}
				for i := 0; i < 30*90 && g.partyRunning("hielo") != nil; i++ {
					paStep(g, 1.0/30)
				}
				if g.partyRunning("hielo") != nil {
					t.Fatalf("level %d: the show did not end", idx)
				}
				if peHeld(s.m.gu) && !s.m.gu.out() {
					t.Errorf("level %d: the skater is still held (%s)", idx, s.why)
				}
				if s.outIce || s.onIce < s.skate-0.5 {
					t.Errorf("level %d: he left the ice (%.1f of %.1f s on it)", idx, s.onIce, s.skate)
				}
				if s.why != "" { // an alarm or a fright of the party itself: it ended early, as it must; another party
					whys += s.why + " "
					continue
				}
				ok = true
				if s.skate < iceSkMinT-1 || g.time-t0 > iceSkMaxT+iceSkWalkMax+12 {
					t.Errorf("level %d: skated %.0f s, %.0f s in all", idx, s.skate, g.time-t0)
				}
				if s.spins+s.jumps+s.swans == 0 || s.falls != 1 {
					t.Errorf("level %d: spins %d jumps %d swans %d falls %d", idx, s.spins, s.jumps, s.swans, s.falls)
				}
			}
			if !ok {
				t.Errorf("level %d two %v: no show reached the bow (%s)", idx, two, whys)
			}
		}
	}
	// killed in the middle of the show
	paGlobals(t)
	g := peLoad(t, campaignIdx("snow"), false)
	dbgPartyEvent = "hielo"
	iceSkLv.lvl = nil
	var s *iceSkState
	for i := 0; i < 30*60 && (s == nil || s.phase == isWalk); i++ {
		paStep(g, 1.0/30)
		if a := g.partyRunning("hielo"); a != nil {
			s = a.D.(*iceSkState)
		}
	}
	if s == nil || s.phase == isWalk {
		t.Fatal("the skater never got onto the ice")
	}
	g.killGuest(s.m.gu, false)
	paStep(g, 1.0/30)
	if g.partyRunning("hielo") != nil {
		t.Error("the show goes on with a dead skater")
	}
}

// The three never move g.rng, and they are listed for -pevents as reactive minor events.
func TestPevAnimalsRegistered(t *testing.T) {
	list := partyList()
	for _, id := range []string{"caricia", "pelota", "hielo"} {
		e := partyByID(id)
		if e == nil || !e.Reactive || !e.Minor || e.Big {
			t.Fatalf("%s: not registered as a reactive minor event", id)
		}
		if !strings.Contains(list, id) {
			t.Errorf("%s not in -pevents", id)
		}
	}
}
