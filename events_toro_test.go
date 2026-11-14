package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the mechanical bull (events_toro.go) with the real Game after loadLevel.

// toroGlobals: every global of the bull is put back after the test.
func toroGlobals(t *testing.T) {
	t.Helper()
	pf, pi, po, pr, ps := toroForce, toroInTests, toroOff, toroRng, toroSeed
	pl, pn, pc := toroLvl, toroNext, toroCool
	ts := [...]float64{toroSessMin, toroSessVar, toroPauseMin, toroPauseVar, toroCoolMin, toroCoolVar}
	t.Cleanup(func() {
		toroForce, toroInTests, toroOff, toroRng, toroSeed = pf, pi, po, pr, ps
		toroLvl, toroNext, toroCool = pl, pn, pc
		toroSessMin, toroSessVar, toroPauseMin, toroPauseVar, toroCoolMin, toroCoolVar = ts[0], ts[1], ts[2], ts[3], ts[4], ts[5]
		toroDecision = false
	})
}

// toroVenues: the campaign levels where the bull may come.
func toroVenues() []int {
	var out []int
	for i := range campaignLevels {
		if i < len(levelDefs) && toroFits(levelDefs[i]) {
			out = append(out, i)
		}
	}
	return out
}

// toroUnplace: the reach of a level with and without its bull (the bull taken out by hand).
func toroReachBoth(l *Level, pr *Prop) (with, without []bool) {
	with = l.reachFrom(l.NearestFree(l.Def.Start))
	props, sp, sn := l.Props, l.solidsP, l.solidsN
	var np []*Prop
	for _, o := range props {
		if o != pr {
			np = append(np, o)
		}
	}
	drop := func(rs []Rect) []Rect {
		var out []Rect
		gone := false
		for _, r := range rs {
			if !gone && r == pr.Foot {
				gone = true
				continue
			}
			out = append(out, r)
		}
		return out
	}
	l.Props, l.solidsP, l.solidsN = np, drop(sp), drop(sn)
	l.initNav()
	without = l.reachFrom(l.NearestFree(l.Def.Start))
	l.Props, l.solidsP, l.solidsN = props, sp, sn
	l.initNav()
	return
}

// The bull comes only in its venues (forced: in every one of them it finds a spot), never elsewhere; its spot keeps every
// floor cell reachable, nothing stands on its mattress, the line, the mount and the landings are open and reachable.
func TestToroPropOnlyInItsVenuesAndReachable(t *testing.T) {
	g := partyGame(t)
	toroGlobals(t)
	venues := map[int]bool{}
	for _, i := range toroVenues() {
		venues[i] = true
		t.Logf("venue %d %s (%s)", i, levelDefs[i].Title, levelDefs[i].Icon)
	}
	if len(venues) < 4 {
		t.Fatalf("only %d campaign venues for the bull", len(venues))
	}
	toroForce = true
	placed := 0
	for seed := int64(1); seed <= 2; seed++ {
		for i := range campaignLevels {
			toroSeed, toroRng = seed*31+int64(i), nil
			g = peLoad(t, i, false)
			pr := g.lvl.bullRide()
			if !venues[i] {
				if pr != nil {
					t.Errorf("%s: a bull outside its venues", g.lvl.Def.Title)
				}
				continue
			}
			if pr == nil {
				t.Errorf("%s (seed %d): forced bull found no spot", g.lvl.Def.Title, seed)
				continue
			}
			placed++
			l := g.lvl
			with, without := toroReachBoth(l, pr)
			for c := range without {
				if without[c] && !with[c] && !l.blocked[c] {
					t.Errorf("%s: the bull cuts off cell %d", l.Def.Title, c)
					break
				}
			}
			mat := Rect{pr.X - 31, pr.Y - 24, 62, 24}
			for _, o := range l.Props {
				if o != pr && !o.gone && !overlayKinds[o.Name] && (pbOverlap(o.base(), mat) || pbOverlap(o.Foot, mat)) {
					t.Errorf("%s: %s stands on the mattress", l.Def.Title, o.Name)
				}
			}
			for _, q := range []Vec{bullMount(pr), bullLand(pr, -1), bullLand(pr, 1)} {
				if !l.bullDry(q) || !with[l.navIdx(q)] {
					t.Errorf("%s: a place of the bull is closed: %v", l.Def.Title, q)
				}
			}
			if l.bullQueueDir(pr, with) < 0 {
				t.Errorf("%s: no room for the line", l.Def.Title)
			}
			if dist(Vec{pr.X, pr.Y}, l.Def.Start) < 100 {
				t.Errorf("%s: the bull is at the start", l.Def.Title)
			}
			for _, dr := range l.Def.Dance {
				if dr.Grow(10).Hits(mat) {
					t.Errorf("%s: the bull is on the dance floor", l.Def.Title)
				}
			}
		}
	}
	t.Logf("bulls placed: %d", placed)
	// not forced, under go test: no bull at all (other tests' parties stay bare)
	toroForce = false
	g = peLoad(t, toroVenues()[0], false)
	if g.lvl.bullRide() != nil {
		t.Error("a bull without being asked for under go test")
	}
}

// One party in three of its venues; never in STEALTH, the RETO DE HOY, the island or a level of another kind.
func TestToroRollIsSometimes(t *testing.T) {
	g := partyGame(t)
	toroGlobals(t)
	toroInTests = true
	fest := levelDefs[toroVenues()[0]]
	other := levelDefs[campaignIdx("disco")]
	toroSeed, toroRng = 99, nil
	yes := 0
	const n = 3000
	for k := 0; k < n; k++ {
		if g.toroRoll(fest) {
			yes++
		}
		if g.toroRoll(other) {
			t.Fatal("a bull rolled in the disco")
		}
	}
	f := float64(yes) / n
	t.Logf("bull in %.1f %% of the parties of its venues", f*100)
	if f < 0.28 || f > 0.39 {
		t.Errorf("bull chance %.2f, want about 1/3", f)
	}
	g.stealth = true
	for k := 0; k < 200; k++ {
		if g.toroRoll(fest) {
			t.Fatal("a bull in stealth")
		}
	}
	g.stealth = false
	g.dc = &dailyChallenge{}
	for k := 0; k < 200; k++ {
		if g.toroRoll(fest) {
			t.Fatal("a bull in the RETO DE HOY")
		}
	}
	g.dc = nil
}

// toroRun: a party with the bull whose first session starts now (pauses as the test wants).
func toroRun(t *testing.T, idx int, two bool, seed int64) *Game {
	t.Helper()
	toroForce = true
	toroSeed, toroRng = seed, nil
	g := peLoad(t, idx, two)
	if g.lvl.bullRide() == nil {
		return nil
	}
	toroLvl, toroCool, toroNext = g.lvl, map[*Guest]float64{}, 0
	return g
}

func toroAct(g *Game) *toroState {
	if a := g.partyRunning("toro"); a != nil {
		if s, ok := a.D.(*toroState); ok {
			return s
		}
	}
	return nil
}

// The rides: a line of at most four, rides of at most 20 s, falls and holds, riders come back to the line later, nobody
// held when it is over, nobody added (caps) and the game's dice untouched.
func TestToroLineRidesAndRejoins(t *testing.T) {
	partyGame(t)
	toroGlobals(t)
	toroSessMin, toroSessVar, toroCoolMin, toroCoolVar = 170, 0, 4, 2
	best := ""
	for _, two := range []bool{false, true} {
		ok := false
		for seed := int64(1); seed <= 6 && !ok; seed++ {
			for _, idx := range toroVenues() {
				g := toroRun(t, idx, two, seed)
				if g == nil {
					continue
				}
				nGuests, total := len(g.guests), g.total
				touched := map[*Guest]bool{}
				maxLine, maxRide, rides, falls, holds, rejoins, sessions := 0, 0.0, 0, 0, 0, 0, 0
				var last *toroState
				for f := 0; f < 30*200; f++ {
					peStep(g, 1.0/30, true)
					s := toroAct(g)
					if s == nil {
						continue
					}
					if s != last {
						sessions++
						last = s
					}
					peTrack(g, touched)
					maxLine = max(maxLine, len(s.line))
					if s.rider != nil && s.rider.phase == trRide {
						maxRide = math.Max(maxRide, s.rider.t)
					}
					rides, falls, rejoins = s.rides, s.falls, s.rejoins
					holds = s.rides - s.falls
					if f%30 == 0 {
						peDraw(g)
						scr := ebiten.NewImage(screenW, screenH)
						if s.rider != nil {
							s.drawGuest(g, scr, s.rider.mem.gu, s.rider.mem.gu.pos)
						}
						g.drawProp(scr, s.prop)
					}
				}
				if s := toroAct(g); s != nil {
					toroEnd(g, s)
					g.pev.act = nil
				}
				if maxLine > toroLineMax {
					t.Fatalf("a line of %d", maxLine)
				}
				if maxRide > toroRideMax+0.05 {
					t.Fatalf("a ride of %.1f s", maxRide)
				}
				if gu := peStale(g, touched); gu != nil {
					t.Fatalf("a guest is still held after the bull: state %d", gu.state)
				}
				if len(g.guests) != nGuests || g.total != total {
					t.Fatalf("the bull changed the party: guests %d -> %d, total %d -> %d", nGuests, len(g.guests), total, g.total)
				}
				best = "rides " + itoa(rides) + " falls " + itoa(falls) + " rejoins " + itoa(rejoins)
				t.Logf("%s two=%v seed %d: sessions %d, max line %d, max ride %.1f s, rides %d (falls %d, holds %d), rejoins %d",
					g.lvl.Def.Title, two, seed, sessions, maxLine, maxRide, rides, falls, holds, rejoins)
				if rides >= 3 && falls >= 1 && rejoins >= 1 && maxLine >= 2 {
					ok = true
					break
				}
			}
		}
		if !ok {
			t.Errorf("two=%v: never a full session with rides, falls and rejoins (%s)", two, best)
		}
	}
}

// Killing the rider or one of the line never jams it: the line closes up and the next one rides; an alarm ends it.
func TestToroKillsDoNotJam(t *testing.T) {
	partyGame(t)
	toroGlobals(t)
	toroSessMin, toroSessVar = 120, 0
	for seed := int64(1); seed <= 8; seed++ {
		for _, idx := range toroVenues() {
			g := toroRun(t, idx, false, seed)
			if g == nil {
				continue
			}
			var s *toroState
			for f := 0; f < 30*60 && (s == nil || s.rider == nil || s.rider.phase != trRide || len(s.line) < 2); f++ {
				peStep(g, 1.0/30, true)
				s = toroAct(g)
			}
			if s == nil || s.rider == nil || s.rider.phase != trRide || len(s.line) < 2 {
				continue
			}
			touched := map[*Guest]bool{}
			peTrack(g, touched)
			g.killGuest(s.rider.mem.gu, true)
			g.killGuest(s.line[len(s.line)-1].gu, true)
			g.policeCalled, g.cops = false, nil // the test looks at the line, not at the police
			rode := s.rides
			for f := 0; f < 30*40 && toroAct(g) == s; f++ {
				peStep(g, 1.0/30, true)
				g.policeCalled, g.cops = false, nil
				peTrack(g, touched)
				if len(s.line) > toroLineMax {
					t.Fatal("line too long")
				}
				if s.rides > rode {
					break
				}
			}
			if toroAct(g) == s && s.rides == rode {
				// the others may have fled from the murder: then the session must have ended, not jammed
				t.Logf("%s seed %d: after the kills nobody rode (line %d, rider %v)", g.lvl.Def.Title, seed, len(s.line), s.rider != nil)
				for _, m := range s.line {
					if m.mine() && m.gu.state == gIdle && m.gu.timer > 1000 && dist(m.gu.pos, m.pt) > 6 {
						t.Fatal("a guest of the line is stuck away from his place")
					}
				}
			}
			// an alarm ends the session at once and lets everybody go
			g.policeCalled = true
			peStep(g, 1.0/30, true)
			if toroAct(g) != nil {
				t.Fatal("the session goes on with the police called")
			}
			g.policeCalled = false
			if gu := peStale(g, touched); gu != nil {
				t.Fatalf("a guest is still held after the alarm: state %d", gu.state)
			}
			return
		}
	}
	t.Error("never a ride with a line to kill in")
}

// -pevent toro:fall puts a bull in the level given and the rider falls at once.
func TestToroDebugFlag(t *testing.T) {
	partyGame(t)
	toroGlobals(t)
	dbgPartyEvent = "toro:fall"
	if partyByID("toro:fall") == nil {
		t.Fatal("-pevent toro:fall not known")
	}
	disco := levelDefs[campaignIdx("disco")]
	g := partyGame(t)
	dbgPartyEvent = "toro:fall"
	if !g.toroRoll(disco) {
		t.Fatal("-pevent toro does not ask for a bull in the level given")
	}
	// peLoad starts a new party game (and clears the flag): the bull is forced there and the flag set again
	toroForce = true
	g = peLoad(t, campaignIdx("festival"), false) // (v1.4.0: the ranch became BARCELONA, no mechanical bull there)
	dbgPartyEvent = "toro:fall"
	toroLvl = nil // the first session starts 3 s into the party (toroDue with the flag)
	if g.lvl.bullRide() == nil {
		t.Fatal("no bull in the festival")
	}
	fell := false
	for f := 0; f < 30*40 && !fell; f++ {
		peStep(g, 1.0/30, true)
		if s := toroAct(g); s != nil {
			fell = s.falls > 0
			if s.rider != nil && s.rider.phase == trRide && s.rider.t > 3.1 {
				t.Fatal("toro:fall ride longer than 3 s")
			}
		}
	}
	if !fell {
		t.Error("no fall with -pevent toro:fall")
	}
	if s := toroAct(g); s != nil {
		toroEnd(g, s)
		g.pev.act = nil
	}
}

func TestToroAndMurgaTextsAndNews(t *testing.T) {
	for _, ss := range [][]string{toroJoin, toroFull, toroCheer, toroRiderL, toroFallL, toroWin, murgaLines, murgaFollow, murgaBye} {
		for _, s := range ss {
			if len(s) > 48 || langDataEN[s] == "" {
				t.Errorf("%q: too long or no English", s)
			}
		}
	}
	for _, id := range []string{"toro", "murga"} {
		p, ok := paperEventPools[id]
		if !ok || len(p.pool) < 3 {
			t.Fatalf("%s: no newspaper briefs", id)
		}
		for _, x := range p.pool {
			if langDataEN[x.es] != x.en {
				t.Errorf("%q not in English", x.es)
			}
		}
		if partyByID(id) == nil {
			t.Errorf("%s not registered", id)
		}
	}
	if e := partyByID("toro"); !e.Reactive || !e.Minor {
		t.Error("toro must be a reactive minor event (it never takes a place of the frame)")
	}
}
