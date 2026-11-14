package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the metal detector arches (metaldet.go) with the real Game after loadLevel.

// mdGlobals: every global of the detector is put back after the test.
func mdGlobals(t *testing.T) {
	t.Helper()
	pf, pi, po, pr, ps, prun, pnone, pb := mdForce, mdInTests, mdOff, mdRng, mdSeed, md, mdNoneAt, mdBeepP
	t.Cleanup(func() {
		mdForce, mdInTests, mdOff, mdRng, mdSeed, md, mdNoneAt, mdBeepP = pf, pi, po, pr, ps, prun, pnone, pb
	})
	mdBeepP = 1 // every crossing out of the cooldown beeps, unless the test wants the dice
}

// mdLoad: a party of level idx with the arches forced (the run is nil if no door fits).
func mdLoad(t *testing.T, idx int, two bool, seed int64) (*Game, *mdRun) {
	t.Helper()
	mdForce = true
	var g *Game
	for k := int64(0); k < 6; k++ { // a small object placed at random may stand in every doorway: try another party
		mdSeed, mdRng, md, mdNoneAt = seed+k*1000, nil, nil, nil
		g = peLoad(t, idx, two)
		g.policeCalled, g.cops = false, nil
		if run := g.mdActive(); run != nil {
			return g, run
		}
	}
	return g, nil
}

// mdVenueIdx: the campaign levels of the detector.
func mdVenueIdx(t *testing.T) []int {
	t.Helper()
	var out []int
	for _, ic := range []string{"casino", "hockey"} { // v1.4.0: the club and the cinema are retired
		i := campaignIdx(ic)
		if i < 0 {
			t.Fatalf("no campaign party %q", ic)
		}
		out = append(out, i)
	}
	return out
}

// mdCross: the player walks through the arches (one frame on each side).
func mdCross(g *Game, run *mdRun, p *Player, down bool) {
	c := run.door.Center()
	up, dn := Vec{c.X, run.door.Y - mdBand + 6}, Vec{c.X, run.door.Y + run.door.H + mdBand - 6}
	if !down {
		up, dn = dn, up
	}
	p.pos = up
	g.updateMetalDet(1.0 / 60)
	p.pos = dn
	g.updateMetalDet(1.0 / 60)
}

// The arches come only in the casino and the ice hockey rink; there they find a doorway (forced; at least once in 3 tries), add nothing
// solid (props, solids and the walking map are the same), sit in a gap away from the start with nothing drawn on them,
// and the way through the door stays open from the start on both sides.
func TestMetalDetVenuesGeometry(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	venues := map[int]bool{}
	for _, i := range mdVenueIdx(t) {
		venues[i] = true
	}
	found := map[int]int{}
	t.Cleanup(func() {
		for i := range venues {
			if found[i] == 0 {
				t.Errorf("%s: the arches never found a door in 3 tries", levelDefs[i].Title)
			}
		}
	})
	for k := 0; k < len(campaignLevels)*3; k++ {
		i := k % len(campaignLevels)
		if k >= len(campaignLevels) && !venues[i] {
			continue // the venues three times: the small objects of a level land elsewhere each time
		}
		g, run := mdLoad(t, i, false, int64(31+k))
		l := g.lvl
		if !venues[i] {
			if run != nil {
				t.Errorf("%s: metal detector outside its venues", l.Def.Title)
			}
			continue
		}
		if run == nil {
			// a small object of the level (placed at random each time) may stand in every doorway: then no arches
			t.Logf("%s (try %d): forced metal detector found no door", l.Def.Title, k)
			continue
		}
		found[i]++
		// nothing solid: the same props, solids and blocked cells as a level without the arches
		nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
		blocked := append([]bool(nil), l.blocked...)
		md, mdNoneAt = nil, nil
		if g.mdActive() == nil {
			t.Fatalf("%s: no arches on the second look", l.Def.Title)
		}
		if len(l.Props) != nProps || len(l.solidsP) != nP || len(l.solidsN) != nN {
			t.Errorf("%s: the arches added props or solids", l.Def.Title)
		}
		for k := range blocked {
			if blocked[k] != l.blocked[k] {
				t.Fatalf("%s: the walking map changed", l.Def.Title)
			}
		}
		dr := run.door
		if dr.W < mdMinW || dr.W > mdMaxW || run.n < 1 || float64(run.n)*mdArchW > dr.W+0.5 {
			t.Errorf("%s: door %v with %d arches", l.Def.Title, dr, run.n)
		}
		if dist(dr.Center(), l.Def.Start) < 140 {
			t.Errorf("%s: arches by the start", l.Def.Title)
		}
		arch := mdArchRect(dr)
		for _, o := range l.Props {
			if !o.gone && !overlayKinds[o.Name] && pbOverlap(o.base(), arch) {
				t.Errorf("%s: %s drawn over the arches", l.Def.Title, o.Name)
			}
		}
		// reachable from the start on both sides, and a path goes through the door
		reach := l.reachFrom(l.NearestFree(l.Def.Start))
		c := dr.Center()
		up, dn := Vec{c.X, dr.Y - mdBand + 6}, Vec{c.X, dr.Y + dr.H + mdBand - 6}
		if !reach[l.navIdx(up)] || !reach[l.navIdx(dn)] {
			t.Errorf("%s: a side of the arches is cut off", l.Def.Title)
		}
		if p := l.Path(up, dn); len(p) == 0 {
			t.Errorf("%s: no path through the arches", l.Def.Title)
		}
		for x := dr.X + 4; x < dr.X+dr.W-4; x += 8 {
			if l.SolidP(x, dr.Y+dr.H/2) {
				t.Errorf("%s: the gap is solid at x=%.0f", l.Def.Title, x)
				break
			}
		}
	}
}

// The dice: without forcing, some parties of the venues get the arches and some do not (never every one).
func TestMetalDetSometimes(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	idx := campaignIdx("casino")
	with := 0
	for s := int64(1); s <= 12; s++ {
		mdForce, mdInTests = false, true
		mdSeed, mdRng, md, mdNoneAt = s*101, nil, nil, nil
		g := peLoad(t, idx, false)
		if g.mdActive() != nil {
			with++
		}
	}
	if with == 0 || with == 12 {
		t.Errorf("arches in %d of 12 casino parties: want sometimes", with)
	}
}

// A player who crosses sets the alarm off: the arches blink red and ONE guard walks to the door to look (no chase,
// no handcuffs mark, no police); crossing again within the cooldown does nothing; if it goes on (another alarm soon
// after) the police are called to the place, without handcuffs; after that nobody else is sent (2026-11-12).
func TestMetalDetPlayerTripsAndCooldown(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	g, run := mdLoad(t, campaignIdx("casino"), false, 7)
	if run == nil {
		t.Fatal("no arches")
	}
	if len(g.guards) == 0 {
		t.Fatal("the casino has no guards")
	}
	g.time = 30
	mdCross(g, run, g.p, true)
	if run.trips != 1 || g.time >= run.alarmUntil {
		t.Fatalf("crossing: trips %d alarm until %.1f (now %.1f)", run.trips, run.alarmUntil, g.time)
	}
	gd := run.chaser
	if gd == nil || gd.state != gdInvestigate || len(gd.path) == 0 {
		t.Fatalf("no guard sent to look (chaser %v)", gd)
	}
	if gd.icon == "" {
		t.Error("the guard says nothing")
	}
	sent := 0
	for _, o := range g.guards {
		if o.state == gdInvestigate || o.state == gdReact || o.state == gdChase {
			sent++
		}
	}
	if sent != 1 {
		t.Errorf("%d guards answer the first alarm, want 1", sent)
	}
	if g.mdMarked(g.p) || g.isSuspectP(g.p) || g.policeCalled {
		t.Error("the first alarm put the handcuffs mark on the player or called the police")
	}
	// the arches blinking red draw without trouble
	dst := ebiten.NewImage(960, 540)
	g.mdDraw(dst, run)
	g.mdDrawMarks(dst)
	if g.p.dead {
		t.Error("the alarm killed the player")
	}
	// the nearest free guard
	for _, o := range g.guards {
		if o != gd && !o.dead && o.vip == 0 && len(o.route) > 0 && dist(o.pos, g.p.pos) < dist(gd.pos, g.p.pos)-0.01 && o.state == gdPatrol {
			t.Errorf("a nearer guard was not sent")
		}
	}
	// back and forth within the cooldown: silent
	g.time += 2
	mdCross(g, run, g.p, false)
	mdCross(g, run, g.p, true)
	if run.trips != 1 {
		t.Errorf("trips %d within the cooldown", run.trips)
	}
	// it goes on: the police are called to the place, without handcuffs
	g.time += mdCooldown
	mdCross(g, run, g.p, false)
	if run.trips != 2 || !g.policeCalled || run.step != 2 {
		t.Fatalf("second alarm: trips %d police %v step %d", run.trips, g.policeCalled, run.step)
	}
	if g.wanted > 0 || g.isSuspectP(g.p) || g.mdMarked(g.p) {
		t.Errorf("the police came with the handcuffs (wanted %.0f)", g.wanted)
	}
	// and then nobody else: no new guard, no second call
	guardsBusy := func() int {
		n := 0
		for _, o := range g.guards {
			if o.state == gdInvestigate || o.state == gdReact || o.state == gdChase {
				n++
			}
		}
		return n
	}
	before := guardsBusy()
	pend := g.pendingCar
	g.time += mdCooldown
	mdCross(g, run, g.p, true)
	if run.trips != 3 || guardsBusy() > before || g.pendingCar != pend {
		t.Errorf("a third alarm sent more people (busy %d -> %d, car %.1f -> %.1f)", before, guardsBusy(), pend, g.pendingCar)
	}
}

// Not every crossing beeps (its own dice), and a quiet crossing sends nobody.
func TestMetalDetNotEveryCrossingBeeps(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	g, run := mdLoad(t, campaignIdx("casino"), false, 7)
	if run == nil {
		t.Fatal("no arches")
	}
	mdBeepP, mdRng, mdSeed = 0.6, nil, 4242
	g.time = 30
	n := 30
	for i := 0; i < n; i++ {
		g.time += mdCooldown + 1
		mdCross(g, run, g.p, i%2 == 0)
	}
	if run.trips == 0 || run.trips == n {
		t.Errorf("%d of %d crossings beeped: want some, not all", run.trips, n)
	}
}

// Guests walk through without an alarm (the arches only blink green).
func TestMetalDetGuestsNoAlarm(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	g, run := mdLoad(t, campaignIdx("hockey"), false, 3)
	if run == nil {
		t.Fatal("no arches")
	}
	g.time = 30
	g.p.pos = g.lvl.Def.Start
	var gu *Guest
	for _, o := range g.guests {
		if !o.out() {
			gu = o
			break
		}
	}
	if gu == nil {
		t.Fatal("no guest")
	}
	c := run.door.Center()
	for y := run.door.Y - mdBand + 2; y <= run.door.Y+run.door.H+mdBand-2; y += 3 {
		gu.pos = Vec{c.X, y}
		g.updateMetalDet(1.0 / 60)
	}
	if run.trips != 0 || run.chaser != nil || g.time < run.alarmUntil {
		t.Errorf("a guest set the alarm off (trips %d)", run.trips)
	}
	if run.greenUntil <= g.time {
		t.Error("the arches did not blink green for the guest")
	}
}

// The guard only comes to look: he walks to the door, looks around and goes back to his round; the player who is
// out of his sight is never chased nor marked, and nobody dies.
func TestMetalDetGuardLooksAndGoesBack(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	g, run := mdLoad(t, campaignIdx("casino"), false, 7)
	if run == nil {
		t.Fatal("no arches")
	}
	g.time = 30
	mdCross(g, run, g.p, true)
	gd := run.chaser
	if gd == nil {
		t.Fatal("no guard sent")
	}
	g.p.pos = g.lvl.NearestFree(g.lvl.Def.Start) // outside the building, out of his sight
	dt := 1.0 / 60
	arrived := false
	for f := 0; f < 60*40; f++ {
		g.time += dt
		g.updateMetalDet(dt)
		restore := g.mdFocus(gd)
		g.updateGuard(gd, dt)
		restore()
		if gd.state == gdChase || gd.state == gdReact {
			t.Fatal("the guard chased the player he never saw")
		}
		if gd.state == gdWait || gd.state == gdPatrol {
			arrived = true
			break
		}
	}
	if !arrived {
		t.Fatalf("the guard never got to the door (state %d)", gd.state)
	}
	g.updateMetalDet(dt)
	if g.mdMarked(g.p) || run.chaser != nil {
		t.Error("the mark or the guard stayed after he looked")
	}
	if g.p.dead {
		t.Error("the player died")
	}
}

// Two players: the guard looks for the one who crossed, not the nearest one.
func TestMetalDetTwoPlayersChaseCrosser(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	g, run := mdLoad(t, campaignIdx("casino"), true, 7)
	if run == nil || g.p2 == nil {
		t.Fatal("no arches or no second player")
	}
	g.time = 30
	p1, p2 := g.first(), g.p2
	mdCross(g, run, p2, true)
	gd := run.chaser
	if gd == nil {
		t.Fatal("no guard sent")
	}
	if run.target != p2 || g.mdMarked(p2) || g.mdMarked(p1) {
		t.Fatal("the alarm is not about the one who crossed, or it marked somebody")
	}
	p1.pos = Vec{gd.pos.X + 6, gd.pos.Y} // J1 right beside the guard
	restore := g.mdFocus(gd)
	if g.p != p2 {
		t.Error("the guard does not look at the crosser")
	}
	restore()
	if g.p != p1 {
		t.Error("the focus was not restored")
	}
}

// Without guards an officer on scene comes to look at the door.
func TestMetalDetCopComesWithoutGuards(t *testing.T) {
	_ = partyGame(t)
	mdGlobals(t)
	g, run := mdLoad(t, campaignIdx("hockey"), false, 5)
	if run == nil {
		t.Fatal("no arches")
	}
	prev := g.guards
	t.Cleanup(func() { g.guards, g.cops = prev, nil })
	g.guards = nil
	c := &Cop{pos: g.lvl.NearestFree(g.lvl.Def.Start), state: cSweep}
	g.cops = []*Cop{c}
	g.time = 30
	mdCross(g, run, g.p, true)
	if run.cop != c || c.state != cSearch || c.icon == "" || g.mdMarked(g.p) {
		t.Errorf("officer not sent (state %d)", c.state)
	}
}
