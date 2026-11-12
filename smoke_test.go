package main

import (
	"image/color"
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

const smokeDT = 1.0 / 60

// smokeOpenGame: one player on a bare 960x540 field (no walls unless given), facing right.
func smokeOpenGame(t *testing.T, two bool, walls ...Rect) *Game {
	t.Helper()
	old := progressFrozen
	t.Cleanup(func() { progressFrozen = old })
	g := twoGame(two)
	def := &LevelDef{W: 960, H: 540, Start: Vec{480, 300}}
	for _, r := range walls {
		def.Walls = append(def.Walls, Wall{R: r})
	}
	g.lvl = buildLevel(def)
	g.visK = 1
	g.p.pos, g.p.dir = def.Start, DirRight
	if g.p2 != nil {
		g.p2.pos, g.p2.dir = Vec{480, 450}, DirLeft
	}
	g.smk = smokeState{}
	return g
}

// runNade steps the grenades until they burst (or 3 s), checking that none is ever inside a solid.
func runNade(t *testing.T, g *Game) {
	t.Helper()
	for i := 0; i < 180 && len(g.smk.nades) > 0; i++ {
		g.updateSmoke(smokeDT)
		for _, n := range g.smk.nades {
			if g.lvl.SolidP(n.pos.X, n.pos.Y) {
				t.Fatalf("the grenade went into a solid at %v", n.pos)
			}
		}
	}
	if len(g.smk.nades) > 0 {
		t.Fatal("the grenade never burst")
	}
}

// The smoke is active EXACTLY 6 s (one frame of tolerance), then it is gone.
func TestSmokeLastsExactlySixSeconds(t *testing.T) {
	g := smokeOpenGame(t, false)
	g.burstSmoke(Vec{480, 300}, 0)
	frames := 0
	for g.smokeActive() && frames < 1000 {
		g.updateSmoke(smokeDT)
		frames++
	}
	if d := float64(frames) * smokeDT; math.Abs(d-smokeDur) > smokeDT+1e-9 || smokeDur != 6 {
		t.Fatalf("the smoke lasted %.3f s, want 6.0 s ±1 frame", d)
	}
	// the fades are inside the 6 s: thick in the middle, thin at both ends
	if smokeAlpha(0) != 0 || smokeAlpha(3) != 1 || smokeAlpha(smokeDur-0.01) > 0.05 || smokeAlpha(smokeFadeIn) != 1 {
		t.Errorf("fades: %v %v %v", smokeAlpha(0), smokeAlpha(3), smokeAlpha(smokeDur-0.01))
	}
}

// Picking a grenade up and throwing it with the use key: an arc in the direction he faces, one bounce, the
// cloud 120-160 px away. Throwing it is no crime.
func TestSmokeGrenadePickUpAndThrow(t *testing.T) {
	g := smokeOpenGame(t, false)
	g.placeSmoke(1, rand.New(rand.NewSource(1)), true)
	var pr *Prop
	for _, x := range g.lvl.Props {
		if x.Name == smokeKind {
			pr = x
		}
	}
	if pr == nil {
		t.Fatal("no grenade placed next to the start")
	}
	g.p.pos = Vec{pr.X, pr.Y + 4}
	if lab, _ := g.useLabel(pr); lab != "Recoger granada de humo" {
		t.Fatalf("label %q", lab)
	}
	g.interact()
	if g.p.item != "smoke" || !pr.gone {
		t.Fatalf("not picked up: item %q", g.p.item)
	}
	if lab, _ := g.useLabel(&Prop{K: propKinds[smokeKind]}); lab != "" {
		t.Error("one grenade in the hand: a second one must not offer itself")
	}
	g.p.pos, g.p.dir = Vec{480, 300}, DirRight
	g.interact()
	if g.p.item != "" || len(g.smk.nades) != 1 {
		t.Fatalf("not thrown: item %q, %d in the air", g.p.item, len(g.smk.nades))
	}
	maxH := 0.0
	bounces := 0
	wasDown := false
	for i := 0; i < 180 && len(g.smk.nades) > 0; i++ {
		g.updateSmoke(smokeDT)
		if len(g.smk.nades) == 1 {
			n := g.smk.nades[0]
			maxH = math.Max(maxH, n.h)
			if n.h <= 0 && !wasDown && i > 0 {
				bounces++
			}
			wasDown = n.h <= 0
		}
	}
	if len(g.smk.clouds) != 1 {
		t.Fatal("no cloud")
	}
	c := g.smk.clouds[0].pos
	if d := c.X - 480; d < 120 || d > 160 || math.Abs(c.Y-300) > 1 {
		t.Errorf("the cloud is at %v: %0.f px, want 120-160 px to the right", c, d)
	}
	if maxH < 8 {
		t.Errorf("no arc: max height %.1f", maxH)
	}
	if g.wanted != 0 || g.stars != 0 || g.policeCalled {
		t.Error("a smoke grenade is not a crime")
	}
}

// The grenade never goes through a wall: it bounces off and the cloud stays on the thrower's side.
func TestSmokeGrenadeStopsAtWalls(t *testing.T) {
	g := smokeOpenGame(t, false, Rect{540, 200, 20, 200})
	g.p.item = "smoke"
	if !g.throwSmoke(g.p) {
		t.Fatal("not thrown")
	}
	runNade(t, g)
	if c := g.smk.clouds[0].pos; c.X >= 540 {
		t.Fatalf("the grenade went through the wall: %v", c)
	}
}

// A kill inside the smoke, or seen only through it, has no witness: no alert, no call, no cuffs, no stars.
func TestSmokeWitnessKillInsideOrThroughSeesNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cloud Vec
		gu    Vec
	}{
		{"inside", Vec{200, 300}, Vec{280, 300}},
		{"through", Vec{250, 370}, Vec{300, 300}},
	} {
		g := suspectGame()
		g.policeCalled, g.wanted = false, 0
		gu := suspectGuest(g, tc.gu, DirLeft)
		if tc.name == "inside" {
			c := suspectCop(g, Vec{200, 220}, DirDown)
			c.onSceneT = 30
		}
		gd := &Guard{pos: Vec{200, 200}, dir: DirDown, state: gdWait}
		if tc.name == "inside" {
			g.guards = []*Guard{gd}
		}
		if !g.sees(gu.pos, gu.dir, g.p.pos, guestRange, guestDot) || (len(g.cops) > 0) != g.copSeesAt(g.p.pos) {
			t.Fatalf("%s: setup: without smoke the guest and the officer see the player (%v %v)", tc.name, g.sees(gu.pos, gu.dir, g.p.pos, guestRange, guestDot), g.copSeesAt(g.p.pos))
		}
		g.smk.clouds = []smokeCloud{{pos: tc.cloud, t: 1}}
		if tc.name == "through" && (dist(gu.pos, tc.cloud) <= smokeR || dist(g.p.pos, tc.cloud) <= smokeR) {
			t.Fatal("setup: both must be outside the cloud")
		}
		g.witness()
		if gu.sawKiller || g.callerGuest != nil || gu.state != gIdle {
			t.Errorf("%s: the guest saw the kill through the smoke (state %v)", tc.name, gu.state)
		}
		if g.playerIsSuspect() || g.playerAlert(g.p) || g.wanted != 0 || g.policeCalled || gd.sight > 0 {
			t.Errorf("%s: cuffs/alert/search after a kill in the smoke (wanted %.0f)", tc.name, g.wanted)
		}
	}
}

// Inside the cloud the guests, guards and officers are dazed: they stand there for the smoke + 1 s and then
// go on as before. Nobody flees. The player is not affected.
func TestSmokeDazesAndRecovers(t *testing.T) {
	g := suspectGame()
	gu := suspectGuest(g, Vec{230, 300}, DirLeft)
	out := suspectGuest(g, Vec{500, 300}, DirLeft) // far from it: not touched
	gd := &Guard{pos: Vec{200, 340}, dir: DirDown, state: gdWait, route: []Vec{{200, 340}, {300, 340}}, home: Vec{200, 340}, leash: 170}
	g.guards = []*Guard{gd}
	c := suspectCop(g, Vec{170, 300}, DirRight)
	c.onSceneT = 5 // he has been on scene only 5 s
	g.burstSmoke(Vec{200, 300}, 0)
	start := gu.pos
	for i := 0; i < int(smokeDur/smokeDT)-1; i++ {
		g.updateSmoke(smokeDT)
		g.updateGuest(gu, smokeDT)
		g.updateGuard(gd, smokeDT)
		g.updateCop(c, smokeDT)
		if gu.pos != start || gu.state == gFlee || gu.state == gRun {
			t.Fatalf("frame %d: the dazed guest moved or ran (%v %v)", i, gu.pos, gu.state)
		}
		if c.state == cLeave {
			t.Fatal("a dazed officer left the scene before his 20 s")
		}
	}
	if gu.smogT <= 0 || gd.smogT <= 0 || c.smogT <= 0 {
		t.Fatalf("not dazed: guest %.2f guard %.2f cop %.2f", gu.smogT, gd.smogT, c.smogT)
	}
	if out.smogT > 0 {
		t.Error("a guest outside the cloud was dazed")
	}
	if c.onSceneT < 10 {
		t.Errorf("the officer's time on scene must keep counting: %.1f", c.onSceneT)
	}
	if g.p.stunT > 0 || g.p.knockT > 0 {
		t.Error("the player is never dazed by the smoke")
	}
	for i := 0; i < int((smokeDazeExtra+0.2)/smokeDT)+2; i++ {
		g.updateSmoke(smokeDT)
		g.updateGuest(gu, smokeDT)
		g.updateGuard(gd, smokeDT)
		g.updateCop(c, smokeDT)
	}
	if g.smokeActive() || gu.smogT > 0 || gd.smogT > 0 || c.smogT > 0 {
		t.Fatalf("not recovered: guest %.2f guard %.2f cop %.2f", gu.smogT, gd.smogT, c.smogT)
	}
	if !g.sees(gu.pos, DirLeft, Vec{180, 300}, guestRange, guestDot) {
		t.Error("after the smoke the guests see again")
	}
	// v1.3.0: the 20 s on scene only hold while somebody is suspected (after his 10-15 s search an officer with no
	// suspect goes back: game.go copNext / ambulance.go copMayLeave)
	if c.state == cLeave && c.onSceneT < copMinStay && g.caseHasSuspect() {
		t.Error("the officer left before his 20 s with a suspect")
	}
}

// Zombies are not affected by the smoke.
func TestSmokeZombiesUnaffected(t *testing.T) {
	g := suspectGame()
	z := suspectGuest(g, Vec{210, 300}, DirLeft)
	z.state = gZombie
	g.burstSmoke(Vec{200, 300}, 0)
	g.updateSmoke(smokeDT)
	if z.smogT > 0 {
		t.Error("a zombie was dazed")
	}
}

// Two players: each one throws his own grenade with his own key, and each cloud is his.
func TestSmokeTwoPlayersEachThrowsHisOwn(t *testing.T) {
	g := smokeOpenGame(t, true)
	g.p.item, g.p2.item = "smoke", "smoke"
	g.p.pos, g.p.dir = Vec{300, 150}, DirRight
	g.p2.pos, g.p2.dir = Vec{600, 420}, DirLeft
	g.asPlayer(0, func() { g.interact() })
	g.asPlayer(1, func() { g.interact() })
	if len(g.smk.nades) != 2 || g.first().item != "" || g.p2.item != "" {
		t.Fatalf("%d grenades in the air, items %q %q", len(g.smk.nades), g.first().item, g.p2.item)
	}
	runNade(t, g)
	if len(g.smk.clouds) != 2 {
		t.Fatal("two clouds expected")
	}
	for _, c := range g.smk.clouds {
		if c.owner == 0 && c.pos.X <= 300 || c.owner == 1 && c.pos.X >= 600 {
			t.Errorf("cloud of player %d went the wrong way: %v", c.owner+1, c.pos)
		}
	}
	// one grenade in the hand each: with both slots full nothing more is taken
	g.p2.item, g.p2.spare = "poison", "disguise"
	if canTake(g.p2, "smoke") {
		t.Error("both slots full: no room for a grenade")
	}
	// the limit of clouds at once
	g.p.item = "smoke"
	g.smk.clouds = append(g.smk.clouds, smokeCloud{pos: Vec{100, 100}})
	if g.throwSmoke(g.p) || g.p.item != "smoke" {
		t.Error("over the limit of clouds the grenade stays in the hand")
	}
}

// "Sometimes": about one party in three gets 1-2 grenades, decided by the seed and the level (deterministic).
func TestSmokeSometimesBySeed(t *testing.T) {
	n, with := 0, 0
	for seed := int64(1); seed <= 1500; seed++ {
		for lv := 0; lv < 4; lv++ {
			k := smokeRoll(seed*7919, lv)
			if k != smokeRoll(seed*7919, lv) {
				t.Fatal("not deterministic")
			}
			if k < 0 || k > 2 {
				t.Fatalf("%d grenades", k)
			}
			n++
			if k > 0 {
				with++
			}
		}
	}
	if f := float64(with) / float64(n); f < 0.25 || f > 0.40 {
		t.Fatalf("%.1f %% of the parties have grenades, want 25-40 %%", 100*f)
	}
}

// Every level: the grenades lie on free, reachable, dry floor (not on a solid), with their prompt.
func TestSmokeGrenadePlacementEveryLevel(t *testing.T) {
	defs := testLevels(t)
	for i, def := range defs {
		g := hideGameOn(t, def, false)
		g.placeSmoke(2, rand.New(rand.NewSource(int64(i))), false)
		reach := g.lvl.reachFrom(def.Start)
		got := 0
		for _, pr := range g.lvl.Props {
			if pr.Name != smokeKind {
				continue
			}
			got++
			p := Vec{pr.X, pr.Y}
			if !g.lvl.Free(p) || g.lvl.SolidP(p.X, p.Y) || g.lvl.InPool(p) || !reach[int(p.Y/navCell)*g.lvl.cw+int(p.X/navCell)] {
				t.Errorf("L%d %s: grenade on a bad spot %v", i+1, def.Title, p)
			}
			g.p.pos = Vec{p.X, p.Y + 6}
			if lab, _ := g.useLabel(pr); lab != "Recoger granada de humo" {
				t.Errorf("L%d: label %q", i+1, lab)
			}
		}
		if got != 2 {
			t.Errorf("L%d %s: %d grenades placed, want 2", i+1, def.Title, got)
		}
	}
}

// The security cameras see nothing through the smoke, and a cloud elsewhere changes nothing; with the blackout
// the cameras stay off and the smoke still blocks the sight of everybody.
func TestSmokeCameraAndBlackout(t *testing.T) {
	g, c := camGame(t)
	if !c.sees(g, g.p.pos) {
		t.Fatal("setup")
	}
	g.smk.clouds = []smokeCloud{{pos: Vec{700, 450}}}
	if !c.sees(g, g.p.pos) {
		t.Error("a cloud far away must not blind the camera")
	}
	g.smk.clouds = []smokeCloud{{pos: g.p.pos}}
	if c.sees(g, g.p.pos) {
		t.Error("the camera sees through the smoke")
	}
	g.witness()
	if g.policeCalled || g.playerIsSuspect() {
		t.Error("a camera filmed a kill inside the smoke")
	}
	g.blk = blackoutState{planned: true, started: true, t: 5, dur: 10}
	if c.sees(g, Vec{200, 150}) {
		t.Error("blackout: no cameras")
	}
	if g.sees(Vec{200, 260}, DirUp, g.p.pos, guestRange, guestDot) {
		t.Error("blackout + smoke: still nobody sees into the smoke")
	}
	smokePx(t)
	scr := ebiten.NewImage(screenW, screenH)
	g.drawSmoke(scr) // with the darkness on top: no panic
	g.drawBlackout(scr)
}

// smokePx: the 1x1 white pixel fillRect draws with (newGame makes it).
func smokePx(t *testing.T) {
	old := px
	if px == nil {
		px = ebiten.NewImage(1, 1)
		px.Fill(color.White)
	}
	t.Cleanup(func() { px = old })
}

// No allocation per frame: the update, the sight test and the drawing of the clouds.
func TestSmokeNoAllocPerFrame(t *testing.T) {
	g := suspectGame()
	for i := 0; i < 30; i++ {
		suspectGuest(g, Vec{150 + float64(i%6)*20, 260 + float64(i/6)*20}, DirLeft)
	}
	g.smk.clouds = make([]smokeCloud, 0, 4)
	g.smk.clouds = append(g.smk.clouds, smokeCloud{pos: Vec{200, 300}}, smokeCloud{pos: Vec{600, 300}})
	g.smk.nades = make([]smokeNade, 0, 4)
	g.updateSmoke(smokeDT) // first frame: set up
	a := testing.AllocsPerRun(100, func() {
		g.updateSmoke(smokeDT / 10)
		g.sees(Vec{400, 300}, DirLeft, Vec{100, 300}, guestRange, guestDot)
		g.smokeBlocks(Vec{0, 0}, Vec{900, 500})
	})
	if a != 0 {
		t.Errorf("update/sight allocate %.1f per frame", a)
	}
	smokePx(t)
	scr := ebiten.NewImage(screenW, screenH)
	g.drawSmoke(scr) // builds the sprites once
	// our own code allocates nothing; ebiten itself may book a few bytes per DrawImage outside the game loop, so
	// the bound is what the same number of bare DrawImage calls costs
	draws := 0
	for _, gu := range g.guests {
		if gu.smogT > 0 {
			draws += 3
		}
	}
	draws += len(g.smk.clouds) * (1 + len(smokeOrbit))
	var op ebiten.DrawImageOptions
	base := testing.AllocsPerRun(50, func() { scr.DrawImage(smokeStarImg, &op) })
	d := testing.AllocsPerRun(50, func() { g.drawSmoke(scr) })
	if d > base*float64(draws) {
		t.Errorf("drawing the smoke allocates %.1f per frame, %d bare draws cost %.1f", d, draws, base*float64(draws))
	}
	t.Logf("draw: %.0f allocs for %d DrawImage (ebiten: %.1f each)", d, draws, base)
}

// The HUD line and the pick-up prompt, in Spanish and English, 1 and 2 players: inside the screen margins.
func TestSmokeHudAndPromptMarginsLang(t *testing.T) {
	g := auditGame(t)
	oldLang := uiLang
	t.Cleanup(func() { uiLang = oldLang })
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, p := range []*Player{{item: "smoke"}, {item: "poison", doses: 5, spare: "smoke", idx: 1}} {
			name := "GRANADA DE HUMO"
			if p.item == "poison" {
				name = "VENENO x5"
			}
			for _, lab := range []string{"OBJETO: ", "J2 OBJETO: "} {
				s := lab + name + smokeHudSuffix(p)
				w := g.textW(s, 1)
				if hudX < 16 || hudX+w > 330 {
					t.Errorf("%s: HUD line %q is %.0f px wide (x %.0f..%.0f)", lang, tr(s), w, hudX, hudX+w)
				}
				if lang == "en" && (tr(s) == s || containsAny(tr(s), "GRANADA", "OBJETO", "VENENO")) {
					t.Errorf("not translated: %q", tr(s))
				}
			}
		}
		txt := "[E] Recoger granada de humo"
		w := g.textW(txt, 1) + map2P(true)
		for _, x := range []float64{0, 8, screenW / 2, screenW - 4} {
			cx := clampUseTextX(x, w)
			if cx-w/2 < 16 || cx+w/2 > screenW-16 {
				t.Errorf("%s: prompt at %.0f leaves the margin", lang, x)
			}
		}
		for _, s := range []string{"Recoger granada de humo", "Granada de humo: pulsa E para lanzarla", "¡Granada de humo!", "Ya hay demasiado humo: espera un poco"} {
			if lang == "en" && tr(s) == s {
				t.Errorf("no English for %q", s)
			}
			if lang == "es" && len([]rune(s)) > 48 {
				t.Errorf("too long for a message: %q", s)
			}
		}
	}
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}
