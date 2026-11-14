package main

import (
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------------------------------------------
// v1.4.0 tanda H (user feedback): space station without helipads and with space things and a living sky; and the
// shared check that a touched level is still whole (reachable, nothing on anything).
// ---------------------------------------------------------------------------------------------------------------

// venueWhole: the layout has no unreachable spot, no overlaps, and every roam area, dance floor, point of interest
// and performer can be walked to from the start (flood fill of the player's real movement).
func venueWhole(t *testing.T, label string, d *LevelDef) *Level {
	t.Helper()
	if bad := genBad(d); len(bad) > 0 {
		t.Errorf("%s: unreachable %v", label, bad)
	}
	l := buildLevel(d)
	if iss := levelOverlaps(l); len(iss) > 0 {
		t.Errorf("%s: %d overlaps, first %v", label, len(iss), iss)
	}
	var pts []Vec
	for _, r := range d.Roam {
		pts = append(pts, r.Center())
	}
	for _, r := range d.Dance {
		pts = append(pts, r.Center())
	}
	for _, ps := range l.POIs {
		pts = append(pts, ps...)
	}
	for _, pf := range d.Performers {
		pts = append(pts, pf.Pos)
	}
	seen, w, _ := playerReach(l, []Vec{d.Start})
	for _, p := range pts {
		q := l.NearestFree(p)
		if dist(p, q) > 24 || l.Path(d.Start, q) == nil {
			t.Errorf("%s: %v cannot be reached from the start", label, p)
			continue
		}
		if c := int(q.Y/2)*w + int(q.X/2); c < 0 || c >= len(seen) || !seen[c] {
			t.Errorf("%s: the player cannot walk to %v", label, q)
		}
	}
	return l
}

func propCount(d *LevelDef) map[string]int {
	n := map[string]int{}
	for _, p := range d.Props {
		n[p.Kind]++
	}
	return n
}

// 1. the station: no helipads (everything docks in a ship), space props, and the sky moves
func TestStationSpaceLookNoHelipads(t *testing.T) {
	e := campaignLevels[campaignIdx("station")]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2} {
		d := genNamed("station", seed, "")
		if d == nil {
			t.Fatalf("seed %d: no layout", seed)
		}
		if d.Police == "heli" {
			t.Errorf("seed %d: the police come by helicopter", seed)
		}
		for _, f := range d.Floors {
			if f.Kind == "helipad" {
				t.Errorf("seed %d: helipad floor at %v", seed, f.R)
			}
		}
		if d.Transit != "space" {
			t.Errorf("seed %d: transit %q, want space ships", seed, d.Transit)
		}
		n := propCount(d)
		for _, k := range []string{"srocket", "sdome", "smodule", "ssatdish", "swindow"} {
			if n[k] < 1 {
				t.Errorf("seed %d: no %s in the station", seed, k)
			}
		}
		l := venueWhole(t, "station", d)
		for _, p := range l.Props {
			if !d.Shape(p.X, p.Y-2, 0) {
				t.Errorf("seed %d: %s at %.0f,%.0f floats in space", seed, p.Name, p.X, p.Y)
			}
		}
	}
	// the sky is alive: the drawing changes with time and never paints over the station
	d := genNamed("station", e.seed, "")
	for _, s := range spaceFor(d).stars {
		if d.Shape(s.x, s.y, 14) {
			t.Fatalf("a star over the station at %v,%v", s.x, s.y)
		}
	}
	if len(spaceFor(d).stars) < 80 {
		t.Errorf("only %d twinkling stars", len(spaceFor(d).stars))
	}
	for _, lane := range spaceShipLanes {
		for x := -40.0; x < 1000; x += 4 {
			if !spaceClear(d, x-10, lane-4, 22, 9) && x > 0 && x < 960 {
				t.Fatalf("the ship lane y=%v crosses the station at x=%v", lane, x)
			}
		}
	}
}

// the station's sky draws something at every moment, a ship shows up in the lanes, and it costs nothing when off
func TestStationSkyMoves(t *testing.T) {
	g := ambientSetup(t)
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	g.loadLevel(ambientLevelIdx("ESTACIÓN ESPACIAL"))
	g.lvl.graded = true
	t.Cleanup(func() { spaceCur = nil })
	ships := 0
	for tm := 0.0; tm < 30; tm += 0.5 {
		g.time = tm
		spaceFxCount = 0
		g.drawBackAnim(scr)
		if spaceFxCount < 100 {
			t.Fatalf("t=%.1f: only %d space effects drawn", tm, spaceFxCount)
		}
		if math.Mod(tm, spaceShipEvery) < 5 {
			ships++
		}
	}
	if ships == 0 {
		t.Error("no ship crossed in 30 s")
	}
	ambientOn = false
	spaceFxCount = 0
	g.drawBackAnim(scr)
	if spaceFxCount != 0 {
		t.Error("ambientOn=false still draws the space sky")
	}
}

// ---- venue acts (venueacts.go) -----------------------------------------------------------------------------------

func vaGlobals(t *testing.T) {
	t.Helper()
	pr, ps, pv, pf := vaRng, vaSeed, va, fastEv
	t.Cleanup(func() { vaRng, vaSeed, va, fastEv = pr, ps, pv, pf })
}

// vaLoad: a party of the campaign level with that icon, the acts fresh and seeded.
func vaLoad(t *testing.T, icon string, seed int64) *Game {
	t.Helper()
	vaGlobals(t)
	vaSeed, vaRng, va = seed, nil, nil
	g := peLoad(t, campaignIdx(icon), false)
	g.policeCalled, g.cops = false, nil
	return g
}

func vaStepG(g *Game, dt float64) {
	g.time += dt
	g.updateVenueActs(dt)
	for _, gu := range g.guests {
		g.updateGuest(gu, dt)
	}
}

// 2. the pirate crowd: nearly everybody a pirate, with peg legs, hooks, patches, parrots and the cartoon looks
func TestPirateCrewLooks(t *testing.T) {
	if crowdChance["pirate"] < 0.85 {
		t.Fatalf("only %.0f%% of the guests come as pirates", crowdChance["pirate"]*100)
	}
	r := rand.New(rand.NewSource(5))
	n := map[string]int{}
	for i := 0; i < 600; i++ {
		s := crowdSpec("pirate", r)
		if s.pir&pirBitPeg != 0 {
			n["peg"]++
		}
		if s.pir&pirBitHook != 0 {
			n["hook"]++
		}
		if s.pir&pirBitParrot != 0 {
			n["parrot"]++
		}
		if s.Glasses == GlassPatch {
			n["patch"]++
		}
		if s.Hat == HatCowboy && s.HatCol == hx(0xe8c870) {
			n["strawhat"]++
		}
		if s.Hair == hx(0x3a9a4a) && s.Carry == HoldSaber {
			n["swordsman"]++
		}
		if s.Hair2 == HairPony && s.Hair == hx(0xe8c060) {
			n["ponytail"]++
		}
		if s.Skin == hx(0x9ad0a8) {
			n["ghost"]++
		}
	}
	for _, k := range []string{"peg", "hook", "parrot", "patch", "strawhat", "swordsman", "ponytail", "ghost"} {
		if n[k] < 20 {
			t.Errorf("%s: only %d of 600 pirates", k, n[k])
		}
	}
	// the bits are really drawn: every view of a pirate with all of them differs from the same pirate without them
	base := pirateCrewSpec(genericSpec(rand.New(rand.NewSource(9))), rand.New(rand.NewSource(10)))
	base.pir, base.Glasses, base.Carry = 0, GlassNone, HoldNone
	full := base
	full.pir, full.Glasses = pirBitPeg|pirBitHook|pirBitParrot, GlassPatch
	for _, vw := range []int{vFront, vSide} {
		p := Pose{}
		a, b := genChar(&base, vw, p), genChar(&full, vw, p)
		diff := 0
		for i := range a.p {
			if a.p[i] != b.p[i] {
				diff++
			}
		}
		if diff < 20 {
			t.Errorf("view %d: the pirate bits change only %d bytes", vw, diff)
		}
	}
	gs := pirateGuardSpec(rand.New(rand.NewSource(3)))
	if gs.Hat != HatPirate || gs.Glasses != GlassPatch {
		t.Errorf("the ship's guard is not a pirate: hat %d glasses %d", gs.Hat, gs.Glasses)
	}
}

// 2. dynamite barrels (a trap) and loot on the deck; the layout stays whole
func TestPirateDynamiteAndLoot(t *testing.T) {
	e := campaignLevels[pirateIdx]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2} {
		d := genNamed("pirate", seed, "")
		n := propCount(d)
		if n["pdynamite"] < 1 || n["pgold"] < 2 {
			t.Errorf("seed %d: dynamite x%d, loot x%d", seed, n["pdynamite"], n["pgold"])
		}
		venueWhole(t, "pirate", d)
	}
	if propKinds["pdynamite"].Trap != "explode" {
		t.Error("the dynamite barrel is not a trap")
	}
	// the bay moves: crests every frame, a dolphin's spots, a shark's fin some time in 25 s; never over the ship
	g := ambientSetup(t)
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	g.loadLevel(campaignIdx("pirate"))
	t.Cleanup(func() { pirSeaCur = nil })
	b := pirSeaFor(g.lvl.Def)
	if len(b.crests) < 80 || len(b.spots) < 3 {
		t.Errorf("%d wave crests, %d dolphin spots", len(b.crests), len(b.spots))
	}
	for _, c := range b.crests {
		if g.lvl.Def.Shape(c.x, c.y, 10) {
			t.Fatalf("a wave crest over the ship at %v,%v", c.x, c.y)
		}
	}
	fin := false
	for tm := 0.0; tm < 25; tm += 0.25 {
		g.time = tm
		pirSeaCount = 0
		g.drawBackAnim(scr)
		if pirSeaCount < len(b.crests) {
			t.Fatalf("t=%.2f: %d drawn", tm, pirSeaCount)
		}
		if pirSeaCount > len(b.crests) {
			fin = true
		}
	}
	if !fin {
		t.Error("no shark's fin in 25 s")
	}
}

// 2. the insult duels: the classic pairs, both languages, short; never more than vaInsultCap at once; the answer is
// the comeback of the insult just said, by the other duellist
func TestPirateInsultDuels(t *testing.T) {
	if len(insultPairs) < 6 {
		t.Fatalf("%d insult pairs", len(insultPairs))
	}
	for _, p := range insultPairs {
		for _, s := range []string{p.insult, p.reply, p.insultEN, p.replyEN} {
			if n := len([]rune(s)); n > 48 {
				t.Errorf("%q: %d characters (max 48)", s, n)
			}
		}
		if langDataEN[p.insult] != p.insultEN || langDataEN[p.reply] != p.replyEN {
			t.Errorf("%q / %q: no English", p.insult, p.reply)
		}
	}
	g := vaLoad(t, "pirate", 7)
	fastEv = true
	replyOf := map[string]string{}
	for _, p := range insultPairs {
		replyOf[p.insult] = p.reply
	}
	maxIns, said, answered := 0, 0, 0
	last := map[*vaAct]*Guest{}
	lastLine := map[*vaAct]string{}
	for f := 0; f < 30*90; f++ {
		vaStepG(g, 1.0/30)
		run := g.vaActive()
		if run == nil {
			t.Fatal("no acts on the pirate ship")
		}
		if c := run.count(vaInsult); c > maxIns {
			maxIns = c
		}
		for _, a := range run.acts {
			if a.kind != vaInsult || !a.met {
				continue
			}
			for _, gu := range []*Guest{a.a.gu, a.b.gu} {
				if gu.icon == "" || gu.icon == lastLine[a] || gu.iconT < 2.6 {
					continue
				}
				if _, ok := replyOf[gu.icon]; ok {
					said++
				} else if prev := lastLine[a]; replyOf[prev] == gu.icon {
					if last[a] == gu {
						t.Errorf("%q answered by the one who insulted", gu.icon)
					}
					answered++
				} else {
					t.Errorf("%q does not answer %q", gu.icon, prev)
				}
				last[a], lastLine[a] = gu, gu.icon
			}
		}
	}
	t.Logf("90 s on the ship: %d insults, %d comebacks, at most %d duels at once", said, answered, maxIns)
	if maxIns > vaInsultCap {
		t.Errorf("%d insult duels at once (max %d)", maxIns, vaInsultCap)
	}
	if said < 2 || answered < 2 {
		t.Errorf("in 90 s: %d insults, %d comebacks", said, answered)
	}
	// the ship's security are pirates
	pir := 0
	for _, gd := range g.guards {
		if gd.vip == 0 && !gd.special && gd.cs != nil && gd.cs.Spec != nil && gd.cs.Spec.Hat == HatPirate {
			pir++
		}
	}
	if pir == 0 {
		t.Error("no pirate guard on the ship")
	}
}

// 3. the 70s disco: three small lit floors, each with its own drawing that moves, nothing standing on them, the
// layout whole; the huge afro is drawn bigger than the old one
func TestDiscoMiniFloors(t *testing.T) {
	e := campaignLevels[campaignIdx("disco70")]
	for _, seed := range []int64{e.seed, e.seed + 1013, 1, 2} {
		d := genNamed("disco70", seed, "")
		n := 0
		for _, f := range d.Floors {
			if f.Kind == "disco70" && f.R != discoFloor {
				n++
				if discoMiniOf(f.R) < 0 {
					t.Errorf("seed %d: unknown small floor %v", seed, f.R)
				}
			}
		}
		if n < 2 {
			t.Errorf("seed %d: %d small dance floors", seed, n)
		}
		if d.Dance[0] != discoFloor {
			t.Errorf("seed %d: dance areas %v", seed, d.Dance)
		}
		l := venueWhole(t, "disco70", d)
		for _, p := range l.Props {
			foot := Rect{p.X - p.K.FW/2, p.Y - p.K.FH, p.K.FW, p.K.FH}
			for _, m := range discoMinis {
				if rectsOverlap(foot, m.R) {
					t.Errorf("seed %d: %s at %.0f,%.0f stands on the small floor %v", seed, p.Name, p.X, p.Y, m.R)
				}
			}
		}
	}
	// the drawings differ from each other and move with the beat
	pic := func(i int, b float64) string {
		m := discoMinis[i]
		w, h := int(m.R.W)/discoCell, int(m.R.H)/discoCell
		out := make([]byte, 0, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out = append(out, discoMiniValue(m.kind, x, y, w, h, b))
			}
		}
		return string(out)
	}
	seen := map[int]bool{}
	for i, m := range discoMinis {
		seen[m.kind] = true
		lit := 0
		for _, c := range pic(i, 0.5) {
			if c != 0 {
				lit++
			}
		}
		if lit == 0 || lit == len(pic(i, 0.5)) {
			t.Errorf("floor %d: %d lit tiles", i, lit)
		}
		if pic(i, 0.5) == pic(i, 1.7) {
			t.Errorf("floor %d: the drawing does not move with the beat", i)
		}
	}
	if len(seen) != len(discoMinis) {
		t.Error("two small floors share the same drawing")
	}
	// the huge afro is bigger than the old one
	hair := func(h int) int {
		s := CharSpec{Skin: hx(0xe8d0b8), Hair: hx(0x2a1810), Shirt: hx(0xd05a30), Shirt2: hx(0xd05a30), Pants: hx(0x5a2a6a),
			Shoes: hx(0xf4f0e0), Top: TopTee, Hair2: h, noOverride: true}
		b := genChar(&s, vFront, Pose{})
		n := 0
		for i := 0; i < len(b.p); i += 4 {
			if b.p[i] == 0x2a && b.p[i+1] == 0x18 && b.p[i+2] == 0x10 {
				n++
			}
		}
		return n
	}
	if xl, old := hair(HairAfroXL), hair(HairAfro); xl < old*3/2 {
		t.Errorf("the huge afro has %d hair pixels, the old one %d", xl, old)
	}
}

// vaWatch runs the party for secs seconds (fast acts) and reports which acts really played (both guests in place).
func vaWatch(t *testing.T, g *Game, secs float64, each func()) map[int]int {
	t.Helper()
	fastEv = true
	played := map[int]int{}
	seen := map[*vaAct]bool{}
	for f := 0; f < int(secs*30); f++ {
		vaStepG(g, 1.0/30)
		for _, a := range g.vaActive().acts {
			if a.met && !seen[a] {
				seen[a] = true
				played[a.kind]++
			}
		}
		if each != nil {
			each()
		}
	}
	return played
}

// 4. the ski resort: skiers and snowboarders glide around, snowball fights, snowmen that stay, and the giant snowball
// once at most: it rolls from the top to the bottom and kills the guests in its way
func TestSkiActsAndGiantSnowball(t *testing.T) {
	g := vaLoad(t, "snow", 3)
	alive0 := 0
	for _, gu := range g.guests {
		if !gu.out() {
			alive0++
		}
	}
	balls, maxBalls, was := 0, 0, false
	startY := 0.0
	played := vaWatch(t, g, 120, func() {
		b := &g.vaActive().ball
		if b.on && !was {
			startY = b.y
			balls++
		}
		if b.on {
			maxBalls = 1
		}
		was = b.on
	})
	run := g.vaActive()
	if len(run.skiers) < 3 {
		t.Errorf("%d skiers", len(run.skiers))
	}
	boards := 0
	for _, k := range run.skiers {
		boards += k
	}
	if boards == 0 || boards == len(run.skiers) {
		t.Errorf("%d boards among %d skiers: want both skis and boards", boards, len(run.skiers))
	}
	for _, k := range []int{vaSki, vaSnowball, vaSnowman} {
		if played[k] == 0 {
			t.Errorf("act %d never played in 120 s (played %v)", k, played)
		}
	}
	built := 0
	for _, sm := range run.snowmen {
		if sm.k >= 1 {
			built++
		}
	}
	if built == 0 {
		t.Error("no snowman finished")
	}
	if balls != 1 || !run.ball.done || maxBalls != 1 {
		t.Fatalf("giant snowball: started %d times, done %v", balls, run.ball.done)
	}
	if startY > 0 {
		t.Errorf("the giant snowball starts at y=%.0f, not at the top", startY)
	}
	if run.ball.killed == 0 {
		t.Log("the giant snowball missed everybody this time")
	}
	// it never comes back in the same party
	for f := 0; f < 30*200; f++ {
		vaStepG(g, 1.0/30)
		if g.vaActive().ball.on {
			t.Fatal("a second giant snowball in the same party")
		}
	}
	// a ball that rolls through a crowd kills them: put three guests in its way
	g2 := vaLoad(t, "snow", 4)
	run2 := g2.vaActive()
	run2.started = true
	run2.ball = vaBall{at: g2.time + 1, on: true, x: 480, y: -vaBallRadius, r: vaBallRadius}
	var victims []*Guest
	for _, gu := range g2.guests {
		if !gu.out() && gu.perf == "" && len(victims) < 3 {
			gu.pos = Vec{480, 200 + float64(len(victims))*60}
			victims = append(victims, gu)
		}
	}
	for f := 0; f < 30*12 && !run2.ball.done; f++ {
		g2.time += 1.0 / 30
		g2.vaBallStep(run2, 1.0/30)
	}
	for _, v := range victims {
		if !v.dead {
			t.Errorf("a guest at %v survived the giant snowball", v.pos)
		}
	}
	if !run2.ball.done {
		t.Error("the giant snowball never left the map")
	}
}

// 5. the birthday party: a troupe of clowns juggling, making balloons, squirting water; a clown car drives about
func TestBirthdayClowns(t *testing.T) {
	g := vaLoad(t, "cake", 5)
	played := vaWatch(t, g, 120, nil)
	run := g.vaActive()
	clowns := 0
	for _, gu := range run.roles {
		if sp := gu.cs.Spec; sp != nil && sp.Hair2 == HairAfro && sp.Skin == hx(0xf8f4f0) {
			clowns++
		}
	}
	if clowns < 5 {
		t.Errorf("%d clowns", clowns)
	}
	for _, k := range []int{vaJuggle, vaBalloon, vaSquirt} {
		if played[k] == 0 {
			t.Errorf("act %d never played in 120 s (played %v)", k, played)
		}
	}
	if !run.car.on {
		t.Fatal("no clown car")
	}
	for x := -run.car.span; x <= run.car.span; x += 4 {
		if !g.lvl.Free(Vec{run.car.cx + x, run.car.cy}) {
			t.Fatalf("the clown car drives through a solid at %v", run.car.cx+x)
		}
	}
	if p1, _ := run.carAt(1); p1 == func() Vec { p, _ := run.carAt(4); return p }() {
		t.Error("the clown car does not move")
	}
}

// 6. the hockey rink: players with sticks pass the puck to each other and to guests, and slapshots hit guests; some die
// (never more than vaPuckKills in one party)
func TestHockeyPlayersPassAndHit(t *testing.T) {
	g := vaLoad(t, "hockey", 6)
	hits := 0
	played := vaWatch(t, g, 240, func() {
		for _, a := range g.vaActive().acts {
			if a.kind == vaShot && a.step == 2 {
				hits++
			}
		}
	})
	run := g.vaActive()
	sticks := 0
	for _, gu := range run.roles {
		if gu.cs.Spec != nil && gu.cs.Spec.Carry == HoldStick && gu.cs.Spec.Hat == HatHelmet {
			sticks++
		}
	}
	if sticks < 5 {
		t.Errorf("%d hockey players", sticks)
	}
	if played[vaPass] == 0 || played[vaShot] == 0 {
		t.Errorf("passes %d, slapshots %d in 240 s", played[vaPass], played[vaShot])
	}
	if hits == 0 && run.puckKills == 0 {
		t.Error("no slapshot ever hit a guest")
	}
	if run.puckKills > vaPuckKills {
		t.Errorf("%d deaths by puck (max %d)", run.puckKills, vaPuckKills)
	}
	// a slapshot that kills: forced
	g2 := vaLoad(t, "hockey", 8)
	run2 := g2.vaActive()
	run2.started = true
	var shooter, target *Guest
	for _, gu := range g2.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		if shooter == nil {
			shooter = gu
		} else if target == nil {
			target = gu
		}
	}
	vaRng = rand.New(rand.NewSource(1))
	killed := false
	for try := 0; try < 60 && !killed; try++ {
		a := &vaAct{kind: vaShot, met: true, b2: true, until: g2.time + 4}
		a.a.gu, a.b.gu = shooter, target
		a.a.hold(gFun)
		a.b.hold(gFun)
		for f := 0; f < 60; f++ {
			g2.time += 1.0 / 30
			if !g2.vaShotStep(run2, a, vaRng) {
				break
			}
		}
		killed = target.dead
	}
	if !killed {
		t.Error("a slapshot never killed in 60 tries")
	}
}

// 7. the train station: its own fast music (no more rap), and trains one after another with some pauses
func TestTrainStationBusyAndLively(t *testing.T) {
	gi := genreForTitle("ESTACIÓN DE TREN")
	if gi.id == "rap" || gi.bpm < 150 {
		t.Errorf("the train station plays %s at %v BPM", gi.id, gi.bpm)
	}
	if _, ok := genreSongs[gi.id]; !ok {
		t.Fatalf("no song for %s", gi.id)
	}
	if langDataEN[gi.label] == "" {
		t.Errorf("%q has no English", gi.label)
	}
	for i, e := range campaignLevels {
		if e.title != "ESTACIÓN DE TREN" && genreForTitle(e.title).id == gi.id && e.level == nil {
			t.Errorf("party %d also plays %s", i+1, gi.id)
		}
	}
	pr, ps, pv := trainRng, trainSeed, trainRun
	t.Cleanup(func() { trainRng, trainSeed, trainRun = pr, ps, pv })
	trainRng, trainSeed = nil, 21
	g := peLoad(t, campaignIdx("train"), false)
	g.policeCalled, g.cops = false, nil
	for _, gu := range g.guests { // nobody on the tracks: this test counts trains, not victims
		gu.state = gIdle
	}
	trains, longest, quiet := 0, 0.0, 0.0
	had := false
	for f := 0; f < 30*300; f++ {
		g.time += 1.0 / 30
		g.updateTrains(1.0 / 30)
		if g.train != nil && !had {
			trains++
			quiet = 0
		}
		if g.train == nil {
			quiet += 1.0 / 30
			longest = math.Max(longest, quiet)
		}
		had = g.train != nil
		if g.mode != ModePlay {
			g.mode = ModePlay
		}
	}
	if trains < 14 { // before v1.4.0: one every 45-90 s, about 5 in 5 minutes
		t.Errorf("only %d trains in 5 minutes", trains)
	}
	if longest < 20 {
		t.Errorf("no pause between the runs of trains (longest gap %.1f s)", longest)
	}
	t.Logf("5 minutes: %d trains, longest pause %.1f s", trains, longest)
}

// 3. guests really go dancing on the small floors of the 70s disco
func TestDiscoMiniFloorsGetDancers(t *testing.T) {
	g := vaLoad(t, "disco70", 9)
	on := 0
	played := vaWatch(t, g, 60, func() {
		for _, a := range g.vaActive().acts {
			if a.kind == vaMiniDance && a.met && discoMiniOf(func() Rect {
				for _, m := range discoMinis {
					if m.R.Grow(4).Has(a.a.gu.pos.X, a.a.gu.pos.Y) {
						return m.R
					}
				}
				return Rect{}
			}()) >= 0 {
				on++
			}
		}
	})
	if played[vaMiniDance] < 3 || on == 0 {
		t.Errorf("%d dancers went to the small floors (%d dancer-frames on them)", played[vaMiniDance], on)
	}
}
