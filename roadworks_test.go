package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the roadworks and their crane (roadworks.go, crane.go) with the real Game after loadLevel.

// rwGlobals: every global of the works is put back after the test.
func rwGlobals(t *testing.T) {
	t.Helper()
	pf, pi, po, pr, ps, prun, pnone := rwForce, rwInTests, rwOff, rwRng, rwSeed, rw, rwNoneAt
	t.Cleanup(func() {
		rwForce, rwInTests, rwOff, rwRng, rwSeed, rw, rwNoneAt = pf, pi, po, pr, ps, prun, pnone
		rwDecision = false
	})
}

// rwVenues: the campaign levels where the works may come.
func rwVenues() []int {
	var out []int
	for i := range campaignLevels {
		if i < len(levelDefs) && rwFits(levelDefs[i]) {
			out = append(out, i)
		}
	}
	return out
}

// rwLoad: a party of level idx with the works forced (nil if they found no spot).
func rwLoad(t *testing.T, idx int, two bool, seed int64) *Game {
	t.Helper()
	rwForce = true
	rwSeed, rwRng, rw, rwNoneAt = seed, nil, nil, nil
	g := peLoad(t, idx, two)
	g.policeCalled, g.cops = false, nil
	if g.rwActive() == nil {
		return nil
	}
	return g
}

// rwStep: one frame of the party with the works, the guests and the street traffic.
func rwStep(g *Game, dt float64) {
	g.time += dt
	g.updateParty(dt)
	g.updateWorks(dt)
	for _, gu := range g.guests {
		g.updateGuest(gu, dt)
	}
	g.updateAmbient(dt)
}

// rwOwn: the props of the works.
func rwOwn(name string) bool {
	return name == rwHoleName || name == rwConeName || name == rwBarName || name == craneName
}

// The works come only in the levels with the street (forced: they find a spot in nearly all of them), never elsewhere;
// they keep every floor cell and every place of use reachable from the start, their places are open, the cones stand
// in the road edge and the crane's base on the pavement.
func TestRoadworksOnlyOnStreetsAndReachable(t *testing.T) {
	g := partyGame(t)
	rwGlobals(t)
	venues := map[int]bool{}
	for _, i := range rwVenues() {
		venues[i] = true
	}
	if len(venues) < 10 {
		t.Fatalf("only %d campaign levels for the works", len(venues))
	}
	rwForce = true
	placed, tried := 0, 0
	for seed := int64(1); seed <= 2; seed++ {
		for i := range campaignLevels {
			rwSeed, rwRng, rw, rwNoneAt = seed*37+int64(i), nil, nil, nil
			g = peLoad(t, i, false)
			s := g.lvl.rwSiteOf()
			if !venues[i] {
				if s != nil {
					t.Errorf("%s: roadworks in a level without street", g.lvl.Def.Title)
				}
				continue
			}
			tried++
			if s == nil {
				t.Logf("%s (seed %d): forced works found no spot", g.lvl.Def.Title, seed)
				continue
			}
			placed++
			l := g.lvl
			d := l.Def
			if s.crane == nil {
				t.Fatalf("%s: works without a crane", d.Title)
			}
			// the same level without the works: nothing that was reachable is cut off
			with := l.reachFrom(l.NearestFree(d.Start))
			props, sp, sn := l.Props, l.solidsP, l.solidsN
			var np []*Prop
			for _, o := range props {
				if !rwOwn(o.Name) {
					np = append(np, o)
				}
			}
			mine := map[Rect]int{}
			for _, o := range props {
				if rwOwn(o.Name) && !o.K.NoSolid {
					mine[o.Foot]++
				}
			}
			drop := func(rs []Rect) []Rect {
				left := map[Rect]int{}
				for k, v := range mine {
					left[k] = v
				}
				var out []Rect
				for _, r := range rs {
					if left[r] > 0 {
						left[r]--
						continue
					}
					out = append(out, r)
				}
				return out
			}
			l.Props, l.solidsP, l.solidsN = np, drop(sp), drop(sn)
			l.initNav()
			without := l.reachFrom(l.NearestFree(d.Start))
			l.Props, l.solidsP, l.solidsN = props, sp, sn
			l.initNav()
			for c := range without {
				if without[c] && !with[c] && !l.blocked[c] {
					t.Errorf("%s: the works cut off cell %d", d.Title, c)
					break
				}
			}
			// every place of use and every performer is still reachable from the start
			for kind, ps := range l.POIs {
				for _, p := range ps {
					if q := l.NearestFree(p); !with[l.navIdx(q)] && without[l.navIdx(q)] {
						t.Errorf("%s: %s at %v not reachable any more", d.Title, kind, p)
					}
				}
			}
			for _, p := range append(s.spots(), s.load(), craneUseAt(Vec{s.crane.X, s.crane.Y}, s.sd, s.cs)) {
				if !l.rwDry(p) || !with[l.navIdx(p)] {
					t.Errorf("%s: a place of the works is closed: %v", d.Title, p)
				}
				if math.Abs(p.Y-s.lane) < 14 {
					t.Errorf("%s: a place of the works is in the lane of the cars: %v (lane %.0f)", d.Title, p, s.lane)
				}
			}
			if dist(Vec{s.hole.X, s.edge}, d.Start) < 120 {
				t.Errorf("%s: the works are at the start", d.Title)
			}
			for _, o := range l.Props {
				if o.Name == rwConeName && (o.Y-s.edge)*s.sd > 0 {
					t.Errorf("%s: a cone on the pavement", d.Title)
				}
				if o.Name == craneName && (o.Y-s.edge)*s.sd < 0 {
					t.Errorf("%s: the crane in the road", d.Title)
				}
			}
			for _, dr := range d.Dance {
				if dr.Grow(10).Has(s.hole.X, s.edge) {
					t.Errorf("%s: the works on the dance floor", d.Title)
				}
			}
		}
	}
	t.Logf("works placed in %d of %d forced parties", placed, tried)
	if placed*10 < tried*8 {
		t.Errorf("the works found a spot in only %d of %d forced parties", placed, tried)
	}
	// not forced, under go test: no works at all (other tests' parties stay bare)
	rwForce = false
	rw, rwNoneAt = nil, nil
	g = peLoad(t, rwVenues()[0], false)
	if g.rwActive() != nil {
		t.Error("roadworks without being asked for under go test")
	}
}

// About one party in 3-4 of the street levels; never in STEALTH, the RETO DE HOY, the island or a level without street.
func TestRoadworksRollIsSometimes(t *testing.T) {
	g := partyGame(t)
	rwGlobals(t)
	rwInTests = true
	street := levelDefs[rwVenues()[0]]
	var other *LevelDef
	for i := range campaignLevels {
		if !rwFits(levelDefs[i]) {
			other = levelDefs[i]
			break
		}
	}
	if other == nil {
		t.Fatal("every level has a street?")
	}
	rwSeed, rwRng = 4321, nil
	yes := 0
	const n = 4000
	for k := 0; k < n; k++ {
		if g.rwRoll(street) {
			yes++
		}
		if g.rwRoll(other) {
			t.Fatalf("roadworks rolled in %s", other.Title)
		}
	}
	f := float64(yes) / n
	t.Logf("roadworks in %.1f %% of the street parties", f*100)
	if f < 0.24 || f > 0.36 {
		t.Errorf("roadworks chance %.2f, want one in 3-4", f)
	}
	g.stealth = true
	for k := 0; k < 200; k++ {
		if g.rwRoll(street) {
			t.Fatal("roadworks in stealth")
		}
	}
	g.stealth = false
	g.dc = &dailyChallenge{}
	for k := 0; k < 200; k++ {
		if g.rwRoll(street) {
			t.Fatal("roadworks in the RETO DE HOY")
		}
	}
	g.dc = nil
	if g.rwRoll(jurassicLevel()) {
		t.Fatal("roadworks on the island")
	}
}

// 3-4 workers taken from the guests (nobody added: 1 and 2 players), dressed as workers, at their places and drilling;
// the party events do not take them.
func TestRoadworksWorkersFromTheGuests(t *testing.T) {
	partyGame(t)
	rwGlobals(t)
	for _, two := range []bool{false, true} {
		done := false
		for seed := int64(1); seed <= 4 && !done; seed++ {
			for _, idx := range rwVenues() {
				g := rwLoad(t, idx, two, seed)
				if g == nil {
					continue
				}
				nGuests, total := len(g.guests), g.total
				drilled := map[*Guest]bool{}
				for f := 0; f < 30*30; f++ {
					rwStep(g, 1.0/30)
					g.policeCalled, g.cops = false, nil
					for _, w := range rw.ws {
						if rw.drilling(w.m.gu) {
							drilled[w.m.gu] = true
							if g.worksJitter(w.m.gu) != 0 && g.worksJitter(w.m.gu) != 1 {
								t.Fatal("the shake is not a whole pixel")
							}
						}
						if !w.gone && g.pevFreeGuest(w.m.gu) {
							t.Fatal("a party event could take a worker")
						}
					}
				}
				run := rw
				if n := len(run.ws); n < 3 || n > 4 {
					t.Fatalf("%d workers", n)
				}
				for _, w := range run.ws {
					sp := w.m.gu.cs.Spec
					if sp == nil || sp.Hat != HatHardhat || sp.Top != TopVest {
						t.Fatal("a worker without hard hat and vest")
					}
				}
				if len(g.guests) != nGuests || g.total != total {
					t.Fatalf("the works changed the party: guests %d -> %d, total %d -> %d", nGuests, len(g.guests), total, g.total)
				}
				t.Logf("%s two=%v seed %d: %d workers, %d drilled", g.lvl.Def.Title, two, seed, len(run.ws), len(drilled))
				if len(drilled) >= 2 {
					done = true
					break
				}
			}
		}
		if !done {
			t.Errorf("two=%v: never two workers drilling", two)
		}
	}
}

// With heavy traffic the cars move aside by the works and never run a worker over, not even one standing in the lane.
func TestRoadworksCarsDoNotRunOverWorkers(t *testing.T) {
	partyGame(t)
	rwGlobals(t)
	checked := 0
	for _, idx := range rwVenues() {
		g := rwLoad(t, idx, false, 3)
		if g == nil {
			continue
		}
		prevTank, prevGod := g.tankDue, g.god
		g.tankDue, g.tankN, g.god = false, 9, true // the player (standing at the start, on the pavement) is not the test
		t.Cleanup(func() { g.tankDue, g.god = prevTank, prevGod })
		swerved, passed := 0, map[*Ambient]bool{}
		for f := 0; f < 30*90; f++ {
			if g.ambT > 1.2 {
				g.ambT = 1.2 // a car every second or so
			}
			rwStep(g, 1.0/30)
			g.policeCalled, g.cops = false, nil
			if g.mode != ModePlay {
				t.Fatalf("%s: the party stopped (mode %d)", g.lvl.Def.Title, g.mode)
			}
			swerved = max(swerved, len(rw.swerve))
			for a := range rw.swerve {
				if math.Abs(a.x-rw.site.hole.X) < 20 {
					passed[a] = true
				}
			}
		}
		for _, w := range rw.ws {
			if w.m.gu.dead {
				t.Fatalf("%s: a worker died with the traffic", g.lvl.Def.Title)
			}
		}
		if len(passed) < 3 {
			t.Fatalf("%s: only %d cars passed the works", g.lvl.Def.Title, len(passed))
		}
		// a worker who walks into the lane: the cars wait for him
		w := rw.ws[0]
		gu := w.m.gu
		gu.pos = Vec{rw.site.hole.X, rw.site.lane}
		w.m.hold(gIdle)
		for f := 0; f < 30*25; f++ {
			if g.ambT > 1.2 {
				g.ambT = 1.2
			}
			gu.pos = Vec{rw.site.hole.X, rw.site.lane}
			w.m.keep()
			rwStep(g, 1.0/30)
			g.policeCalled, g.cops = false, nil
			if gu.dead {
				for _, a := range g.amb {
					t.Logf("car %s kind %d x %.0f y %.0f vx %.0f wait %.2f", a.name, a.kind, a.x, a.y, a.vx, a.wait)
				}
				t.Logf("worker at %v, roads %d, mode %d (play %d), p dead %v", gu.pos, len(g.lvl.Def.Roads), g.mode, ModePlay, g.p.dead)
				t.Fatalf("%s: a car ran over the worker in the lane", g.lvl.Def.Title)
			}
		}
		t.Logf("%s: %d cars passed the works, up to %d moved aside at once", g.lvl.Def.Title, len(passed), swerved)
		checked++
		if checked >= 3 {
			break
		}
	}
	if checked == 0 {
		t.Fatal("no level with works to check")
	}
}

// Killing a worker: the others take fright and later go back to work; nobody is left held away from his place.
func TestRoadworksKillWorkerNoJam(t *testing.T) {
	partyGame(t)
	rwGlobals(t)
	for seed := int64(1); seed <= 6; seed++ {
		for _, idx := range rwVenues() {
			g := rwLoad(t, idx, false, seed)
			if g == nil {
				continue
			}
			for f := 0; f < 30*5; f++ {
				rwStep(g, 1.0/30)
			}
			run := rw
			victim := run.ws[0].m.gu
			g.killGuest(victim, true)
			back := false
			for f := 0; f < 30*90; f++ {
				rwStep(g, 1.0/30)
				g.policeCalled, g.cops = false, nil
				for _, w := range run.ws[1:] {
					if !w.gone && !w.away && w.m.mine() && w.m.want == gFun {
						back = true
					}
					if !w.gone && w.away && peHeld(w.m.gu) && w.m.gu.state == gIdle {
						t.Fatal("a frightened worker is held by the works")
					}
					if !w.gone && !w.away && w.m.want == gGoto && g.time-w.walkT0 > rwWalkMax+1 {
						t.Fatal("a worker walks back for ever")
					}
				}
			}
			if !run.ws[0].gone || !run.ws[0].mourned {
				t.Fatal("the dead worker is still on the list")
			}
			alive := 0
			for _, w := range run.ws[1:] {
				if !w.gone {
					alive++
				}
			}
			if back {
				t.Logf("%s seed %d: the others went back to work (%d alive)", g.lvl.Def.Title, seed, alive)
				return
			}
			t.Logf("%s seed %d: nobody back to work after the kill (%d alive)", g.lvl.Def.Title, seed, alive)
		}
	}
	t.Error("after a kill the workers never went back to work")
}

// The crane: the label and the action share their reach, the load crushes only who is right under it (a trap kill of
// the player: credit and "traps"), once; the parts draw in every phase without panicking.
func TestRoadworksCraneDropsOnce(t *testing.T) {
	partyGame(t)
	rwGlobals(t)
	for _, two := range []bool{false, true} {
		g := (*Game)(nil)
		for _, idx := range rwVenues() {
			if g = rwLoad(t, idx, two, 5); g != nil {
				break
			}
		}
		if g == nil {
			t.Fatal("no works anywhere")
		}
		prevTraps := g.progress.Stats["traps"]
		t.Cleanup(func() { g.progress.Stats["traps"] = prevTraps })
		for f := 0; f < 30*2; f++ {
			rwStep(g, 1.0/30)
		}
		run := rw
		s := run.site
		scr := ebiten.NewImage(screenW, screenH)
		draw := func() {
			for _, it := range g.worksItems(scr, nil) {
				it.fn()
			}
			for _, pr := range g.lvl.Props {
				if rwOwn(pr.Name) {
					g.drawProp(scr, pr)
				}
			}
		}
		draw()
		pl := g.p
		if two {
			pl = g.p2
		}
		// far from the crane: no label, no action
		far := g.lvl.Def.Start
		pl.pos = far
		g.withPlayer(pl, func() {
			if lab, _ := g.useLabel(s.crane); lab != "" {
				t.Fatalf("crane label from far away: %q", lab)
			}
		})
		if g.craneInteract(pl) {
			t.Fatal("dropped from far away")
		}
		pl.pos = g.lvl.NearestFree(craneUseAt(Vec{s.crane.X, s.crane.Y}, s.sd, s.cs))
		g.withPlayer(pl, func() {
			if lab, _ := g.useLabel(s.crane); lab != "Soltar la carga" {
				t.Fatalf("crane label %q", lab)
			}
			if g.useTextProp() != s.crane {
				t.Fatal("the text of another prop wins beside the crane")
			}
		})
		if distRect(pl.pos, s.crane.Foot) >= craneReach {
			t.Fatal("the place to use the crane is out of reach")
		}
		// one guest right under the load, one a step aside
		L := s.load()
		var under, aside *Guest
		for _, gu := range g.guests {
			if gu.out() || rwIsWorker(gu) || gu.perf != "" {
				continue
			}
			if under == nil {
				under = gu
			} else if aside == nil {
				aside = gu
				break
			}
		}
		for _, w := range run.ws { // the worker who stands there is under it too
			if dist(w.m.gu.pos, L) < craneHitR && !w.m.gu.out() {
				w.m.gu.pos = Vec{L.X + 40, L.Y}
			}
		}
		under.pos, aside.pos = L, Vec{L.X + craneHitR + 10, L.Y}
		under.state, aside.state, under.timer, aside.timer = gIdle, gIdle, 9999, 9999
		kills := g.progress.Stats["traps"]
		if !g.craneInteract(pl) {
			t.Fatal("the crane did not answer")
		}
		if g.craneInteract(pl) {
			t.Fatal("dropped twice")
		}
		for f := 0; f < 30 && run.crane.phase != crDown; f++ {
			under.pos, aside.pos = L, Vec{L.X + craneHitR + 10, L.Y}
			g.time += 1.0 / 30
			g.updateWorks(1.0 / 30)
			draw()
		}
		if run.crane.phase != crDown {
			t.Fatal("the load never landed")
		}
		if !under.dead {
			t.Fatal("the guest under the load is alive")
		}
		if aside.dead {
			t.Fatal("the guest beside the load died")
		}
		if g.progress.Stats["traps"] != kills+1 {
			t.Fatalf("traps stat %d -> %d", kills, g.progress.Stats["traps"])
		}
		if pl.dead {
			t.Fatal("the player at the crane died")
		}
		g.withPlayer(pl, func() {
			if lab, _ := g.useLabel(s.crane); lab != "" {
				t.Fatalf("label after the drop: %q", lab)
			}
		})
		if g.craneInteract(pl) {
			t.Fatal("a second drop")
		}
		draw()
	}
}

// -roadworks (or -pevent obras) forces the works where they fit, even under go test; obras:drop drops the load by itself.
func TestRoadworksDebugFlag(t *testing.T) {
	partyGame(t)
	rwGlobals(t)
	prevFlag := *rwFlag
	t.Cleanup(func() { *rwFlag = prevFlag })
	*rwFlag = true
	for _, idx := range rwVenues() {
		rwSeed, rwRng, rw, rwNoneAt = 9, nil, nil, nil
		g := peLoad(t, idx, false) // partyGame clears dbgPartyEvent: the stage is set after it
		dbgPartyEvent = "obras:drop"
		if g.rwActive() == nil {
			continue
		}
		for f := 0; f < 30*4; f++ {
			rwStep(g, 1.0/30)
			g.policeCalled, g.cops = false, nil
		}
		if rw.crane.phase == crReady {
			t.Fatal("obras:drop did not drop the load")
		}
		if distRect(g.p.pos, rw.site.crane.Foot) >= craneReach {
			t.Fatal("the player was not put beside the crane")
		}
		return
	}
	t.Fatal("-roadworks forced no works")
}

// The texts are short and in both languages; the props draw.
func TestRoadworksTextsAndProps(t *testing.T) {
	for _, set := range [][]string{rwWarn, rwChat, rwMourn, rwBack, rwDodge, craneLabelES} {
		for _, s := range set {
			if len(s) > 48 {
				t.Errorf("%q longer than 48 bytes", s)
			}
			if en, ok := langDataEN[s]; !ok || en == "" || en == s {
				t.Errorf("no English for %q", s)
			}
		}
	}
	for _, n := range []string{rwHoleName, rwConeName, rwBarName, craneName} {
		k := propKinds[n]
		if k == nil {
			t.Fatalf("no prop %s", n)
		}
		for v := 0; v < max(1, k.Vars); v++ {
			b := newPix(k.W, k.H)
			k.Draw(b, v)
		}
	}
}
