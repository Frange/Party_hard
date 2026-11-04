package main

import (
	"math"
	"math/rand"
	"testing"
)

func lum(b *pixBuf, x, y int) float64 {
	i := (y*b.w + x) * 4
	return 0.3*float64(b.p[i]) + 0.59*float64(b.p[i+1]) + 0.11*float64(b.p[i+2])
}

// a bowl is a sunken pool: a dark flat floor inside a lighter rim, not a spike or a ramp
func TestSkateBowlIsConcave(t *testing.T) {
	for v := 0; v < 3; v++ {
		b := newPix(bowlW, bowlH)
		drawSkateBowl(b, v)
		opaque := 0
		for i := 3; i < len(b.p); i += 4 {
			if b.p[i] > 0 {
				opaque++
			}
		}
		if opaque < bowlW*bowlH/3 {
			t.Fatalf("variant %d: the bowl is too small to read (%d px)", v, opaque)
		}
		floor := lum(b, bowlW/2, bowlH/2+6)
		rim := lum(b, bowlW/2, 3) // the far rim (also checked at the kidney's notch below)
		if v == 1 {
			rim = lum(b, 14, bowlH/2-10)
		}
		if floor >= rim-30 {
			t.Errorf("variant %d: the floor (%.0f) is not darker than the rim (%.0f): it does not look sunken", v, floor, rim)
		}
		// the middle of the floor is solid colour: the bowl is a flat pit, so the floor is wide
		flat := 0
		for x := bowlW/2 - 12; x < bowlW/2+12; x++ {
			if lum(b, x, bowlH/2+6) < 100 {
				flat++
			}
		}
		if flat < 18 {
			t.Errorf("variant %d: the floor of the bowl is not a wide flat area", v)
		}
	}
}

// the skate park uses bowls, not the old pointed ramps
func TestSkateParkHasBowls(t *testing.T) {
	d := genNamed("skatepark", 211, "SKATE PARK")
	if d == nil {
		t.Fatal("the skate park does not generate")
	}
	bowls := 0
	for _, p := range d.Props {
		switch p.Kind {
		case "skatebowl":
			bowls++
		case "ramp", "halfpipe":
			t.Errorf("a pointed %s is still in the skate park", p.Kind)
		}
	}
	if bowls < 3 {
		t.Errorf("only %d bowls", bowls)
	}
}

func skateGame(seed int64) *Game {
	progressFrozen = true
	d := genNamed("skatepark", 211, "SKATE PARK")
	g := &Game{rng: rand.New(rand.NewSource(seed)), lvl: buildLevel(d), snd: &Sounds{muted: true}}
	g.p = &Player{pos: Vec{480, 440}}
	g.mode = ModePlay
	return g
}

func inAnyBowl(l *Level, p Vec) bool { return l.InBowl(p) }

// the guests' paths go round the bowls (even to a goal inside one); everybody else's go through
func TestBowlsAreClosedToGuests(t *testing.T) {
	g := skateGame(1)
	l := g.lvl
	if len(l.bowls) != 3 {
		t.Fatalf("%d bowl zones", len(l.bowls))
	}
	rng := rand.New(rand.NewSource(2))
	randPt := func() Vec { return Vec{70 + rng.Float64()*820, 100 + rng.Float64()*300} }
	found, tried, through := 0, 0, 0
	for i := 0; i < 600; i++ {
		a, b := randPt(), randPt()
		path := l.PathG(a, b)
		if path == nil {
			continue
		}
		cur := l.outOfBowls(a)
		for _, w := range path {
			for k := 1; k <= 20; k++ {
				q := Vec{cur.X + (w.X-cur.X)*float64(k)/20, cur.Y + (w.Y-cur.Y)*float64(k)/20}
				if inAnyBowl(l, q) {
					t.Fatalf("a guest path crosses a bowl at (%.0f,%.0f)", q.X, q.Y)
				}
			}
			cur = w
		}
		tried++
		if !l.InBowl(a) && !l.InBowl(b) {
			found++
		}
		// the normal grid does use the bowls
		for _, w := range l.Path(a, b) {
			if inAnyBowl(l, w) {
				through++
				break
			}
		}
	}
	if tried < 500 {
		t.Errorf("guests found a path in only %d of 600 trips", tried)
	}
	if through == 0 {
		t.Error("the player, the police and the killers cannot cross a bowl either")
	}
	_ = found
	// the player walks into the middle of a bowl
	c := Vec{l.bowls[0].X + l.bowls[0].W/2, l.bowls[0].Y + l.bowls[0].H/2}
	if l.SolidP(c.X, c.Y) || !l.Free(c) {
		t.Error("the floor of the bowl traps the player")
	}
}

// 20 guests wandering for four minutes never set foot in a bowl
func TestGuestsNeverEnterBowls(t *testing.T) {
	g := skateGame(7)
	for i := 0; i < 20; i++ {
		sp := genericSpec(g.rng)
		pos := g.lvl.RandomRoam(g.rng)
		g.guests = append(g.guests, &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: newCharSet(&sp), state: gIdle, timer: g.rng.Float64() * 3, off: g.rng.Float64() * 6})
	}
	// start with some of them inside a bowl: they are taken out at once
	for i := 0; i < 4; i++ {
		g.guests[i].pos = Vec{g.lvl.bowls[i%3].X + 50, g.lvl.bowls[i%3].Y + 30}
	}
	travelled, near := 0.0, 1e9
	last := map[*Guest]Vec{}
	for tick := 0; tick < 30*240; tick++ {
		for _, gu := range g.guests {
			g.updateGuest(gu, 1.0/30)
			if g.lvl.InBowl(gu.pos) {
				t.Fatalf("tick %d: a guest is inside a bowl at (%.0f,%.0f)", tick, gu.pos.X, gu.pos.Y)
			}
			if p, ok := last[gu]; ok {
				travelled += dist(p, gu.pos)
			}
			last[gu] = gu.pos
			for _, z := range g.lvl.bowls {
				near = math.Min(near, distRect(gu.pos, z.Grow(0)))
			}
		}
		g.time += 1.0 / 30
	}
	if travelled < 20*400 || near > 30 {
		t.Errorf("the test proves nothing: they walked %.0f px in total and came as close as %.0f px to a bowl", travelled, near)
	}
}

func kidGuest(g *Game, slug string, pos Vec) *Guest {
	spec := presetByName(slug)(g.rng)
	gu := &Guest{pos: pos, dir: DirDown, cs: newCharSet(&spec), name: slug, state: gIdle, timer: 5}
	g.guests = append(g.guests, gu)
	return gu
}

// a kid goes to a bowl now and then, rides it and leaves; while he rides he does not run from the player
func TestKidRidesTheBowlAndIgnoresThePlayer(t *testing.T) {
	g := skateGame(3)
	gu := kidGuest(g, "diego", Vec{120, 330})
	g.time = 1
	step := func() {
		g.updateGuest(gu, 1.0/30)
		g.time += 1.0 / 30
	}
	riding, rode, left := 0, false, false
	for tick := 0; tick < 30*600 && !left; tick++ {
		st := kidSessions[gu]
		if st != nil && st.stage == 2 {
			rode = true
			riding++
			// the player stands right in front of him, close
			v := dirVec(gu.dir)
			g.p.pos = Vec{gu.pos.X + v.X*30, gu.pos.Y + v.Y*30}
			if !g.lvl.InBowl(gu.pos) {
				t.Fatalf("the kid left the bowl while riding: (%.0f,%.0f)", gu.pos.X, gu.pos.Y)
			}
		} else if rode && st == nil {
			left = true
		} else {
			g.p.pos = Vec{480, 440}
		}
		step()
		if st != nil && st.stage == 2 && (gu.state == gScared || gu.state == gFlee || gu.state == gRun) {
			t.Fatal("the kid ran away from the player while doing his tricks")
		}
	}
	if !rode || riding < 30*10 {
		t.Fatalf("the kid never rode a bowl (rode %v, %d frames)", rode, riding)
	}
	if !left {
		t.Fatal("the kid never left the bowl")
	}
	// back to normal: he runs from a player who is in front of him
	scared := false
	for tick := 0; tick < 60 && !scared; tick++ {
		v := dirVec(gu.dir)
		g.p.pos = Vec{gu.pos.X + v.X*40, gu.pos.Y + v.Y*40}
		kidBowlNext[gu] = g.time + 1e6
		step()
		scared = gu.state == gScared
	}
	if !scared {
		t.Error("after the routine the kid is no longer afraid of the player")
	}
}

// he can be killed in the bowl like always: a victim, a trophy count, a body in the bowl; nobody calls the police
func TestKidDiesInTheBowl(t *testing.T) {
	g := skateGame(5)
	gu := kidGuest(g, "cesar", Vec{120, 330})
	g.total = len(g.guests)
	g.time = 1
	for tick := 0; tick < 30*600; tick++ {
		g.updateGuest(gu, 1.0/30)
		g.time += 1.0 / 30
		if kidRiding(gu) {
			break
		}
	}
	if !kidRiding(gu) {
		t.Fatal("the kid never rode")
	}
	before := g.aliveCount()
	g.p.pos = Vec{gu.pos.X + 5, gu.pos.Y}
	g.stab()
	if !gu.dead {
		t.Fatal("the player could not kill the kid in the bowl")
	}
	if g.kills != 1 || g.aliveCount() != before-1 || g.progress.Stats["kill_cesar"] != 1 {
		t.Errorf("kills %d, alive %d (was %d), kill_cesar %d", g.kills, g.aliveCount(), before, g.progress.Stats["kill_cesar"])
	}
	if len(g.bodies) != 1 || !g.lvl.InBowl(g.bodies[0].pos) {
		t.Error("the body is not in the bowl")
	}
	if g.policeCalled || g.callerGuest != nil {
		t.Error("the police were called")
	}
}
