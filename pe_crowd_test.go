package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"
)

// Tests of the crowd events: waiters, drunks, philosopher, gossiping granny, karaoke and hiding under the tables.

var crowdIDs = []string{"waiters", "drunks", "philosopher", "granny", "karaoke"}

// crowdSnap: the guests' gaits before an event, to see them put back.
func crowdGaits(g *Game) map[*Guest]string {
	m := map[*Guest]string{}
	for _, gu := range g.guests {
		m[gu] = gu.gait
	}
	return m
}

// crowdTrack notes the guests THIS event has taken (other events of the party, written by others, may hold their own guests).
func crowdTrack(g *Game, id string, touched map[*Guest]bool) {
	a := g.partyRunning(id)
	if a == nil {
		return
	}
	h, ok := a.D.(interface{ pevHas(*Guest) bool })
	if !ok {
		return
	}
	for _, gu := range g.guests {
		if h.pevHas(gu) {
			touched[gu] = true
		}
	}
}

// crowdStart: the event running in a fresh party, with the marks of earlier tests cleared (and again at the end).
func crowdStart(t *testing.T, id string, two bool) *Game {
	t.Helper()
	clear := func() {
		for k := range pevFxs {
			delete(pevFxs, k)
		}
	}
	clear()
	t.Cleanup(clear)
	return peStart(t, id, two)
}

func TestPevCrowdTextsAreComplete(t *testing.T) {
	if len(philoLines) < 20 {
		t.Errorf("the philosopher has %d sayings, 20 needed", len(philoLines))
	}
	if len(grannyLines) < 12 {
		t.Errorf("the granny has %d whispers, 12 needed", len(grannyLines))
	}
	all := [][]string{waitersLines, drunksLines, philoLines, philoNods, grannyLines, grannyScold, karaokeLines, karaokeCrowd, karaokeSign, hideLines, hideOutLines}
	seen := map[string]bool{}
	for _, ss := range all {
		for _, s := range ss {
			if len(s) > 48 {
				t.Errorf("%q is longer than 48 bytes", s)
			}
			en, ok := langDataEN[s]
			if !ok || en == "" {
				t.Errorf("no English for %q", s)
			}
			if len(en) > 48 {
				t.Errorf("the English of %q is longer than 48 bytes: %q", s, en)
			}
			if !strings.HasPrefix(s, "...") && en == s && len([]rune(s)) > 14 {
				t.Errorf("%q is not translated", s)
			}
			seen[s] = true
		}
	}
	// the philosopher never repeats a saying in a row and has no duplicates
	m := map[string]bool{}
	for _, s := range philoLines {
		if m[s] {
			t.Errorf("duplicated saying %q", s)
		}
		m[s] = true
	}
	// no brand or real person
	for s := range seen {
		low := strings.ToLower(s + " " + langDataEN[s])
		for _, b := range []string{"disney", "mickey", "coca", "pepsi", "google", "netflix"} {
			if strings.Contains(low, b) {
				t.Errorf("%q mentions %q", s, b)
			}
		}
	}
}

func TestPevCrowdRegisteredAndEligible(t *testing.T) {
	g := partyGame(t)
	list := partyList()
	n := len(campaignLevels)
	for _, id := range append(append([]string{}, crowdIDs...), "tablehide") {
		e := partyByID(id)
		if e == nil {
			t.Fatalf("%s is not registered", id)
		}
		if !strings.Contains(list, id) {
			t.Errorf("%s is not listed in -pevents", id)
		}
		if e.Reactive {
			continue
		}
		if e.Prob < 0.10 || e.Prob > 0.40 || e.AtMax <= e.AtMin || e.Dur <= 0 {
			t.Errorf("%s: bad chance or window", id)
		}
		in := peEligible(g, id)
		t.Logf("%-11s fits %2d of %d campaign levels", id, len(in), n)
		if len(in) == 0 || len(in) > n*2/3 {
			t.Errorf("%s should fit some campaign levels, not all (%d of %d)", id, len(in), n)
		}
		if e.Eligible(g, partyTraits{}) {
			t.Errorf("%s fits an empty level", id)
		}
		for _, i := range in {
			tr := partyTraitsOf(levelDefs[i], i)
			switch id {
			case "philosopher":
				if !tr.indoor || !tr.bar {
					t.Errorf("philosopher fits level %d (%s) without an indoor bar", i, tr.icon)
				}
			case "karaoke":
				if !tr.stage && !karaokeIcons[tr.icon] {
					t.Errorf("karaoke fits level %d (%s) without a stage", i, tr.icon)
				}
			case "waiters", "drunks":
				if !tr.bar && !waitersIcons[tr.icon] {
					t.Errorf("%s fits level %d (%s)", id, i, tr.icon)
				}
			}
		}
	}
	if e := partyByID("karaoke"); e.Group != "stage" || e.Big {
		t.Error("karaoke is of the stage group")
	}
	if e := partyByID("tablehide"); !e.Reactive {
		t.Error("hiding under the tables follows the danger: reactive")
	}
	// the parties with tables are only a few
	cnt := 0
	for i := range campaignLevels {
		if tablesIcons[campaignLevels[i].icon] > 0 {
			cnt++
		}
	}
	if cnt == 0 || cnt > n/3 {
		t.Errorf("%d of %d parties have tables", cnt, n)
	}
}

// Each event starts where it fits, runs to its end and leaves the guests free and as they were.
func TestPevCrowdRunsAndReleasesTheGuests(t *testing.T) {
	for _, id := range crowdIDs {
		id := id
		t.Run(id, func(t *testing.T) {
			g := crowdStart(t, id, false)
			if g.partyRunning(id) == nil {
				t.Fatal("not running")
			}
			touched := map[*Guest]bool{}
			sawHeld := false
			for s := 0; s < 30*220 && g.partyRunning(id) != nil; s++ {
				peStep(g, 1.0/30, true)
				if s%10 == 0 {
					peDraw(g)
				}
				crowdTrack(g, id, touched)
				for gu := range touched {
					if peHeld(gu) {
						sawHeld = true
					}
				}
			}
			if g.partyRunning(id) != nil {
				t.Fatal("it did not finish in 220 s")
			}
			if !sawHeld && id != "drunks" {
				t.Error("it never held a guest")
			}
			if gu := peStale(g, touched); gu != nil {
				t.Errorf("a guest is still held at the end (state %d timer %.0f dur %.0f fun %d)", gu.state, gu.timer, gu.dur, gu.fun)
			}
			if len(touched) > len(g.guests)/2 {
				t.Errorf("%d of %d guests were taken: too many", len(touched), len(g.guests))
			}
			for gu, f := range pevFxs {
				if !f.granny || f.sway != 0 || f.trip != 0 || f.hid != nil {
					t.Errorf("a mark is left on a guest: %+v", *f)
				}
				if gu.gait == "hips" || gu.gait == "glide" {
					t.Errorf("a guest keeps the style %q", gu.gait)
				}
			}
			if id != "granny" && len(pevFxs) != 0 {
				t.Errorf("%d marks left", len(pevFxs))
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

func TestPevCrowdDoesNotMoveGameRng(t *testing.T) {
	for _, id := range crowdIDs {
		g := crowdStart(t, id, false)
		g.rng = rand.New(rand.NewSource(99))
		for s := 0; s < 30*45; s++ {
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

func TestPevCrowdTwoPlayers(t *testing.T) {
	for _, id := range crowdIDs {
		g := crowdStart(t, id, true)
		if !g.is2P() || g.p2 == nil {
			t.Fatalf("%s: not a two player game", id)
		}
		g.p2.pos = Vec{g.p.pos.X + 20, g.p.pos.Y}
		touched := map[*Guest]bool{}
		for s := 0; s < 30*120; s++ {
			peStep(g, 1.0/30, true)
			if s%15 == 0 {
				peDraw(g)
			}
			crowdTrack(g, id, touched)
		}
		if gu := peStale(g, touched); gu != nil && g.partyRunning(id) == nil {
			t.Errorf("%s: a guest is held after the end (state %d timer %.0f dur %.0f next %d fun %d)", id, gu.state, gu.timer, gu.dur, gu.next, gu.fun)
		}
	}
}

// With many corpses around the guests panic and leave the events: nobody is left held.
func TestPevCrowdWithManyCorpses(t *testing.T) {
	for _, id := range crowdIDs {
		g := crowdStart(t, id, false)
		for i := 0; i < 24; i++ {
			g.killByPlayer = true
			g.spawnBody(Vec{g.p.pos.X + float64(i%6)*12 - 30, g.p.pos.Y + 20 + float64(i/6)*10}, g.guests[0].cs, DirDown)
			g.killByPlayer = false
		}
		touched := map[*Guest]bool{}
		for s := 0; s < 30*150; s++ {
			peStep(g, 1.0/30, true)
			if s%20 == 0 {
				peDraw(g)
			}
			crowdTrack(g, id, touched)
		}
		if gu := peStale(g, touched); gu != nil {
			t.Errorf("%s: a guest is held after 150 s with corpses (state %d timer %.0f dur %.0f next %d fun %d running %v)", id, gu.state, gu.timer, gu.dur, gu.next, gu.fun, g.partyRunning(id) != nil)
		}
		if g.partyRunning(id) == nil {
			for _, gu := range g.guests {
				if f := pevFxs[gu]; f != nil && (f.sway != 0 || f.trip != 0 || f.hid != nil) {
					t.Errorf("%s: a mark is left", id)
				}
			}
		}
	}
}

// ---- waiters ------------------------------------------------------------------------------------------------

func TestPevWaiterTrayServesAndTrips(t *testing.T) {
	peRetry(t, "waiters", false, func(g *Game) string {
		st := g.partyRunning("waiters").D.(*waitersState)
		w := st.w[0]
		sawTray, drinks := false, 0
		drinkers := map[*Guest]bool{}
		for s := 0; s < 30*70; s++ {
			peStep(g, 1.0/30, true)
			if (w.phase == wGo || w.phase == wOffer) && len(w.tray) >= 3 && len(w.tray) <= 5 {
				sawTray = true
			}
			if len(w.tray) > 5 {
				return "tray over 5"
			}
			for _, gu := range g.guests {
				if gu.state == gDrink && gu.dur == 1.9 && !drinkers[gu] {
					drinkers[gu] = true
					drinks++
				}
			}
			if sawTray && drinks >= 1 {
				break
			}
		}
		if !sawTray {
			return "no tray of 3-5 things seen"
		}
		if drinks < 1 {
			return "nobody took a glass"
		}
		return ""
	})
}

func TestPevWaiterTripsAndNobodyDies(t *testing.T) {
	peRetry(t, "waiters", false, func(g *Game) string {
		st := g.partyRunning("waiters").D.(*waitersState)
		w := st.w[0]
		w.tripAfter = 1
		fell := false
		for s := 0; s < 30*80 && !fell; s++ {
			peStep(g, 1.0/30, true)
			fell = w.phase == wFall
		}
		if !fell {
			return "never tripped"
		}
		if len(w.tray) != 0 || len(st.shards) == 0 {
			return "the tray did not fall"
		}
		nb := len(g.bodies)
		for s := 0; s < 30*6; s++ {
			peStep(g, 1.0/30, false) // only the party: the game's own deaths are not ours
			peDraw(g)
		}
		if len(g.bodies) != nb {
			return "a body"
		}
		for i := 0; i < 30*40 && w.phase != wLoad && w.phase != wGo; i++ {
			peStep(g, 1.0/30, true)
		}
		if w.phase != wLoad && w.phase != wGo {
			return "he did not go back to restock"
		}
		return ""
	})
}

func TestPevWaiterPlayerTakesAGlassOnce(t *testing.T) {
	g := crowdStart(t, "waiters", false)
	st := g.partyRunning("waiters").D.(*waitersState)
	w := st.w[0]
	w.phase, w.t0 = wOffer, g.time
	w.tray = []int{trayGlass, trayWine, trayCanape, trayOlive}
	w.pf.pos = Vec{g.p.pos.X + 12, g.p.pos.Y}
	pts := g.points
	if !g.pevInteract(g.p) {
		t.Fatal("E beside a waiter with a full tray does nothing")
	}
	if g.points != pts+wtrBonus || len(w.tray) != 3 {
		t.Errorf("first glass: points %d -> %d, tray %d", pts, g.points, len(w.tray))
	}
	if !g.pevInteract(g.p) || g.points != pts+wtrBonus || len(w.tray) != 2 {
		t.Errorf("second glass must be free of bonus: points %d, tray %d", g.points, len(w.tray))
	}
	w.tray = nil
	if g.pevInteract(g.p) {
		t.Error("an empty tray gives nothing")
	}
	w.tray = []int{trayGlass}
	w.pf.pos = Vec{g.p.pos.X + 90, g.p.pos.Y}
	if g.pevInteract(g.p) {
		t.Error("a waiter far away gives nothing")
	}
}

// ---- drunks -------------------------------------------------------------------------------------------------

func TestPevDrunkWeaveTripAndRestore(t *testing.T) {
	peRetry(t, "drunks", false, func(g *Game) string {
		st := g.partyRunning("drunks").D.(*drunksState)
		if len(st.d) < 3 || len(st.d) > 8 {
			return "wrong number of drunks"
		}
		orig := map[*Guest]string{}
		for _, d := range st.d {
			orig[d.m.gu] = d.gait
			f := pevFxs[d.m.gu]
			if f == nil || f.sway <= 0 || d.m.gu.gait != "stagger" {
				return "a drunk without sway or staggering walk"
			}
		}
		// the weave is only drawn: the position is the game's
		scr := newTestImage()
		moved := false
		for s := 0; s < 30*10 && !moved; s++ {
			peStep(g, 1.0/30, true)
			for _, d := range st.d {
				gu := d.m.gu
				if gu.moving {
					pos := gu.pos
					before := gu.pos
					if g.pevGuestDraw(scr, gu, &pos) {
						continue
					}
					if gu.pos != before || math.Abs(pos.X-gu.pos.X) > 5 || pos.Y != gu.pos.Y {
						return "the weave changed the position or went too far"
					}
					if pos.X != gu.pos.X {
						moved = true
					}
				}
			}
		}
		if !moved {
			return "never saw a weaving walker"
		}
		// force a trip on someone who walks
		var tripper *drunkOne
		for s := 0; s < 30*40 && tripper == nil; s++ {
			for _, d := range st.d {
				d.nextTrip = 0
			}
			peStep(g, 1.0/30, true)
			for _, d := range st.d {
				if f := pevFxs[d.m.gu]; f != nil && g.time < f.trip {
					tripper = d
				}
			}
		}
		if tripper == nil {
			return "nobody tripped"
		}
		gu := tripper.m.gu
		pos := gu.pos
		if !g.pevGuestDraw(scr, gu, &pos) {
			return "a tripped guest must be drawn lying"
		}
		p0 := gu.pos
		for s := 0; s < 30*3; s++ {
			peStep(g, 1.0/30, true)
			peDraw(g)
		}
		if dist(gu.pos, p0) > 30 {
			return "the tripped guest travelled"
		}
		if f := pevFxs[gu]; f != nil && g.time < f.trip {
			return "still on the floor"
		}
		if gu.state == gIdle && gu.timer > 1000 && g.pevTaken(gu) == false {
			return "left held after the trip"
		}
		// the end
		for s := 0; s < 30*80 && g.partyRunning("drunks") != nil; s++ {
			peStep(g, 1.0/30, true)
		}
		for gu, og := range orig {
			if gu.gait != og {
				return "gait not restored"
			}
		}
		if len(pevFxs) != 0 {
			return "marks left"
		}
		return ""
	})
}

// ---- philosopher --------------------------------------------------------------------------------------------

func TestPevPhilosopherLeansOnTheBarAndTalks(t *testing.T) {
	peRetry(t, "philosopher", false, func(g *Game) string {
		st := g.partyRunning("philosopher").D.(*philoState)
		gu := st.m.gu
		said := map[string]bool{}
		for s := 0; s < 30*58 && g.partyRunning("philosopher") != nil; s++ {
			peStep(g, 1.0/30, true)
			if s%10 == 0 {
				peDraw(g)
			}
			for _, l := range philoLines {
				if gu.icon == l {
					said[l] = true
				}
			}
		}
		if !st.arrived {
			return "never reached the bar"
		}
		near := false
		for _, p := range g.lvl.POIs["drink"] {
			if dist(p, st.spot) < 1 && dist(gu.pos, st.spot) < 6 {
				near = true
			}
		}
		if !near && g.partyRunning("philosopher") == nil && dist(gu.pos, st.spot) > 40 {
			return "he is not at the bar"
		}
		if len(said) < 2 {
			return "fewer than two sayings"
		}
		if g.partyRunning("philosopher") != nil {
			return "did not end"
		}
		if gu.gait == "stagger" && st.gait != "stagger" {
			return "gait not restored"
		}
		return ""
	})
}

// ---- gossiping granny -----------------------------------------------------------------------------------------

func TestPevGossipGrannyWatchesAndLeaves(t *testing.T) {
	peRetry(t, "granny", false, func(g *Game) string {
		st := g.partyRunning("granny").D.(*grannyState)
		gu := st.m.gu
		if sp := gu.cs.Spec; sp == nil || !sp.Female || sp.Hair2 != HairBun || sp.Glasses != GlassRound {
			return "not dressed as a granny"
		}
		g.p.dir = DirDown
		said := map[string]bool{}
		for s := 0; s < 30*50 && st.phase != grWatch; s++ {
			peStep(g, 1.0/30, true)
		}
		if st.phase != grWatch {
			return "never got to watch"
		}
		if d := dist(gu.pos, g.p.pos); d > grannyDist+40 || d < 14 {
			return "not at about two tiles"
		}
		for s := 0; s < 30*9; s++ { // she whispers
			peStep(g, 1.0/30, true)
			for _, l := range grannyLines {
				if gu.icon == l {
					said[l] = true
				}
			}
		}
		if len(said) < 1 {
			return "she did not whisper"
		}
		// the player stares at her
		scolded := false
		for s := 0; s < 30*8 && st.phase != grAway; s++ {
			g.p.dir = faceTo(g.p.pos, gu.pos)
			peStep(g, 1.0/30, true)
			for _, l := range grannyScold {
				if gu.icon == l {
					scolded = true
				}
			}
		}
		if st.phase != grAway || !scolded {
			return "she did not leave when stared at"
		}
		return ""
	})
}

// If she dies, the newspaper gets its fact (once); any other guest dying does not count.
func TestPevGossipGrannyDeathIsAFact(t *testing.T) {
	g := crowdStart(t, "granny", false)
	prev := g.progress.Stats
	t.Cleanup(func() { g.progress.Stats = prev })
	g.progress.Stats = map[string]int{}
	var other *Guest
	st := g.partyRunning("granny").D.(*grannyState)
	for _, gu := range g.guests {
		if gu != st.m.gu && !gu.out() && gu.perf == "" {
			other = gu
			break
		}
	}
	g.killGuest(other, false)
	if g.progress.Stats["abuela_dead"] != 0 {
		t.Fatal("a plain guest counts as the granny")
	}
	g.killGuest(st.m.gu, false)
	if g.progress.Stats["abuela_dead"] != 1 {
		t.Fatalf("abuela_dead = %d, want 1", g.progress.Stats["abuela_dead"])
	}
	g.killGuest(st.m.gu, false) // already dead: nothing more
	for s := 0; s < 5; s++ {
		peStep(g, 1.0/30, true)
	}
	if g.partyRunning("granny") != nil {
		t.Error("the event goes on with a dead granny")
	}
	if g.progress.Stats["abuela_dead"] != 1 {
		t.Error("counted twice")
	}
}

// ---- karaoke ------------------------------------------------------------------------------------------------

func TestPevKaraokeSingsAndShowsTheScore(t *testing.T) {
	peRetry(t, "karaoke", false, func(g *Game) string {
		st := g.partyRunning("karaoke").D.(*karaokeState)
		gu := st.singer.gu
		if len(st.crowd.mem) < 2 {
			return "no audience"
		}
		if st.score < 0 || st.score > 99 || st.good != (st.score >= 78) {
			return "bad score"
		}
		sang, sawResult := false, false
		for s := 0; s < 30*45 && g.partyRunning("karaoke") != nil; s++ {
			peStep(g, 1.0/30, true)
			if g.partyRunning("karaoke") != nil && gu.gait != "hips" && st.phase != kGather {
				return "the singer lost his style"
			}
			if st.phase == kSing {
				for _, l := range karaokeLines {
					if gu.icon == l {
						sang = true
					}
				}
			}
			if st.phase == kResult {
				sawResult = true
			}
			if s%5 == 0 {
				peDraw(g)
			}
		}
		if !sang || !sawResult {
			return "no song or no result"
		}
		if g.partyRunning("karaoke") != nil {
			return "did not end"
		}
		if gu.gait == "hips" {
			return "gait not restored"
		}
		return ""
	})
}

// The score sign fits its box in both languages, on whole pixels.
func TestPevKaraokeSignFits(t *testing.T) {
	g := partyGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	for _, lg := range []string{"es", "en"} {
		uiLang = lg
		for _, s := range []string{"PUNTOS", "¡BIS!", "¡AUCH!"} {
			if w := g.textW(s, 1); w > 64-8 || w != math.Round(w) {
				t.Errorf("%s: %q is %.1f px wide for a 64 px sign", lg, s, w)
			}
		}
	}
}

// ---- tables -------------------------------------------------------------------------------------------------

func tablesTestLevels(t *testing.T) []int {
	t.Helper()
	partyGame(t) // levelDefs is complete once a Game exists
	var out []int
	for i := range campaignLevels {
		if tablesIcons[campaignLevels[i].icon] > 0 {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		t.Fatal("no level has tables")
	}
	return out
}

func TestTableSetsAreSafeAndReachable(t *testing.T) {
	for _, i := range tablesTestLevels(t) {
		d := levelDefs[i]
		l := buildLevelRaw(d)
		r0 := l.reachFrom(d.Start)
		nOld := len(l.Props)
		n := l.addTables()
		if n < 1 {
			t.Errorf("level %d (%s): no table placed", i, d.Icon)
			continue
		}
		if n > tablesIcons[d.Icon] {
			t.Errorf("level %d: %d sets, %d wanted", i, n, tablesIcons[d.Icon])
		}
		// no new prop stands on anything
		for a := nOld; a < len(l.Props); a++ {
			pa := l.Props[a]
			for b, pb := range l.Props {
				if a == b || (b >= nOld && b < a) {
					continue
				}
				if pairClash(pa.Name, pa.K, pa.Foot, pa.base(), pb.Name, pb.K, pb.Foot, pb.base()) {
					t.Errorf("level %d: %s at %.0f,%.0f clashes with %s at %.0f,%.0f", i, pa.Name, pa.X, pa.Y, pb.Name, pb.X, pb.Y)
				}
			}
			for _, w := range l.Walls {
				if w.Style != WallVoid && wallClash(pa.Name, pa.K, pa.Foot, pa.base(), w) {
					t.Errorf("level %d: %s on a wall", i, pa.Name)
				}
			}
			if l.badGround(Vec{pa.X, pa.Y}) || l.InPool(Vec{pa.X, pa.Y}) {
				t.Errorf("level %d: %s on bad ground", i, pa.Name)
			}
			for _, dr := range d.Dance {
				if dr.Has(pa.X, pa.Y) && pa.Name == "clothtable" {
					t.Errorf("level %d: a table on the dance floor", i)
				}
			}
		}
		// what could be walked before can be walked now
		r1 := l.reachFrom(d.Start)
		for c := range r0 {
			if r0[c] && !l.blocked[c] && !r1[c] {
				t.Errorf("level %d: the cell %d,%d was reachable and is cut off now", i, c%l.cw, c/l.cw)
				break
			}
		}
		// both places in front of every table are open floor, reachable, and the guests get there
		for _, p := range l.Props {
			if p.Name != "clothtable" {
				continue
			}
			slots := tableSlots(l, Vec{p.X, p.Y})
			for _, s := range slots {
				if s == (Vec{}) || !l.Free(s) || !r1[int(s.Y/navCell)*l.cw+int(s.X/navCell)] {
					t.Errorf("level %d: a place under the table at %.0f,%.0f is not usable", i, p.X, p.Y)
				}
			}
		}
		// the guests' own start and the walking area stay reachable
		for _, rc := range d.Roam {
			cx, cy := int((rc.X+rc.W/2)/navCell), int((rc.Y+rc.H/2)/navCell)
			if l.cellFree(cx, cy) && r0[cy*l.cw+cx] && !r1[cy*l.cw+cx] {
				t.Errorf("level %d: the middle of a walking area is cut off", i)
			}
		}
	}
}

func TestTableSetsAreDeterministicAndOnlyInSomeLevels(t *testing.T) {
	tablesTestLevels(t)
	for i, d := range levelDefs {
		if i >= len(campaignLevels) {
			break
		}
		want := tablesIcons[campaignLevels[i].icon]
		a := buildLevelRaw(d)
		b := buildLevelRaw(d)
		na, nb := a.addTables(), b.addTables()
		if want == 0 && (na != 0 || countKind(a, "clothtable") != 0) {
			t.Errorf("level %d (%s) must have no tables", i, campaignLevels[i].icon)
		}
		if na != nb {
			t.Errorf("level %d: %d sets, then %d", i, na, nb)
		}
		pos := func(l *Level) []Vec {
			var out []Vec
			for _, p := range l.Props {
				if p.Name == "clothtable" {
					out = append(out, Vec{p.X, p.Y})
				}
			}
			return out
		}
		pa, pb := pos(a), pos(b)
		if len(pa) != len(pb) {
			t.Errorf("level %d: table count differs", i)
			continue
		}
		for k := range pa {
			if pa[k] != pb[k] {
				t.Errorf("level %d: the tables are not deterministic", i)
			}
		}
	}
	// off switch
	tablesOff = true
	t.Cleanup(func() { tablesOff = false })
	if n := buildLevelRaw(levelDefs[0]).addTables(); n != 0 {
		t.Error("tablesOff is ignored")
	}
}

func TestTableSpritesAreSharpAndInsideTheirBox(t *testing.T) {
	const W, H = 260, 220
	ax, ay := 130, 160
	for _, px := range []int{1, 2} {
		k := func(v float64) int { return int(math.Round(v * float64(px))) }
		draws := map[string]func(plot plotFn){
			"finger":  func(p plotFn) { drawFingerPix(p, ax, ay, px, 1, k) },
			"fingerL": func(p plotFn) { drawFingerPix(p, ax, ay, px, -1, k) },
			"glasses": func(p plotFn) { drawRoundGlassesPix(p, ax, ay-k(28), px) },
			"lens":    func(p plotFn) { drawLensPix(p, ax+k(9), ay-k(22), px) },
			"bow":     func(p plotFn) { drawBowTiePix(p, ax, ay-k(26), px) },
			"vest":    func(p plotFn) { drawVestPix(p, ax, ay, px, k) },
			"tray": func(p plotFn) {
				drawTrayPix(p, ax+k(7), ay-k(15), px, []int{trayGlass, trayWine, trayCanape, trayOlive, trayGlass})
			},
			"mic":     func(p plotFn) { drawMicPix(p, ax, ay, px, k) },
			"ears":    func(p plotFn) { drawEarsPix(p, ax, ay, px, k) },
			"shards":  func(p plotFn) { drawShardsPix(p, ax, ay, px, 6, 3) },
			"badnote": func(p plotFn) { drawBadNotesPix(p, ax, ay-k(46), px, 0.7) },
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
			if painted == 0 {
				t.Errorf("%s paints nothing", name)
			}
		}
	}
	// the cloth table sprite: 40x30 with its foot box inside the opaque part
	k := propKinds["clothtable"]
	if k == nil || k.W != 40 || k.H != 30 {
		t.Fatal("clothtable is missing")
	}
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	k.Draw(b, 0)
	opaque := 0
	for i := 3; i < len(b.p); i += 4 {
		if b.p[i] > 0 {
			opaque++
		}
	}
	if opaque < 600 {
		t.Errorf("the table is almost empty (%d px)", opaque)
	}
}

// ---- hiding under the tables ---------------------------------------------------------------------------------

// peWithTables: peLoad (pe_test.go) leaves the cloth tables out unless a test asks for them.
var peWithTables bool

func hideLoad(t *testing.T, two bool) *Game {
	t.Helper()
	prevWith := peWithTables
	peWithTables = true
	t.Cleanup(func() { peWithTables = prevWith })
	probe := partyGame(t)
	for i := range campaignLevels {
		if tablesIcons[campaignLevels[i].icon] == 0 {
			continue
		}
		g := peLoad(t, i, two)
		if countKind(g.lvl, "clothtable") > 0 {
			hideEpisode, hideRolled, hideNextTry, hideCoolLvl = false, false, 0, nil
			t.Cleanup(func() { hideEpisode, hideRolled, hideNextTry, hideCoolLvl = false, false, 0, nil; dbgPartyEvent = "" })
			return g
		}
	}
	_ = probe
	t.Fatal("no loaded level with tables")
	return nil
}

func TestTableHideGuestsRunUnderAndComeOut(t *testing.T) {
	for _, two := range []bool{false, true} {
		var g *Game
		var st *hideState
		for try := 0; try < 6 && (st == nil || !hideAnyIn(st)); try++ { // some parties have nobody close to the table
			g = hideLoad(t, two)
			g.rng = rand.New(rand.NewSource(int64(5 + try)))
			g.time = 4
			peForce(g, "tablehide")
			st = nil
			for s := 0; s < 30*28; s++ {
				peStep(g, 1.0/30, true)
				if a := g.partyRunning("tablehide"); a != nil {
					st = a.D.(*hideState)
					if hideAnyIn(st) {
						break
					}
				}
			}
		}
		if st == nil || len(st.h) == 0 || len(st.h) > hideMax || !hideAnyIn(st) {
			t.Fatalf("2P=%v: nobody hid", two)
		}
		scr := newTestImage()
		sawIn := 0
		for _, h := range st.h {
			if !h.in {
				continue
			}
			sawIn++
			f := pevFxs[h.m.gu]
			if f == nil || f.hid != h.tbl {
				t.Fatalf("a hidden guest has no mark")
			}
			pos := h.m.gu.pos
			if !g.pevGuestDraw(scr, h.m.gu, &pos) {
				t.Error("a hidden guest is still drawn whole")
			}
			if dist(h.m.gu.pos, h.slot) > 3 {
				t.Error("a hidden guest is not at his place")
			}
		}
		if sawIn == 0 {
			t.Fatal("none under a table")
		}
		for s := 0; s < 30*4; s++ { // a few seconds drawing them hidden
			peStep(g, 1.0/30, true)
			peDraw(g)
		}
		// the danger is over: they come out and nobody stays held
		dbgPartyEvent = ""
		touched := map[*Guest]bool{}
		for _, h := range st.h {
			touched[h.m.gu] = true
		}
		for _, gu := range g.guests { // whatever frightened the others is over, too
			if (gu.state == gScared || gu.state == gFlee) && !touched[gu] {
				gu.state, gu.timer, gu.path = gIdle, 1, nil
			}
		}
		for s := 0; s < 30*26 && g.partyRunning("tablehide") != nil; s++ { // 3.5 s of calm, after any fright of the party itself
			peStep(g, 1.0/30, true)
		}
		if a := g.partyRunning("tablehide"); a != nil {
			sc := 0
			for _, gu := range g.guests {
				if gu.state == gScared || gu.state == gFlee {
					sc++
				}
			}
			if hideDanger(g) { // the party itself frightened people again: staying in is right
				t.Logf("2P=%v: new danger in the party, they stay in (%d scared)", two, sc)
				continue
			}
			t.Fatalf("2P=%v: they never came out (danger %v, %d scared, calm %.1f, %d hiders, time %.1f)", two, hideDanger(g), sc, a.D.(*hideState).calmFor, len(a.D.(*hideState).h), g.time)
		}
		if gu := peStale(g, touched); gu != nil {
			t.Error("a guest is still held")
		}
		if len(pevFxs) != 0 {
			t.Errorf("%d marks left", len(pevFxs))
		}
	}
}

func hideAnyIn(st *hideState) bool {
	for _, h := range st.h {
		if h.in {
			return true
		}
	}
	return false
}

func TestTableHideRealDangerStartsSometimes(t *testing.T) {
	g := hideLoad(t, false)
	e := partyByID("tablehide")
	tr := partyTraitsOf(g.lvl.Def, g.levelIdx)
	g.pev = pevState{ready: true, traits: tr}
	started, episodes := 0, 0
	for ep := 0; ep < 16; ep++ {
		// three scared guests = danger
		n := 0
		for _, gu := range g.guests {
			if n < 3 && lifeFree(gu) && gu.perf == "" {
				gu.state, gu.timer = gScared, 30
				n++
			}
		}
		if !hideDanger(g) || !e.Eligible(g, tr) && !hideEpisode {
			t.Fatal("three scared guests are a danger")
		}
		episodes++
		for s := 0; s < 30*6; s++ {
			peStep(g, 1.0/30, true)
			if g.partyRunning("tablehide") != nil {
				started++
				break
			}
		}
		// the danger ends
		for _, gu := range g.guests {
			if gu.state == gScared {
				gu.state, gu.timer, gu.path = gIdle, 1, nil
			}
		}
		for s := 0; s < 30*10 && g.partyRunning("tablehide") != nil; s++ {
			peStep(g, 1.0/30, true)
		}
		peStep(g, 1.0/30, true)
	}
	t.Logf("started in %d of %d episodes", started, episodes)
	if started == 0 || started == episodes {
		t.Errorf("it must start sometimes, not always: %d of %d", started, episodes)
	}
}

func TestTableHideOnlyWhereThereAreTables(t *testing.T) {
	g := partyGame(t)
	e := partyByID("tablehide")
	for i := range campaignLevels {
		tr := partyTraitsOf(levelDefs[i], i)
		dbgPartyEvent = "tablehide"
		g.time = 5
		ok := e.Eligible(g, tr)
		dbgPartyEvent = ""
		if ok != (tablesIcons[campaignLevels[i].icon] > 0 || seatHideIcon(campaignLevels[i].icon)) { // + the long tables (seats.go)
			t.Errorf("level %d (%s): eligible=%v", i, campaignLevels[i].icon, ok)
		}
	}
	hideEpisode, hideRolled = false, false
}

func TestTableHideKilledHiderIsFreed(t *testing.T) {
	g := hideLoad(t, false)
	g.time = 4
	peForce(g, "tablehide")
	var st *hideState
	for s := 0; s < 30*30; s++ {
		peStep(g, 1.0/30, true)
		if a := g.partyRunning("tablehide"); a != nil {
			st = a.D.(*hideState)
			if len(st.h) > 0 && st.h[0].in {
				break
			}
		}
	}
	if st == nil || len(st.h) == 0 || !st.h[0].in {
		t.Skip("nobody under a table in this party")
	}
	gu := st.h[0].m.gu
	g.killGuest(gu, false)
	peStep(g, 1.0/30, true)
	peStep(g, 1.0/30, true)
	if f := pevFxs[gu]; f != nil && f.hid != nil {
		t.Error("the dead guest keeps hiding")
	}
}

func newTestImage() *ebiten.Image { return ebiten.NewImage(screenW, screenH) }
