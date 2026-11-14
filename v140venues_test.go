package main

import (
	"math/rand"
	"testing"
)

// ---- CIRCUITO (racetrack.go) ----------------------------------------------------------------------------------------

// The circuit has no grey bridge and no stands at the bottom any more, and both halves stay reachable on foot.
func TestCircuitNoBridgeNoBottomStands(t *testing.T) {
	d := genNamed("circuit", campaignLevels[campaignIdx("circuit")].seed, "")
	if d == nil {
		t.Fatal("the circuit does not build")
	}
	top := 0
	for _, p := range d.Props {
		if p.Kind == "bridge" {
			t.Errorf("there is still a bridge at %.0f,%.0f", p.X, p.Y)
		}
		if p.Kind == "stands" {
			if p.Y > 285 {
				t.Errorf("stands at the bottom (%.0f,%.0f)", p.X, p.Y)
			} else {
				top++
			}
		}
	}
	if top == 0 {
		t.Error("the stands at the top are gone too")
	}
	l := buildLevel(d)
	if p := l.PathG(Vec{480, 110}, l.NearestFree(Vec{480, 440})); p == nil {
		t.Error("the bottom half cannot be reached from the top one")
	}
	if !raceTrack(d) {
		t.Error("the circuit is not a race track")
	}
	if hw := genNamed("highway", 131, ""); hw != nil && raceTrack(hw) {
		t.Error("the highway must keep its own traffic")
	}
}

// Race cars: often (well under the old 3-6 s) and fast (2.5x the 100 px/s of a plain car); a player on the track is
// warned when one comes in; a player off the track is not.
func TestCircuitRaceCarsFastOftenAndWarned(t *testing.T) {
	g := playGame(t, campaignIdx("circuit"), 7)
	prevFast := fastEv
	t.Cleanup(func() { fastEv = prevFast; g.amb, g.bubbles = nil, nil })
	fastEv = false
	g.amb, g.ambT = nil, 0
	seen := map[*Ambient]bool{}
	fast := 0
	for i := 0; i < 60*30; i++ { // 30 s
		g.updateAmbient(1.0 / 60)
		for _, a := range g.amb {
			if !seen[a] {
				seen[a] = true
				if a.kind == ambCar && a.v0 != 0 && (a.v0 >= raceSpeedMin || -a.v0 >= raceSpeedMin) {
					fast++
				}
			}
		}
	}
	if fast < 12 {
		t.Errorf("only %d race cars in 30 s (want a car every ~2 s)", fast)
	}
	// warning: the player on the track, a car comes in
	g.bubbles = nil
	g.p.pos = Vec{470, g.lvl.Def.CarFrom.Y + 10}
	g.p.dead, g.p.hidden = false, false
	g.raceWarn(&Ambient{kind: ambCar, x: -100, y: g.lvl.Def.CarFrom.Y + laneOffset, vx: 280, v0: 280})
	got := false
	for _, b := range g.bubbles {
		if b.txt == raceWarnText {
			got = true
		}
	}
	if !got {
		t.Error("the player on the track was not warned")
	}
	g.bubbles = nil
	g.p.pos = Vec{470, 110}
	g.raceWarn(&Ambient{kind: ambCar, x: -100, y: g.lvl.Def.CarFrom.Y + laneOffset, vx: 280, v0: 280})
	if len(g.bubbles) != 0 {
		t.Error("a player off the track was warned")
	}
}

// ---- CEMENTERIO (graveyard.go) --------------------------------------------------------------------------------------

func gyGame(t *testing.T) (*Game, *gyRun) {
	t.Helper()
	prev := [...]any{gyInTests, gyRng, gySeed, gy}
	t.Cleanup(func() {
		gyInTests, gyRng, gySeed, gy = prev[0].(bool), prev[1].(*rand.Rand), prev[2].(int64), prev[3].(*gyRun)
	})
	g := peLoad(t, campaignIdx("grave"), false)
	gyInTests, gyRng, gySeed, gy = true, nil, 4242, nil
	run := g.gyActive()
	if run == nil {
		t.Fatal("the graveyard has no night of the living dead")
	}
	return g, run
}

func TestGraveyardAlwaysHasItAndOthersDoNot(t *testing.T) {
	g, run := gyGame(t)
	g.updateGraveyard(0)
	if !run.started || run.zombieAt <= g.time || run.ghostAt <= run.zombieAt {
		t.Fatalf("bad schedule: zombies at %.0f, ghosts at %.0f (now %.0f)", run.zombieAt, run.ghostAt, g.time)
	}
	if gyLevel(levelDefs[campaignIdx("skate")]) || gyLevel(levelDefs[0]) {
		t.Error("only the graveyard has it")
	}
}

func TestGraveyardZombiesEatBrainsThenOnlyScare(t *testing.T) {
	g, run := gyGame(t)
	g.updateGraveyard(0)
	g.time = run.zombieAt + 0.01
	g.updateGraveyard(1.0 / 60)
	if n := len(run.zombies); n < 2 || n > 3 {
		t.Fatalf("%d zombies rose (want 2-3)", n)
	}
	for _, z := range run.zombies {
		if z.state != gZombie {
			t.Fatal("a risen dead is not a zombie")
		}
	}
	z := run.zombies[0]
	var victims []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.state != gZombie && len(victims) < gyBrainCap+1 {
			victims = append(victims, gu)
		}
	}
	if len(victims) < gyBrainCap+1 {
		t.Fatal("not enough guests")
	}
	for i, v := range victims {
		if !g.gyBite(z, v) {
			t.Fatal("the graveyard zombie's bite was not handled")
		}
		if i < gyBrainCap && !v.dead {
			t.Errorf("bite %d: the brain was not eaten", i)
		}
		if i == gyBrainCap && (v.dead || v.state == gZombie || v.state != gScared) {
			t.Errorf("past the cap the bite must only scare (dead %v, state %d)", v.dead, v.state)
		}
	}
	// an ordinary zombie (not one of the graveyard's) still turns people
	other := &Guest{}
	if g.gyBite(other, victims[0]) {
		t.Error("an ordinary zombie's bite was taken by the graveyard")
	}
}

func TestGraveyardGhostsOnlyScare(t *testing.T) {
	g, run := gyGame(t)
	g.updateGraveyard(0)
	g.gySpawnGhosts(run, gyDice())
	if len(run.ghosts) != 2 {
		t.Fatalf("%d ghosts", len(run.ghosts))
	}
	gh := run.ghosts[0]
	var near *Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.state != gZombie && !g.pevTaken(gu) {
			near = gu
			break
		}
	}
	near.state = gIdle
	gh.x, gh.y = near.pos.X+10, near.pos.Y
	alive := g.aliveCount()
	g.gyScare(run, gh, gyDice())
	if near.state != gScared || near.dead {
		t.Errorf("the guest next to the ghost did not run away screaming (state %d, dead %v)", near.state, near.dead)
	}
	if g.aliveCount() != alive {
		t.Error("a ghost hurt somebody")
	}
}

func TestGraveyardHuntersTrapGhostsAndSometimesZapAGuest(t *testing.T) {
	g, run := gyGame(t)
	g.updateGraveyard(0)
	g.gySpawnGhosts(run, gyDice())
	g.gyCallCrew(run)
	if len(run.crew) == 0 {
		t.Fatal("no ghost hunters came")
	}
	slugs := map[string]int{}
	for _, gu := range g.guests {
		if !gu.out() {
			slugs[guestSlug(gu)]++
		}
	}
	for _, s := range ghostbusterSlugs {
		if slugs[s] > 1 {
			t.Errorf("%s is twice at the party", s)
		}
	}
	// a hunter next to a ghost traps it
	gh := run.ghosts[0]
	h := run.crew[0]
	gh.x, gh.y = h.pos.X+30, h.pos.Y
	for i := 0; i < 60*int(gyGhostHP+2) && !gh.caught; i++ {
		h.pos = Vec{gh.x - 30, gh.y}
		h.state = gIdle
		g.gyCrewStep(run, gyDice(), 1.0/60)
	}
	if !gh.caught {
		t.Error("the ghost was never trapped")
	}
	// the confusion: a guest right by the ghost is fried, never more than gyZapCap
	gh2 := run.ghosts[len(run.ghosts)-1]
	// the trapping above runs gyCrewStep with the real dice, which may already have zapped somebody by the beam: the
	// count starts here (otherwise the cap was reached early and the zaps below were refused, one run in a few)
	run.zaps = 0
	for k := 0; k < gyZapCap+2; k++ {
		var v *Guest
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" && gu.state != gLift && !g.gyIsCrew(run, gu) {
				v = gu
				break
			}
		}
		v.pos = Vec{gh2.x + 5, gh2.y}
		// gyZap fries the first guest of the list by the ghost OR on the beam: another guest who happened to stand
		// there (the layout and the crowd change from run to run) was hit instead of v, one run in a few
		for _, gu := range g.guests {
			if gu != v && !gu.out() && !g.gyIsCrew(run, gu) &&
				(dist(gu.pos, Vec{gh2.x, gh2.y}) < 60 || segDist(gu.pos, h.pos, Vec{gh2.x, gh2.y}) < 20) {
				gu.pos = Vec{-1000, -1000}
			}
		}
		g.gyZap(run, h, gh2)
		if k < gyZapCap && !v.dead {
			t.Errorf("zap %d: the guest by the ghost was not hit", k)
		}
	}
	if run.zaps != gyZapCap {
		t.Errorf("%d guests zapped (cap %d)", run.zaps, gyZapCap)
	}
}

// ---- SKATE PARK (skategraffiti.go) ----------------------------------------------------------------------------------

func TestSkateParkGraffitiWallsAndCrewAlways(t *testing.T) {
	g := gfGame(t, campaignIdx("skate"), false, 3)
	if len(gfLv.segs) < 2 {
		t.Fatalf("the skate park has %d paintable walls (want the back wall and the free-standing one)", len(gfLv.segs))
	}
	for i := 0; i < 20; i++ {
		gfLv = gfLevel{}
		gfReset(g)
		if !gfLv.want || gfLv.max < 1 || gfLv.max > 3 {
			t.Fatalf("party %d: want %v, %d painters (want always, 1-3)", i, gfLv.want, gfLv.max)
		}
	}
	// the bowls and rails are still there and nothing stands on anything
	d := levelDefs[campaignIdx("skate")]
	n := 0
	for _, p := range d.Props {
		if p.Kind == "skatebowl" || p.Kind == "rail" {
			n++
		}
	}
	if n != 5 {
		t.Errorf("%d bowls and rails (want 5)", n)
	}
	if ov := levelOverlaps(buildLevel(d)); len(ov) > 0 {
		t.Errorf("overlaps: %v", ov)
	}
}
