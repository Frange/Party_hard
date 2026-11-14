package main

import (
	"testing"
)

// vipStyleSetup: restores the style globals.
func vipStyleSetup(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { vipStyleDecision = vipStyle{}; vipDecision = ""; vipStyleRng = nil })
}

// vipBuildStyle: the level of a definition with a zone of that style.
func vipBuildStyle(def *LevelDef, id string, st vipStyle) *Level {
	vipStyleDecision = st
	defer func() { vipStyleDecision = vipStyle{} }()
	return vipBuildWith(def, id)
}

// vipNoBlockedPaths: every walkable cell reachable from the start in the bare level is still reachable with the
// zone, except the cells of the zone itself and its cordon (flood fill like collision_test).
func vipNoBlockedPaths(t *testing.T, name string, bare, l *Level) {
	t.Helper()
	z := l.vip
	rb, rz := bare.reachFrom(bare.Def.Start), l.reachFrom(l.Def.Start)
	if len(rb) != len(rz) {
		t.Fatalf("%s: different grids", name)
	}
	skip := z.in.Grow(24)
	skip.H += 44
	lost := 0
	for i, ok := range rb {
		if !ok || rz[i] {
			continue
		}
		c := Vec{(float64(i%l.cw) + 0.5) * navCell, (float64(i/l.cw) + 0.5) * navCell}
		if skip.Has(c.X, c.Y) || !l.Free(c) {
			continue
		}
		lost++
	}
	if lost > 0 {
		t.Errorf("%s (%s): the zone cuts off %d walkable cells", name, z.shapeName(), lost)
	}
}

// Every shape the dice can ask for, on every level that has room: clean placement (vipQuality: overlaps, doors,
// start, performers, cameras, gate reachable) and no path cut off.
func TestVIPShapesQualityAllLevels(t *testing.T) {
	vipStyleSetup(t)
	styles := []vipStyle{
		{set: true, shape: vsRect, cols: 6, rows: 2, sec: vipSecHigh},
		{set: true, shape: vsRect, cols: 3, rows: 1, sec: vipSecLow},
		{set: true, shape: vsRound, cols: 4, rows: 3, sec: vipSecNormal},
		{set: true, shape: vsOval, cols: 6, rows: 3, sec: vipSecHigh, partyN: 2},
		{set: true, shape: vsL, cols: 5, rows: 3, flip: true, sec: vipSecNormal},
		{set: true, shape: vsL, cols: 4, rows: 2, sec: vipSecLow},
		{set: true, shape: vsPlatform, cols: 5, rows: 2, sec: vipSecHigh, partyN: 1},
	}
	shapes := map[string]int{}
	for i, d := range builtinLevels() {
		bare := buildLevel(d)
		for _, st := range styles {
			l := vipBuildStyle(d, "king", st)
			if l.vip == nil {
				continue
			}
			name := d.Title + "#" + itoa(i)
			shapes[l.vip.shapeName()]++
			vipQuality(t, name, l)
			vipNoBlockedPaths(t, name, bare, l)
			if !l.vip.legacy {
				for _, c := range l.vip.cells {
					if !l.vip.in.Grow(1).Hits(c) {
						t.Errorf("%s: a cell out of the zone box", name)
					}
				}
				if len(l.vip.vipSpots) < 1 {
					t.Errorf("%s: no place for the VIP", name)
				}
			}
		}
	}
	t.Logf("shapes placed: %v", shapes)
	for _, s := range []string{"rect", "round", "oval", "L", "platform"} {
		if shapes[s] < 3 {
			t.Errorf("shape %s fits in only %d levels", s, shapes[s])
		}
	}
}

// The dice give different shapes and sizes from one party to another (and the classic carpet is not the only one).
func TestVIPShapesVariety(t *testing.T) {
	vipStyleSetup(t)
	vipChanceSetup(t, 99)
	d := vipLevelDef(t, "FESTIVAL")
	shapes, sizes, secs, parties := map[string]bool{}, map[[2]float64]bool{}, map[int]bool{}, 0
	for k := 0; k < 80; k++ {
		st := vipRollStyle(vipStyleDice())
		l := vipBuildStyle(d, "rockstar", st)
		if l.vip == nil {
			t.Fatal("FESTIVAL has room for a zone")
		}
		shapes[l.vip.shapeName()] = true
		sizes[[2]float64{l.vip.in.W, l.vip.in.H}] = true
		secs[st.sec] = true
		if st.partyN > 0 {
			parties++
		}
	}
	t.Logf("shapes %v, %d sizes, %d parties in 80", shapes, len(sizes), parties)
	if len(shapes) < 4 || len(sizes) < 5 || len(secs) < 3 || parties == 0 || parties > 40 {
		t.Errorf("not enough variety: shapes %v sizes %d secs %v parties %d", shapes, len(sizes), secs, parties)
	}
}

// vipStaffCount: live VIPs and live guards of the zone.
func vipStaffCount(g *Game) (vips, guards int) {
	for _, gd := range g.guards {
		if gd.dead || gd.vip == 0 {
			continue
		}
		if gd.vip == vipBoss {
			vips++
		} else {
			guards++
		}
	}
	return
}

// A mini VIP party has 2-3 VIPs and more security than a single VIP; low security still has the two door guards.
func TestVIPMiniPartyMoreSecurity(t *testing.T) {
	vipStyleSetup(t)
	vipStyleDecision = vipStyle{set: true, shape: vsRect, cols: 3, rows: 1, sec: vipSecLow}
	low := vipGame(t, "FESTIVAL", false)
	lv, lg := vipStaffCount(low)
	if lv != 1 || lg < 2 {
		t.Errorf("low security: %d VIPs %d guards, want 1 and >= 2", lv, lg)
	}
	for _, n := range []int{1, 2} {
		vipStyleDecision = vipStyle{set: true, shape: vsOval, cols: 6, rows: 3, sec: vipSecHigh, partyN: n, extra: vipPickExtra(vipStyleDice(), nil, "rockstar", n)}
		g := vipGame(t, "FESTIVAL", false)
		v, gu := vipStaffCount(g)
		if v != 1+n {
			t.Errorf("party of %d: %d VIPs (shape %s)", 1+n, v, g.lvl.vip.shapeName())
		}
		if gu <= lg || gu < 2+2 {
			t.Errorf("party of %d: %d guards, want more than %d", 1+n, gu, lg)
		}
		ids := map[string]bool{}
		for _, gd := range vipStaff(g, vipBoss) {
			if ids[gd.vipID] || vipByID(gd.vipID) == nil {
				t.Errorf("party: repeated or unknown VIP %q", gd.vipID)
			}
			ids[gd.vipID] = true
			if !g.vipInZone(gd.pos) {
				t.Errorf("party: VIP %s outside the zone", gd.vipID)
			}
		}
		for _, gd := range vipStaff(g, vipInner) {
			if !g.vipInZone(gd.pos) {
				t.Error("party: inner guard outside the zone")
			}
		}
	}
	vipStyleDecision = vipStyle{}
}

// NEVER an empty zone: on every level (several seeds, so several shapes), one and two players, the zone has its VIP
// and its guards when the party starts and still has them after a while of party with the killers early (they
// used to take the staff and the VIP as ordinary guards: the empty zone the owner saw).
func TestVIPZoneNeverEmpty(t *testing.T) {
	vipStyleSetup(t)
	vipForce = true
	oldFast := fastEv
	fastEv = true
	t.Cleanup(func() { vipForce, fastEv = false, oldFast })
	zones, shapes := 0, map[string]bool{}
	for _, seed := range []int64{3, 11, 42} {
		vipChanceSetup(t, seed)
		for _, two := range []bool{false, true} {
			g := layerGame(t, two)
			for i := range levelDefs {
				g.loadLevel(i)
				z := g.lvl.vip
				if z == nil {
					continue
				}
				zones++
				shapes[z.shapeName()] = true
				name := levelDefs[i].Title + " " + z.shapeName()
				v, gu := vipStaffCount(g)
				if !z.spawned || v < 1 || gu < 2 {
					t.Fatalf("seed %d %s: empty zone at the start (%d VIPs, %d guards)", seed, name, v, gu)
				}
				if two || seed != 3 { // the long run once per level is enough
					continue
				}
				g.lvl.graded, g.titleT, g.mode, g.god = true, 0, ModePlay, true
				for k := 0; k < 60*25 && g.mode == ModePlay; k++ {
					g.updatePlay(1.0 / 60)
				}
				v, gu = vipStaffCount(g)
				if v < 1 || gu < 2 {
					t.Errorf("seed %d %s: the zone emptied by itself in 25 s (%d VIPs, %d guards)", seed, name, v, gu)
				}
				g.god = false
			}
		}
	}
	t.Logf("%d zones, shapes %v", zones, shapes)
	if zones < 30 || len(shapes) < 3 {
		t.Errorf("%d zones, shapes %v", zones, shapes)
	}
}

// The killers never pick the VIP staff (nor the VIP) as a victim.
func TestVIPKillersSkipTheStaff(t *testing.T) {
	g := vipGame(t, "FESTIVAL", false)
	s := &Slasher{pos: g.lvl.vip.posts[0]}
	for _, gd := range g.guards {
		s.pos = Vec{gd.pos.X, gd.pos.Y + 20}
		s.dir = DirUp
		_, od, _ := g.nearestSeen(s)
		if od != nil && od.vip != 0 {
			t.Fatal("the killer sees a VIP staff member as prey")
		}
		s.tgtG, s.tgtD = nil, nil
		g.slasherPick(s)
		if s.tgtD != nil && s.tgtD.vip != 0 {
			t.Fatal("the killer picked a VIP staff member")
		}
	}
}
