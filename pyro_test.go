package main

import (
	"math"
	"testing"
)

func fireGame(t *testing.T, idx int, seed int64) (*Game, *LevelDef) {
	defs := testLevels(t)
	def := defs[idx]
	g := capGame(def, seed)
	g.as = &Assets{}
	g.levelIdx = idx
	g.pickStartTarget()
	g.fillGuests(def.Start)
	g.total = len(g.guests)
	g.blaze = fireWorld{}
	return g, def
}

func firstFlammable(g *Game) *Prop {
	for _, pr := range g.lvl.Props {
		if g.isFlammable(pr) {
			return pr
		}
	}
	return nil
}

// the fire spreads a little and never beyond its budget, its reach or the cap on flames
func TestFireSpreadBounded(t *testing.T) {
	for _, idx := range []int{0, 3, campaignIdx("roof")} {
		g, _ := fireGame(t, idx, int64(idx)+5)
		start := firstFlammable(g)
		if start == nil {
			t.Fatalf("level %d has nothing that burns", idx)
		}
		f := g.igniteProp(start, nil)
		if f == nil {
			t.Fatal("could not ignite")
		}
		origin := f.origin
		lit := map[*Prop]bool{start: true}
		for step := 0; step < 1200; step++ {
			g.updateFires(0.1)
			if n := g.liveFires(); n > maxFires {
				t.Fatalf("level %d: %d flames at once, cap %d", idx, n, maxFires)
			}
			for _, bf := range g.blaze.fires {
				if bf.prop != nil {
					lit[bf.prop] = true
				}
			}
		}
		if len(lit) > 1+fireSpreadMax {
			t.Errorf("level %d: the fire reached %d props, the budget is %d", idx, len(lit), 1+fireSpreadMax)
		}
		for pr := range lit {
			if dist(origin.pos, Vec{pr.X, pr.Y}) > fireOriginReach+1 {
				t.Errorf("level %d: %s caught fire %.0f px from where it started", idx, pr.Name, dist(origin.pos, Vec{pr.X, pr.Y}))
			}
		}
		if g.liveFires() != 0 {
			t.Errorf("level %d: the fire never went out by itself", idx)
		}
		for _, pr := range g.lvl.Props {
			if !flammable[pr.Name] && (pr.lit || pr.burn > 0) {
				t.Errorf("level %d: %s is not flammable but burnt", idx, pr.Name)
			}
		}
		if start.burn < 0.99 {
			t.Errorf("the burnt prop should stay charred (burn %.2f)", start.burn)
		}
	}
}

// metal, stone and water do not burn; what stands in a pool does not either
func TestNothingMetalBurns(t *testing.T) {
	g, _ := fireGame(t, 0, 1)
	for _, name := range []string{"fridge", "car", "dumpster", "statue", "toilet", "tv", "jukebox", "fountain", "fusebox"} {
		k := propKinds[name]
		if k == nil {
			continue
		}
		if g.isFlammable(&Prop{Name: name, K: k, X: 100, Y: 100}) {
			t.Errorf("%s should not burn", name)
		}
	}
	g2, def := fireGame(t, campaignIdx("roof"), 2) // the rooftop has a pool
	if len(g2.lvl.Pools) == 0 {
		t.Fatalf("%s has no pool (the rooftop should)", def.Title)
	}
	pc := g2.lvl.Pools[0]
	in := &Prop{Name: "sofa", K: propKinds["sofa"], X: pc.X + pc.W/2, Y: pc.Y + pc.H/2}
	if g2.isFlammable(in) {
		t.Errorf("a sofa in the pool of %s burnt", def.Title)
	}
}

// the firefighters arrive and put it out
func TestFirefightersPutItOut(t *testing.T) {
	for _, idx := range []int{0, campaignIdx("roof")} {
		g, _ := fireGame(t, idx, int64(idx)+9)
		pr := firstFlammable(g)
		f := g.igniteProp(pr, nil)
		f.life = 600 // it would burn for ever: only the hose ends it
		for step := 0; step < 2400 && !f.out; step++ {
			g.time += 0.1
			g.updateVehicles(0.1)
			g.updateFire(0.1)
			g.updateFires(0.1)
		}
		if !f.out {
			t.Errorf("level %d: the firefighters never put the fire out", idx)
		}
		if pr.burn <= 0 || pr.burn > 1 {
			t.Errorf("level %d: charred value %.2f", idx, pr.burn)
		}
	}
}

// people who stay in the flames die, but nobody gets credit and the guest count goes down
func TestFireKillsWithoutStars(t *testing.T) {
	g, _ := fireGame(t, 0, 4)
	pr := firstFlammable(g)
	f := g.igniteProp(pr, nil)
	f.life = 600
	var victims []*Guest
	for _, gu := range g.guests {
		if len(victims) < 12 && gu.perf == "" && gu.py == nil {
			victims = append(victims, gu)
		}
	}
	before := g.guestLive()
	kills := g.kills
	for step := 0; step < 600; step++ {
		for _, gu := range victims {
			if !gu.out() {
				gu.pos = Vec{f.pos.X + 2, f.pos.Y}
			}
		}
		g.updateFires(0.1)
	}
	if g.guestLive() >= before {
		t.Errorf("nobody died in a fire that lasted a minute (%d guests before and after)", before)
	}
	if g.kills != kills {
		t.Errorf("fire deaths must not count as the player's kills (%d -> %d)", kills, g.kills)
	}
	if g.stars != 0 || g.wanted != 0 {
		t.Errorf("fire deaths must not raise stars/wanted: %d/%v", g.stars, g.wanted)
	}
}

// the arsonist: does not count for the cap, nor for ending the party; he can be killed
func TestPyroIsNotAGuest(t *testing.T) {
	g, _ := fireGame(t, 3, 7)
	live := g.guestLive()
	alive := g.aliveCount()
	g.spawnPyro()
	if g.guestLive() != live || g.aliveCount() != alive {
		t.Errorf("the arsonist counted as a guest (%d/%d -> %d/%d)", live, alive, g.guestLive(), g.aliveCount())
	}
	var py *Guest
	for _, gu := range g.guests {
		if gu.py != nil {
			py = gu
		}
	}
	if py == nil {
		t.Fatal("no arsonist")
	}
	total := g.total
	g.killGuest(py, true)
	if !py.dead || g.total != total+1 {
		t.Errorf("killing the arsonist: dead=%v total %d -> %d", py.dead, total, g.total)
	}
	if g.progress.Stats["pyros"] != 1 {
		t.Errorf("the pyros stat did not go up")
	}
}

// he lights things on his own, stops after three and leaves
func TestPyroLightsAndLeaves(t *testing.T) {
	g, _ := fireGame(t, 3, 11)
	g.spawnPyro()
	var py *Guest
	for _, gu := range g.guests {
		if gu.py != nil {
			py = gu
		}
	}
	for step := 0; step < 12000 && !py.gone && !py.dead; step++ {
		g.time += 0.05
		g.updateGuest(py, 0.05)
		g.updateFires(0.05)
		if g.liveFires() >= 3 { // keep the test short: put them out as they come
			for _, f := range g.blaze.fires {
				f.putOut(g, true)
			}
		}
	}
	if g.blaze.lit != maxIgnitions {
		t.Errorf("he lit %d fires, want %d", g.blaze.lit, maxIgnitions)
	}
	if !py.gone {
		t.Errorf("the arsonist never left")
	}
}

// the chance of an arsonist: about a third of the parties, never the first one, forced by -pyronow
func TestPyroChance(t *testing.T) {
	defs := testLevels(t)
	count := 0
	const n = 600
	for i := 0; i < n; i++ {
		g := capGame(defs[3], int64(i))
		g.levelIdx = 3
		g.pyroPlan()
		if g.blaze.pyroAt > 0 {
			count++
		}
		g0 := capGame(defs[0], int64(i))
		g0.levelIdx = 0
		g0.pyroPlan()
		if g0.blaze.pyroAt > 0 {
			t.Fatal("an arsonist in the very first party")
		}
	}
	if r := float64(count) / n; math.Abs(r-pyroChance) > 0.08 {
		t.Errorf("arsonist in %.0f%% of the parties, want about %.0f%%", r*100, pyroChance*100)
	}
	pyroNow = true
	defer func() { pyroNow = false }()
	g := capGame(defs[0], 1)
	g.pyroPlan()
	if g.blaze.pyroAt <= 0 {
		t.Error("-pyronow did not force him")
	}
}

// nothing stays stuck for more than five seconds
func TestPyroUnstick(t *testing.T) {
	g, _ := fireGame(t, 3, 3)
	g.spawnPyro()
	var py *Guest
	for _, gu := range g.guests {
		if gu.py != nil {
			py = gu
		}
	}
	py.py.phase = 1
	p0 := py.pos
	py.py.lastPos = p0
	for i := 0; i < 70; i++ {
		g.pyroUnstick(py, 0.1)
	}
	if dist(py.pos, p0) < 1 {
		t.Error("an arsonist stuck for 7 seconds was not moved")
	}
}

// he is drawn as a solid person with a torch, not an empty sprite
func TestPyroVisible(t *testing.T) {
	sp := pyroSpec()
	cs := newCharSet(&sp)
	for _, vw := range []int{vFront, vSide, vBack} {
		b := genChar(cs.Spec, vw, poseFor(AnimWalk, 1))
		n := 0
		for i := 3; i < len(b.p); i += 4 {
			if b.p[i] > 0 {
				n++
			}
		}
		if n < 90 {
			t.Errorf("view %d: only %d opaque pixels", vw, n)
		}
	}
}
