package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"testing"
)

// ---- helpers ---------------------------------------------------------------------------------------------

// streetLevels: the campaign levels where the street events fit.
func streetLevels(t *testing.T, g *Game, id string) []int {
	t.Helper()
	in := peEligible(g, id)
	if len(in) == 0 {
		t.Fatalf("%s fits no campaign level", id)
	}
	return in
}

// streetStep: one frame of the party with its street traffic (peStep has no traffic).
func streetStep(g *Game, dt float64) {
	g.time += dt
	g.updateAmbient(dt)
	g.updateParty(dt)
	for _, gu := range g.guests {
		g.updateGuest(gu, dt)
	}
}

// streetStart: a street level (idx) loaded with the event forced; it runs the party until the event starts.
func streetStart(t *testing.T, id string, idx int, two bool, seed int64) *Game {
	t.Helper()
	peLoadSeed = seed
	g := peLoad(t, idx, two)
	peLoadSeed = 0
	prevGod := g.god
	t.Cleanup(func() { g.god = prevGod })
	g.god = true // the players are never run over while the test watches the street
	g.rng = rand.New(rand.NewSource(seed*31 + int64(idx)))
	partyRng = rand.New(rand.NewSource(seed*7919 + int64(idx)))
	peForce(g, id)
	g.pev.plan[0].at = 6 // a few cars on the road first
	for s := 0; s < 30*60 && g.partyRunning(id) == nil && len(g.pev.plan) > 0; s++ {
		streetStep(g, 1.0/30)
	}
	return g
}

// ---- where ----------------------------------------------------------------------------------------------

// The street events fit the parties with the street of the police cars (and only them), not the highway nor the sea;
// some parties have none.
func TestStreetEventsOnlyWhereThereIsAStreet(t *testing.T) {
	g := partyGame(t)
	for _, id := range []string{"averia", "taxi"} {
		in := streetLevels(t, g, id)
		t.Logf("%s fits %d of %d campaign levels: %v", id, len(in), len(campaignLevels), in)
		if len(in) == len(campaignLevels) {
			t.Errorf("%s fits every level", id)
		}
		for _, i := range in {
			d := levelDefs[i]
			if d.Police != "car" || d.Transit != "" || d.Highway {
				t.Errorf("%s fits level %d (%s) without an ordinary street", id, i, d.Title)
			}
		}
		e := partyByID(id)
		if e.Group != "street" || e.Prob < 0.10 || e.Prob > 0.40 || e.Big || e.Minor {
			t.Errorf("%s: group %q chance %.2f", id, e.Group, e.Prob)
		}
		if e.Eligible(g, partyTraits{idx: -1}) {
			t.Errorf("%s fits a level that is not known", id)
		}
	}
}

// ---- 249: the car out of fuel ---------------------------------------------------------------------------

// A car of the traffic stops: it leaves the lane while it stands (nobody is run over by it), the cars behind wait in a
// queue and never drive through it, the driver gets out, looks, pushes, and the car goes back to its lane at its own
// speed. It ends by itself, well within its time, in every street level tried, with one and two players.
func TestCarBreakdownStopsQueuesAndGoesOn(t *testing.T) {
	probe := partyGame(t)
	levels := streetLevels(t, probe, "averia")
	queued, ran := 0, 0
	for k, idx := range levels {
		if k >= 6 {
			break
		}
		for _, two := range []bool{false, true} {
			g := streetStart(t, "averia", idx, two, int64(k+1))
			a := g.partyRunning("averia")
			if a == nil {
				t.Logf("level %d: no car to stop came by", idx)
				continue
			}
			ran++
			s := a.D.(*averiaState)
			car, v0 := s.car, s.v0
			sawPush, sawQueue := false, false
			for i := 0; i < 30*80 && g.partyRunning("averia") != nil; i++ {
				streetStep(g, 1.0/30)
				if i%15 == 0 {
					peDraw(g)
				}
				if s.phase >= avOut && s.phase < avDone && ambIn(g, car) {
					t.Fatalf("level %d: the stopped car is still in the lane", idx)
				}
				if s.phase == avPush {
					sawPush = true
				}
				for _, o := range g.amb {
					if o == car || o.dead || math.Abs(o.y-car.y) > 8 || math.Copysign(1, o.v0) != s.dir || s.phase >= avDone || s.phase == avCome {
						continue
					}
					if o.wait > 0 {
						sawQueue = true
					}
					// never through it: a car behind it keeps a gap
					if gap := (car.x-o.x)*s.dir - (car.length+o.length)/2; (car.x-o.x)*s.dir > 0 && gap < 2 {
						t.Fatalf("level %d: a car drives into the stopped one (gap %.1f)", idx, gap)
					}
				}
			}
			if g.partyRunning("averia") != nil {
				t.Fatalf("level %d: it did not end in 80 s", idx)
			}
			if !sawPush {
				t.Errorf("level %d: the driver never pushed", idx)
			}
			if sawQueue {
				queued++
			}
			if !car.dead && (!ambIn(g, car) || car.v0 != v0) {
				t.Errorf("level %d: the car did not go back to its lane at its speed (in %v, v0 %.0f want %.0f)", idx, ambIn(g, car), car.v0, v0)
			}
			if s.drv != nil {
				t.Errorf("level %d: the driver is still out", idx)
			}
			if g.stars != 0 || g.wanted != 0 {
				t.Errorf("level %d: the breakdown raised the police (stars %d)", idx, g.stars)
			}
		}
	}
	if ran < 3 {
		t.Fatalf("it ran only %d times", ran)
	}
	if queued == 0 {
		t.Error("no car ever waited behind the stopped one")
	}
}

// ---- 250: the taxi --------------------------------------------------------------------------------------

// The taxi takes 1-4 guests: they are gone (not dead, no points, no casualty) and out of the kill target, the party
// still ends when everybody else is dead; later a taxi brings new guests within the caps (cap.go).
func TestTaxiTakesGuestsAndMoreComeLater(t *testing.T) {
	probe := partyGame(t)
	levels := streetLevels(t, probe, "taxi")
	prevGone, prevBack, prevBreach := taxiGoneN, taxiBackN, capBreaches
	t.Cleanup(func() { taxiGoneN, taxiBackN, capBreaches = prevGone, prevBack, prevBreach })
	took, back := 0, 0
	for k, idx := range levels {
		if k >= 4 {
			break
		}
		for _, two := range []bool{false, true} {
			g := streetStart(t, "taxi", idx, two, int64(k+3))
			a := g.partyRunning("taxi")
			if a == nil {
				t.Logf("level %d: the taxi did not start", idx)
				continue
			}
			s := a.D.(*taxiState)
			if n := len(s.riders); n < 1 || n > 4 {
				t.Fatalf("level %d: %d riders", idx, n)
			}
			for _, m := range s.riders {
				if !taxiCanTake(g, m.gu) && m.gu.name != "" {
					t.Errorf("level %d: a guest who may not leave was taken", idx)
				}
			}
			taxiGoneN, taxiBackN, capBreaches = 0, 0, 0
			pts0 := g.points
			var riders []*Guest
			for _, m := range s.riders {
				riders = append(riders, m.gu)
			}
			for i := 0; i < 30*140 && g.partyRunning("taxi") != nil; i++ {
				gone0, back0, tot0, n0 := taxiGoneN, taxiBackN, g.total, len(g.guests)
				streetStep(g, 1.0/30)
				if d := taxiGoneN - gone0; d > 0 && len(g.guests) == n0 && g.total != tot0-d {
					t.Errorf("level %d: %d left in the taxi, the target went %d -> %d", idx, d, tot0, g.total)
				}
				if d := taxiBackN - back0; d > 0 && g.total != tot0+d {
					t.Errorf("level %d: %d came in the taxi, the target went %d -> %d", idx, d, tot0, g.total)
				}
				if g.mode != ModePlay {
					t.Fatalf("level %d: the party ended (mode %d)", idx, g.mode)
				}
				if i%15 == 0 {
					peDraw(g)
				}
				if g.total > g.goalMax() || g.guestLive() > g.guestCap() {
					t.Fatalf("level %d: caps broken (total %d/%d live %d/%d)", idx, g.total, g.goalMax(), g.guestLive(), g.guestCap())
				}
			}
			if g.partyRunning("taxi") != nil {
				t.Fatalf("level %d: it did not end in 140 s", idx)
			}
			for _, gu := range riders {
				if gu.gone && gu.dead {
					t.Errorf("level %d: a rider is gone and dead", idx)
				}
			}
			if g.points != pts0 {
				t.Errorf("level %d: the taxi gave points (%d -> %d)", idx, pts0, g.points)
			}
			if taxiGoneN > 0 {
				took++
			}
			if taxiBackN > 0 {
				back++
				if taxiBackN > taxiGoneN {
					t.Errorf("level %d: %d came back for %d gone", idx, taxiBackN, taxiGoneN)
				}
			}
			if capBreaches != 0 {
				t.Errorf("level %d: the cap safety net acted %d times", idx, capBreaches)
			}
			if gu := peStale(g, map[*Guest]bool{}); gu != nil {
				t.Errorf("level %d: a guest is still held", idx)
			}
		}
	}
	if took == 0 {
		t.Fatal("no guest ever left in the taxi")
	}
	if back == 0 {
		t.Error("the taxi never brought new guests")
	}
}

// The goal stays coherent: a party whose last living guests leave in a taxi ends (they are out of the kill target),
// and a guest taken by the taxi never counts as a kill.
func TestTaxiGoalCoherent(t *testing.T) {
	g := partyGame(t)
	g.loadLevel(0)
	g.mode, g.time = ModePlay, 10
	k0 := g.kills
	var alive []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.py == nil {
			alive = append(alive, gu)
		}
	}
	total := g.total
	taxiBoard(g, alive[0])
	if !alive[0].gone || alive[0].dead || g.total != total-1 || g.kills != k0 {
		t.Fatalf("board: gone %v dead %v total %d (was %d) kills %d", alive[0].gone, alive[0].dead, g.total, total, g.kills)
	}
	taxiBoard(g, alive[0]) // twice: nothing more
	if g.total != total-1 {
		t.Fatal("boarding twice counted twice")
	}
}

// Who may never get into the taxi.
func TestTaxiNeverTakesSpecialGuests(t *testing.T) {
	g := partyGame(t)
	g.loadLevel(0)
	g.mode = ModePlay
	var plain *Guest
	for _, gu := range g.guests {
		if taxiCanTake(g, gu) {
			plain = gu
			break
		}
	}
	if plain == nil {
		t.Fatal("no plain guest can take the taxi")
	}
	check := func(what string, f func(gu *Guest), undo func(gu *Guest)) {
		f(plain)
		if taxiCanTake(g, plain) {
			t.Errorf("a %s guest may take the taxi", what)
		}
		undo(plain)
	}
	check("dead", func(gu *Guest) { gu.dead = true }, func(gu *Guest) { gu.dead = false })
	check("named", func(gu *Guest) { gu.name = "X" }, func(gu *Guest) { gu.name = "" })
	check("kept", func(gu *Guest) { gu.keep = true }, func(gu *Guest) { gu.keep = false })
	check("VIP card", func(gu *Guest) { gu.vipCard = true }, func(gu *Guest) { gu.vipCard = false })
	check("scared", func(gu *Guest) { gu.state = gScared }, func(gu *Guest) { gu.state = gIdle })
	check("suspect", func(gu *Guest) { g.suspect = gu }, func(gu *Guest) { g.suspect = nil })
	sl := &Slasher{tgtG: plain}
	prev := g.slasher
	g.slasher = sl
	if taxiCanTake(g, plain) {
		t.Error("the target of a killer may take the taxi")
	}
	g.slasher = prev
}

// The TAXI plate is crisp: two colours only (ink and plate), the four letters inside with a pixel of margin.
func TestTaxiPlateSharp(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	plot := func(x, y, w, h int, c color.RGBA) {
		for j := y; j < y+h; j++ {
			for i := x; i < x+w; i++ {
				img.SetRGBA(i, j, c)
			}
		}
	}
	drawTaxiPlatePix(plot, 5, 5, 19, 9)
	cols := map[color.RGBA]int{}
	for y := 5; y < 14; y++ {
		for x := 5; x < 24; x++ {
			cols[img.RGBAAt(x, y)]++
		}
	}
	if len(cols) != 2 {
		t.Fatalf("%d colours on the plate, want 2", len(cols))
	}
	ink := color.RGBA{0x15, 0x15, 0x1b, 255}
	// the inner margin (one pixel all around the letters) is plate colour
	for x := 6; x < 23; x++ {
		if img.RGBAAt(x, 6) == ink || img.RGBAAt(x, 12) == ink {
			t.Fatalf("a letter touches the edge of the plate at x=%d", x)
		}
	}
	letters := cols[ink] - (2*19 + 2*7)
	if letters < 30 {
		t.Errorf("only %d letter pixels", letters)
	}
}
