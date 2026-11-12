package main

import (
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the cleaning crew (cleaners.go): real games after loadLevel, several levels and seeds.

func TestCleanersRegisteredSometimesAndSomeLevels(t *testing.T) {
	g := partyGame(t)
	e := partyByID("cleaners")
	if e == nil {
		t.Fatal("not registered")
	}
	if e.Prob < 0.10 || e.Prob > 0.40 || e.Cooldown < 120 || e.AtMin < 60 || e.Big {
		t.Errorf("it must be occasional: %+v", e)
	}
	in := peEligible(g, "cleaners")
	n := len(campaignLevels)
	if len(in) == 0 || len(in) >= n {
		t.Errorf("it should fit some levels, not all: %v", in)
	}
	for _, i := range in {
		if ic := levelDefs[i].Icon; cleanNoIcons[ic] {
			t.Errorf("level %d (%s) should not have cleaners", i, ic)
		}
	}
	if e.Eligible(g, partyTraits{}) {
		t.Error("it fits an empty level")
	}
}

// The uniform is light mint and has nothing of the military olive.
func TestCleanersLookNothingLikeTheMilitary(t *testing.T) {
	sp := cleanerSpec(rand.New(rand.NewSource(3)))
	lum := func(c [3]float64) float64 { return 0.3*c[0] + 0.59*c[1] + 0.11*c[2] }
	rgbOf := func(r, g, b uint8) [3]float64 { return [3]float64{float64(r), float64(g), float64(b)} }
	mint := rgbOf(sp.Shirt.R, sp.Shirt.G, sp.Shirt.B)
	if !(mint[1] > mint[0] && mint[1] > mint[2]*0.9 && lum(mint) > 170) {
		t.Errorf("the shirt %v is not a light green", sp.Shirt)
	}
	for _, m := range unitSpecs().military {
		for _, mc := range [][3]float64{rgbOf(m.Shirt.R, m.Shirt.G, m.Shirt.B), rgbOf(m.Pants.R, m.Pants.G, m.Pants.B)} {
			for _, cc := range [][3]float64{mint, rgbOf(sp.Pants.R, sp.Pants.G, sp.Pants.B)} {
				d := math.Sqrt((cc[0]-mc[0])*(cc[0]-mc[0]) + (cc[1]-mc[1])*(cc[1]-mc[1]) + (cc[2]-mc[2])*(cc[2]-mc[2]))
				if d < 120 || lum(cc)-lum(mc) < 80 {
					t.Errorf("cleaner colour %v too close to the military %v", cc, mc)
				}
			}
		}
		if sp.Hat == m.Hat || sp.Top == m.Top {
			t.Error("the cleaner wears the military helmet or armour")
		}
	}
	if sp.Weapon != 0 || sp.Carry != HoldBroom || sp.Top != TopVest {
		t.Error("a cleaner carries a mop and wears the striped vest, never a weapon")
	}
}

// clLevels: some eligible campaign levels (spread over the list).
func clLevels(g *Game) []int {
	in := peEligible(g, "cleaners")
	var out []int
	for k, i := range in {
		if k%4 == 0 {
			out = append(out, i)
		}
	}
	return out
}

// clMess: bodies the crew may take (reported, or not the player's), one unreported kill of the player and a stain.
func clMess(t *testing.T, g *Game, seed int64) (playerBody *Body) {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < 4; i++ {
		sp, ok := g.pevSpot(r, 14, false)
		if !ok {
			t.Fatal("no spot for a body")
		}
		g.bodies = append(g.bodies, &Body{pos: sp, cs: g.guests[r.Intn(len(g.guests))].cs, dir: DirDown, reported: i%2 == 0, byPlayer: i == 3, born: g.time})
		g.decals = append(g.decals, Decal{pos: sp, kind: 0, r: 10})
	}
	if sp, ok := g.pevSpot(r, 10, false); ok {
		g.decals = append(g.decals, Decal{pos: sp, kind: 2, r: 8})
	}
	return g.bodies[len(g.bodies)-1]
}

// clStart: an eligible level with mess and the crew forced in (nil if the crew could not come: no room).
func clStart(t *testing.T, idx int, two bool, seed int64) (*Game, *cleanState, *Body) {
	t.Helper()
	g := peLoad(t, idx, two)
	g.rng = rand.New(rand.NewSource(seed*31 + int64(idx)))
	partyRng = rand.New(rand.NewSource(seed*7919 + int64(idx)))
	g.time = 20
	pb := clMess(t, g, seed+int64(idx))
	peForce(g, "cleaners")
	for s := 0; s < 30*20 && g.partyRunning("cleaners") == nil && len(g.pev.plan) > 0; s++ {
		peStep(g, 1.0/30, true)
	}
	a := g.partyRunning("cleaners")
	if a == nil {
		return g, nil, pb
	}
	return g, a.D.(*cleanState), pb
}

// clRun runs the crew to its end with the checks of every frame.
func clRun(t *testing.T, g *Game, s *cleanState, pb *Body, frames int, each func(f int)) {
	t.Helper()
	scr := ebiten.NewImage(screenW, screenH)
	for f := 0; f < frames && g.partyRunning("cleaners") != nil; f++ {
		peStep(g, 1.0/30, true)
		for _, b := range s.bots {
			gu := b.mem.gu
			if gu.out() {
				continue
			}
			if clInSolid(g.lvl, gu.pos) {
				for _, r := range g.lvl.solidsP {
					if r.Has(gu.pos.X, gu.pos.Y) {
						t.Logf("solid %v", r)
					}
				}
				for _, pr := range g.lvl.Props {
					if pr.Foot.Has(gu.pos.X, gu.pos.Y) {
						t.Logf("prop %s %v", pr.Name, pr.Foot)
					}
				}
				t.Fatalf("level %d: a cleaner walks through something at %v (frame %d phase %d state %d path %v)", g.levelIdx, gu.pos, f, b.phase, gu.state, gu.path)
			}
			if b.carry && b.body != nil && b.body.byPlayer && !b.body.reported {
				t.Fatalf("level %d: a cleaner hides an unreported kill of the player", g.levelIdx)
			}
			if f%6 == 0 {
				p := gu.pos
				g.pevGuestDraw(scr, gu, &p)
			}
		}
		if g.guestLive() > g.guestCap() || g.total > g.goalMax() {
			t.Fatalf("level %d: caps broken: live %d/%d total %d/%d", g.levelIdx, g.guestLive(), g.guestCap(), g.total, g.goalMax())
		}
		if f%15 == 0 {
			peDraw(g)
		}
		if each != nil {
			each(f)
		}
	}
	if g.partyRunning("cleaners") != nil {
		t.Errorf("level %d: the crew never left", g.levelIdx)
	}
	for _, b := range s.bots {
		if gu := b.mem.gu; !gu.out() {
			t.Errorf("level %d: a cleaner stayed after the end at %v", g.levelIdx, gu.pos)
		}
	}
}

func TestCleanersCleanLeaveAndKeepTheKills(t *testing.T) {
	prev := capBreaches
	t.Cleanup(func() { capBreaches = prev })
	probe := partyGame(t)
	worked := 0
	for _, idx := range clLevels(probe) {
		for seed := int64(1); seed <= 2; seed++ {
			capBreaches = 0
			g, s, pb := clStart(t, idx, false, seed)
			if s == nil {
				t.Errorf("level %d seed %d: the crew did not come", idx, seed)
				continue
			}
			if len(s.bots) < 1 || len(s.bots) > 3 {
				t.Fatalf("%d cleaners", len(s.bots))
			}
			kills0 := g.total - g.aliveCount() - clOthersOut(g, s)
			bodies0, decals0 := len(g.bodies), len(g.decals)
			clRun(t, g, s, pb, 30*160, nil)
			// BAJAS moves only with the people of the party who died or left (other events of the frame may kill): never
			// with the bodies the crew takes away, nor with the cleaners who go home
			if k := g.total - g.aliveCount() - clOthersOut(g, s); k != kills0 {
				t.Errorf("level %d: BAJAS minus the dead went from %d to %d", idx, kills0, k)
			}
			if capBreaches != 0 {
				t.Errorf("level %d: the cap safety net acted", idx)
			}
			t.Logf("level %d seed %d: %d cleaners, bodies %d->%d (removed %d), stains %d->%d, vanished %d",
				idx, seed, len(s.bots), bodies0, len(g.bodies), s.removed, decals0, len(g.decals), s.vanished)
			if s.removed > 0 {
				worked++
			}
			if !pb.reported && !cleanInBodies(g, pb) {
				t.Errorf("level %d: the unreported kill of the player was taken away", idx)
			}
		}
	}
	if worked < 2 {
		t.Errorf("the crew picked up bodies in only %d parties", worked)
	}
}

func TestCleanersNoRoomNoCrew(t *testing.T) {
	probe := partyGame(t)
	idx := clLevels(probe)[0]
	g := peLoad(t, idx, false)
	g.time = 20
	clMess(t, g, 5)
	g.total = g.goalMax() // the kill target is full: nobody else may come
	peForce(g, "cleaners")
	for s := 0; s < 30*10; s++ {
		peStep(g, 1.0/30, true)
	}
	if g.partyRunning("cleaners") != nil {
		t.Error("the crew came with no room in the party")
	}
}

func TestCleanersCanBeKilledCarrying(t *testing.T) {
	probe := partyGame(t)
	done := false
	for _, idx := range clLevels(probe) {
		g, s, pb := clStart(t, idx, false, 3)
		if s == nil {
			continue
		}
		var victim *cleanBot
		var bag *Body
		clRun(t, g, s, pb, 30*160, func(f int) {
			if victim != nil {
				return
			}
			for _, b := range s.bots {
				if b.carry && b.body != nil && !b.mem.gu.out() {
					victim, bag = b, b.body
					k0 := g.total - g.aliveCount()
					g.killGuest(b.mem.gu, true)
					if k := g.total - g.aliveCount(); k != k0+1 {
						t.Errorf("killing a cleaner is not a kill: %d -> %d", k0, k)
					}
					return
				}
			}
		})
		if victim == nil {
			continue
		}
		if bag.carried || !cleanInBodies(g, bag) {
			t.Error("the bag of a dead cleaner was not left on the floor")
		}
		found := false
		for _, b := range g.bodies {
			if b.cs == victim.mem.gu.cs || b.cs == victim.csBag {
				found = true
			}
		}
		if !found {
			t.Error("the dead cleaner left no body")
		}
		done = true
		break
	}
	if !done {
		t.Error("no cleaner ever carried a bag")
	}
}

func TestCleanersTwoPlayersAndRng(t *testing.T) {
	probe := partyGame(t)
	idx := clLevels(probe)[1]
	g, s, pb := clStart(t, idx, true, 4)
	if s == nil {
		t.Fatal("the crew did not come (2P)")
	}
	if !g.is2P() {
		t.Fatal("not two players")
	}
	if g.guestCap() <= g.guestMax() {
		t.Error("2P cap missing")
	}
	g.rng = rand.New(rand.NewSource(99))
	clRun(t, g, s, pb, 30*160, nil)
	if got, want := g.rng.Int63(), rand.New(rand.NewSource(99)).Int63(); got != want {
		// the guests themselves use g.rng (peStep updates them): run once more without them to check the crew alone
		g2, s2, _ := clStart(t, idx, false, 4)
		if s2 == nil {
			t.Fatal("no crew")
		}
		g2.rng = rand.New(rand.NewSource(99))
		for f := 0; f < 30*30; f++ {
			peStep(g2, 1.0/30, false)
		}
		if got2, want2 := g2.rng.Int63(), rand.New(rand.NewSource(99)).Int63(); got2 != want2 {
			t.Error("the crew moved g.rng")
		}
		_ = got
		_ = want
	}
}

// A cleaner who cannot walk (boxed in) leaves the party instead of standing there for ever.
func TestCleanersStuckOneLeaves(t *testing.T) {
	probe := partyGame(t)
	idx := clLevels(probe)[0]
	g, s, pb := clStart(t, idx, false, 6)
	if s == nil {
		t.Fatal("the crew did not come")
	}
	b := s.bots[0]
	total0 := g.total
	// pin him: every frame he is put back where he is
	pin := b.mem.gu.pos
	clRun(t, g, s, pb, 30*160, func(f int) {
		if !b.mem.gu.out() {
			b.mem.gu.pos = pin
		}
	})
	if !b.mem.gu.gone && !b.mem.gu.dead {
		t.Error("the stuck cleaner never left")
	}
	if s.vanished == 0 {
		t.Error("no cleaner vanished")
	}
	if g.total > total0 {
		t.Error("the total grew")
	}
}

// clInSolid: the feet of somebody are inside a solid thing (a wall, the footprint of a prop). The walk of every guest
// may graze a corner by a fraction of a pixel (the path is smoothed): 1.5 px of the edge are tolerated.
func clInSolid(l *Level, p Vec) bool {
	for _, r := range l.solidsP {
		if r.Grow(-1.5).Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

// clOthersOut: the people of the party (not the crew) who are out (dead or gone), and the dead guards.
func clOthersOut(g *Game, s *cleanState) int {
	n := 0
	for _, gu := range g.guests {
		if gu.out() && gu.py == nil && (!s.isBot(gu) || gu.dead) { // a cleaner who goes home is not a kill; a dead one is
			n++
		}
	}
	for _, gd := range g.guards {
		if gd.dead && gd.vip == 0 {
			n++
		}
	}
	return n
}
