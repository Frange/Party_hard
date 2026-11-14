package main

import (
	"math"
	"math/rand"
	"testing"
)

// The torch guard (pe_flashlight.go, idea 291) and the mirror balls (discoball.go, idea 297).

func flashGame(t *testing.T, two bool) (*Game, *flashGuard) {
	t.Helper()
	t.Cleanup(func() { flashGuards = map[*Guard]*flashGuard{} })
	// the tests need walls near the patrol (sight through a wall): the walled night parties first (until v1.4.0 the
	// ranch was the first one; it became BARCELONA, a day square without the torch guard)
	old := flashIcons
	t.Cleanup(func() { flashIcons = old })
	flashIcons = map[string]bool{"skate": true, "grave": true}
	g := peStart(t, "linterna", two)
	s, _ := g.partyRunning("linterna").D.(*flashGuard)
	if s == nil || s.gd == nil || !s.on {
		t.Fatal("no torch guard")
	}
	// nobody else in the way (a kill of somebody else, a guest in front of him)
	for _, gu := range g.guests {
		if !gu.out() && dist(gu.pos, s.gd.pos) < 200 {
			gu.pos = g.lvl.NearestFree(Vec{math.Mod(s.gd.pos.X+420, g.lvl.W-60) + 30, math.Mod(s.gd.pos.Y+260, g.lvl.H-60) + 30})
			gu.path = nil
		}
	}
	return g, s
}

// flashAim: puts the guard at a, looking at b (beam and body), standing still.
func flashAim(g *Game, s *flashGuard, a, b Vec) {
	gd := s.gd
	gd.pos, gd.state, gd.timer, gd.path, gd.icon, gd.sight = a, gdWait, 1e6, nil, "", 0
	gd.dir = faceTo(a, b)
	s.ang = math.Atan2(b.Y-a.Y, b.X-a.X)
	s.sight = 0
}

// flashGuardStep: one frame of the guards as updatePlay runs them (each one facing the player nearest to him).
func flashGuardStep(g *Game, dt float64) {
	g.time += dt
	g.tick++
	for _, gd := range g.guards {
		restore := g.focus(gd.pos)
		g.updateGuard(gd, dt)
		restore()
	}
}

// Only in the night venues of flashIcons, never in a day level, and about 1 party in 3 of those (with the memory of the last
// parties a little less); a guard of g.guards that counts for the party (total + 1) and only with room.
func TestFlashlightOnlyAtNightAndRare(t *testing.T) {
	g := partyGame(t)
	e := partyByID("linterna")
	if e == nil || e.Minor || e.Big || e.DailyOK || e.Prob < 0.25 || e.Prob > 0.40 || e.MaxRuns > 1 || e.Cooldown > 0 {
		t.Fatalf("bad registration %+v", e)
	}
	in := peEligible(g, "linterna")
	if len(in) == 0 || len(in) > len(campaignLevels)/2 {
		t.Fatalf("it fits %d campaign levels: %v", len(in), in)
	}
	for _, i := range in {
		if levelDefs[i].Day || !flashIcons[levelDefs[i].Icon] {
			t.Errorf("level %d (%s) is a day level or not a night venue", i, levelDefs[i].Icon)
		}
	}
	for i := range campaignLevels {
		if levelDefs[i].Day && e.Eligible(g, partyTraitsOf(levelDefs[i], i)) {
			t.Errorf("day level %d eligible", i)
		}
	}
	r := rand.New(rand.NewSource(91))
	var hist [][]string
	fit, got := 0, 0
	fits := map[int]bool{}
	for _, i := range in {
		fits[i] = true
	}
	for k := 0; k < 6000; k++ { // parties on random campaign levels, as a player meets them
		idx := r.Intn(len(campaignLevels))
		tr := partyTraitsOf(levelDefs[idx], idx)
		var ids []string
		if fits[idx] {
			fit++
		}
		for _, p := range partyPlan(r, g, tr, hist, false) {
			ids = append(ids, p.ev.ID)
			if p.ev.ID == "linterna" {
				if !fits[idx] {
					t.Fatalf("planned in level %d", idx)
				}
				got++
			}
		}
		hist = append(hist, ids)
		if len(hist) > partyMemory {
			hist = hist[1:]
		}
	}
	f := float64(got) / float64(fit)
	t.Logf("torch guard planned in %.0f %% of the night parties where it fits", f*100)
	if f < 0.18 || f > 0.40 {
		t.Errorf("torch guard in %.0f %% of its parties, want about 1 in 3 (or a little less)", f*100)
	}

	g2, s := flashGame(t, false)
	if g2.lvl.Def.Day || s.lvl != g2.lvl {
		t.Fatal("started in a day level")
	}
	n := 0
	for _, gd := range g2.guards {
		if g2.flashOf(gd) != nil {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d torch guards", n)
	}
	// a second one never comes in the same party
	if flashStart(g2, &partyAct{ev: e}) {
		t.Error("a second torch guard started")
	}
}

func TestFlashlightNeedsRoom(t *testing.T) {
	g, s := flashGame(t, false)
	before, tot := len(g.guards), g.total
	if tot <= 0 || s.gd == nil {
		t.Fatal("bad game")
	}
	delete(flashGuards, s.gd)
	g.total = g.goalMax() + 50 // no room left for anybody
	if g.guestRoom() > 0 {
		t.Skip("cannot make the party full here")
	}
	if flashStart(g, &partyAct{ev: partyByID("linterna")}) {
		t.Error("a torch guard came into a full party")
	}
	if len(g.guards) != before {
		t.Error("a guard was added")
	}
}

// flashPairs finds, on the level, a spot pair (guard, player) with clear sight beyond the ordinary eyes of a guard (but in
// reach of the torch) and one 50-145 px apart with a wall between.
func flashPairs(g *Game) (clearA, clearB, wallA, wallB Vec, okC, okW bool) {
	l := g.lvl
	r := rand.New(rand.NewSource(5))
	for i := 0; i < 20000 && !(okC && okW); i++ {
		a := l.RandomRoam(r)
		ang := float64(r.Intn(4))*math.Pi/2 + (r.Float64()-0.5)*0.3 // he faces one of the four ways
		d := 50 + r.Float64()*95
		b := Vec{a.X + math.Cos(ang)*d, a.Y + math.Sin(ang)*d}
		if !l.Free(a) || !l.Free(b) || l.SolidP(b.X, b.Y) || b.X < 10 || b.Y < 10 || b.X > l.W-10 || b.Y > l.H-10 {
			continue
		}
		if l.LOS(a, b) {
			if !okC && d > guardRange+8 { // beyond the ordinary eyes of a guard: only the torch reaches
				clearA, clearB, okC = a, b, true
			}
		} else if !okW {
			wallA, wallB, okW = a, b, true
		}
	}
	return
}

// The light only sees what the guard really sees: in the cone and with clear sight; through a wall, behind him or out of
// reach, nothing.
func TestFlashlightSightNeedsLineOfSight(t *testing.T) {
	g, s := flashGame(t, false)
	ca, cb, wa, wb, okC, okW := flashPairs(g)
	if !okC || !okW {
		t.Fatalf("no test spots (clear %v wall %v)", okC, okW)
	}
	flashAim(g, s, ca, cb)
	if !g.flashLit(s, cb) {
		t.Error("a player in the cone with clear sight is not lit")
	}
	flashAim(g, s, cb, ca) // turned the other way: the player is behind him
	s.ang += math.Pi
	if g.flashLit(s, ca) {
		t.Error("a player behind him is lit")
	}
	flashAim(g, s, wa, wb)
	if g.flashLit(s, wb) {
		t.Error("the light went through a wall")
	}
	far := Vec{ca.X + (cb.X-ca.X)/dist(ca, cb)*(flashRange+40), ca.Y + (cb.Y-ca.Y)/dist(ca, cb)*(flashRange+40)}
	flashAim(g, s, ca, cb)
	if g.flashLit(s, far) {
		t.Error("lit beyond the reach of the torch")
	}
	// the cone drawn on the floor is cut by the walls too
	flashAim(g, s, wa, wb)
	s.raysAt = -99
	g.flashCast(s, wa, s.ang, flashRange)
	mid := s.rays[flashRays/2]
	if mid > dist(wa, wb)+6 {
		t.Errorf("the beam towards a wall is %.0f px long, the wall is at less than %.0f", mid, dist(wa, wb))
	}
}

// In the light the guard asks who is there and then gives the alarm, and the ordinary guard chase follows. Behind a wall
// nothing happens.
func TestFlashlightAlarmAndChase(t *testing.T) {
	for _, two := range []bool{false, true} {
		g, s := flashGame(t, two)
		ca, cb, wa, wb, okC, okW := flashPairs(g)
		if !okC || !okW {
			t.Fatal("no test spots")
		}
		gd := s.gd
		pl := g.p
		if two {
			pl = g.p2
			g.p.pos = g.lvl.NearestFree(Vec{math.Mod(ca.X+480, g.lvl.W-40) + 20, math.Mod(ca.Y+270, g.lvl.H-40) + 20})
		}
		// behind a wall: no alarm
		flashAim(g, s, wa, wb)
		pl.pos, pl.hidden = wb, false
		for i := 0; i < 60; i++ {
			flashGuardStep(g, 1.0/30)
			gd.pos, gd.state, gd.timer = wa, gdWait, 1e6
		}
		if s.alarms != 0 || gd.state == gdReact || gd.state == gdChase {
			t.Fatalf("2P=%v: the guard saw the player through a wall", two)
		}
		// in the light
		flashAim(g, s, ca, cb)
		pl.pos = cb
		heard := false
		for i := 0; i < 60 && gd.state != gdReact && gd.state != gdChase; i++ {
			flashGuardStep(g, 1.0/30)
			for _, l := range flashWhoLines {
				heard = heard || gd.icon == l
			}
			if gd.state == gdWait {
				gd.pos = ca
			}
		}
		if s.alarms != 1 || gd.state != gdReact && gd.state != gdChase {
			t.Fatalf("2P=%v: no alarm in the light (alarms %d state %d)", two, s.alarms, gd.state)
		}
		alarm := false
		for _, l := range flashAlarmLines {
			alarm = alarm || gd.icon == l
		}
		if !alarm || !heard {
			t.Errorf("2P=%v: the shouts: who %v alarm %q", two, heard, gd.icon)
		}
		for i := 0; i < 30 && gd.state == gdReact; i++ {
			flashGuardStep(g, 1.0/30)
		}
		if gd.state != gdChase && gd.state != gdBeat {
			t.Errorf("2P=%v: no chase after the alarm (state %d)", two, gd.state)
		}
		if gd.special || gd.leash > 170 {
			t.Errorf("2P=%v: tougher than an ordinary guard", two)
		}
	}
}

// The trick: the use key right behind him switches the torch off (from the front it does not); then he is an ordinary
// guard, he does not turn round at once, and the light no longer sees anybody.
func TestFlashlightSwitchOffFromBehind(t *testing.T) {
	g, s := flashGame(t, false)
	ca, cb, _, _, okC, _ := flashPairs(g)
	if !okC {
		t.Fatal("no test spots")
	}
	gd := s.gd
	flashAim(g, s, ca, cb)
	v := Vec{math.Cos(s.ang), math.Sin(s.ang)}
	g.p.pos = Vec{ca.X + v.X*16, ca.Y + v.Y*16} // in front of him
	if g.flashInteract(g.p) || !s.on {
		t.Fatal("switched off from the front")
	}
	g.p.pos = Vec{ca.X - v.X*14, ca.Y - v.Y*14} // right behind him
	if !g.flashBehind(s, g.p) {
		t.Fatal("standing right behind him does not count")
	}
	if !g.flashInteract(g.p) || s.on || s.offWhy != "back" {
		t.Fatalf("not switched off from behind (on %v why %q)", s.on, s.offWhy)
	}
	off := false
	for _, l := range flashOffLines {
		off = off || gd.icon == l
	}
	if !off {
		t.Errorf("no complaint: %q", gd.icon)
	}
	for i := 0; i < 30; i++ { // one second: he fumbles with it, he does not catch the player
		flashGuardStep(g, 1.0/30)
		if gd.state == gdReact || gd.state == gdChase || gd.state == gdBeat {
			t.Fatal("he turned and caught the player at once")
		}
	}
	g.p.pos = cb
	if g.flashLit(s, cb) {
		t.Error("the dead torch still lights")
	}
	// a smoke daze makes him drop it too
	g2, s2 := flashGame(t, false)
	s2.gd.smogT = 2
	flashGuardStep(g2, 1.0/30)
	if s2.on || s2.offWhy != "daze" {
		t.Errorf("a dazed guard kept his torch (%v %q)", s2.on, s2.offWhy)
	}
}

// Killed: an ordinary body and the torch rolls on the floor, lit for a while.
func TestFlashlightKilledDropsTheTorch(t *testing.T) {
	g, s := flashGame(t, false)
	nb, tot := len(g.bodies), g.total
	g.killGuard(s.gd)
	flashGuardStep(g, 1.0/30)
	if len(g.bodies) != nb+1 || g.total != tot {
		t.Errorf("bodies %d -> %d, total %d -> %d", nb, len(g.bodies), tot, g.total)
	}
	if s.on || s.offWhy != "dead" || s.dropT <= 0 {
		t.Fatalf("the torch did not fall: on %v why %q drop %.1f", s.on, s.offWhy, s.dropT)
	}
	peDraw(g) // the torch on the floor and the rest of the overlays draw without trouble
}

// He walks his patrol (several points) and never stands stuck.
func TestFlashlightPatrolNeverStuck(t *testing.T) {
	g, s := flashGame(t, false)
	gd := s.gd
	if len(gd.route) < 2 {
		t.Fatalf("route of %d points", len(gd.route))
	}
	for i := 1; i < len(gd.route); i++ {
		if g.lvl.Path(gd.route[i-1], gd.route[i]) == nil {
			t.Errorf("route point %d unreachable", i)
		}
	}
	g.p.hidden = true // nobody to chase: only the patrol
	t.Cleanup(func() { g.p.hidden = false })
	visits := map[int]bool{}
	still, worst := 0.0, 0.0
	last := gd.pos
	for i := 0; i < 30*60; i++ {
		flashGuardStep(g, 1.0/30)
		if gd.state == gdPatrol {
			visits[gd.ri] = true
			if dist(gd.pos, last) < 0.3 {
				still += 1.0 / 30
			} else {
				still = 0
			}
			worst = math.Max(worst, still)
		} else {
			still = 0
		}
		last = gd.pos
	}
	if len(visits) < 2 {
		t.Errorf("he did not walk his patrol (route indexes %v)", visits)
	}
	if worst > flashStuck+0.5 {
		t.Errorf("stuck %.1f s on patrol", worst)
	}
}

// The mirror ball: in every dance party under a roof (and the party boat), one per level, over the dance floor, never solid.
func TestMirrorBallInTheDanceVenues(t *testing.T) {
	partyGame(t) // the levels of the game loaded
	want := map[string]bool{"disco70": true}
	for k := range mirrorBallIcons {
		want[k] = true
	}
	k := propKinds["mirrorball"]
	if k == nil || !k.NoSolid || !overlayKinds["mirrorball"] {
		t.Fatal("the mirror ball is not a hanging overlay")
	}
	for i := range campaignLevels {
		d := levelDefs[i]
		n := 0
		var at PropInst
		for _, p := range d.Props {
			if p.Kind == "mirrorball" {
				n++
				at = p
			}
		}
		t.Logf("%2d %-22s %-8s mirror balls %d", i, d.Title, d.Icon, n)
		if want[d.Icon] != (n == 1) || n > 1 {
			t.Errorf("level %d (%s): %d mirror balls", i, d.Icon, n)
			continue
		}
		if n == 1 {
			in := false
			for _, r := range d.Dance {
				if r.Grow(12).Has(at.X, at.Y) {
					in = true
				}
			}
			if !in && d.Icon != "disco70" {
				t.Errorf("level %d: the ball is not over a dance floor", i)
			}
		}
	}
	// the random-mode rolls of the same venues get one too
	for i := len(campaignLevels); i < len(levelDefs); i++ {
		n := 0
		for _, p := range levelDefs[i].Props {
			if p.Kind == "mirrorball" {
				n++
			}
		}
		if n > 1 {
			t.Errorf("random level %d: %d mirror balls", i, n)
		}
	}
}
