package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// openLevel: a bare 960x540 field (no walls) and a camera on its north edge looking down.
func camGame(t *testing.T, walls ...Rect) (*Game, *secCam) {
	t.Helper()
	old := progressFrozen
	t.Cleanup(func() { progressFrozen = old })
	g := suspectGame()
	g.policeCalled = false
	def := &LevelDef{W: 960, H: 540, Start: Vec{480, 500}}
	for _, r := range walls {
		def.Walls = append(def.Walls, Wall{R: r})
	}
	g.lvl = buildLevel(def)
	g.mode = ModePlay
	g.visK = 1
	g.p.pos = Vec{200, 200}
	c := &secCam{mount: Vec{200, 100}, eye: Vec{200, 106}, base: math.Pi / 2}
	c.ang = c.base
	c.still = true
	g.cams = []*secCam{c}
	return g, c
}

func TestCameraSweepOscillatesWithPausesAtTheEnds(t *testing.T) {
	if camOffset(0) != -1 || camOffset(camPause*0.99) != -1 {
		t.Error("it must rest at the first end")
	}
	mid := camOffset(camPause + camSweepT/2)
	if math.Abs(mid) > 1e-9 {
		t.Errorf("middle of the sweep must be the axis, got %v", mid)
	}
	if camOffset(camPause+camSweepT+camPause/2) != 1 {
		t.Error("it must rest at the other end")
	}
	period := 2 * (camSweepT + camPause)
	if math.Abs(camOffset(period)-camOffset(0)) > 1e-9 {
		t.Error("the sweep must repeat")
	}
	lo, hi := 9.0, -9.0
	for tt := 0.0; tt < period; tt += 0.05 {
		o := camOffset(tt)
		lo, hi = math.Min(lo, o), math.Max(hi, o)
	}
	if lo != -1 || hi != 1 {
		t.Errorf("range %v..%v: it must sweep the whole +-1", lo, hi)
	}
	c := &secCam{base: math.Pi / 2}
	c.t = 0
	a0 := c.aim()
	c.t = camPause + camSweepT + 0.01
	a1 := c.aim()
	if d := math.Abs(a1 - a0); math.Abs(d-math.Pi/2) > 0.02 {
		t.Errorf("the sweep must cover 90 degrees, covers %.2f rad", d)
	}
}

func TestCameraKillInsideTheConeCallsThePoliceAndCuffs(t *testing.T) {
	g, c := camGame(t)
	if !c.sees(g, g.p.pos) {
		t.Fatal("the player is right in front of the lens")
	}
	g.witness() // nobody else around: only the camera saw it
	if !g.policeCalled || g.wanted < 45 {
		t.Fatalf("a camera is a direct witness: police called=%v wanted=%.0f", g.policeCalled, g.wanted)
	}
	if !g.playerIsSuspect() || g.lastKiller.kind != suspPlayer {
		t.Error("the cuffs (BUSCADO) must go on at once, with no phone call to wait for")
	}
	if c.alert <= 0 {
		t.Error("the cone must turn red when it catches a kill")
	}
}

func TestCameraKillOutsideTheConeDoesNothing(t *testing.T) {
	g, c := camGame(t)
	g.p.pos = Vec{600, 200} // far to the side of where it looks
	if c.sees(g, g.p.pos) {
		t.Fatal("setup: should be outside")
	}
	g.witness()
	if g.policeCalled || g.wanted != 0 || g.playerIsSuspect() {
		t.Error("a kill the camera cannot see must change nothing")
	}
	g.p.pos = Vec{200, 100 + camRange + 40} // in line but too far
	g.witness()
	if g.policeCalled || g.wanted != 0 {
		t.Error("out of range: nothing")
	}
}

func TestCameraSweepMovesTheConeOffThePlayer(t *testing.T) {
	g, c := camGame(t)
	c.base = math.Pi / 4 // a corner camera that sweeps from along the wall to straight down
	g.p.pos = Vec{200 + 90, 106 + 90}
	seenAt, notAt := false, false
	for tt := 0.0; tt < 2*(camSweepT+camPause); tt += 0.1 {
		c.t = tt
		c.ang = c.aim()
		if c.sees(g, g.p.pos) {
			seenAt = true
		} else {
			notAt = true
		}
	}
	if !seenAt || !notAt {
		t.Errorf("the cone must pass over the player and leave him (seen=%v unseen=%v)", seenAt, notAt)
	}
}

func TestCameraConeBlockedByAWall(t *testing.T) {
	g, c := camGame(t, Rect{120, 150, 160, 14})
	g.p.pos = Vec{200, 210} // behind a wall that crosses the cone
	if c.sees(g, g.p.pos) {
		t.Fatal("a wall between the lens and the player: the camera cannot see him")
	}
	g.witness()
	if g.policeCalled || g.wanted != 0 {
		t.Error("a kill behind a wall is not seen")
	}
	g.updateCameras(0.016)
	if c.rays[camRays/2] > 70 {
		t.Errorf("the drawn cone must be cut by the wall: ray %.0f", c.rays[camRays/2])
	}
}

func TestCameraPassingThroughNormalGameOnlyLightsTheLED(t *testing.T) {
	g, c := camGame(t)
	for i := 0; i < 120; i++ {
		g.time += 1.0 / 60
		g.updateCameras(1.0 / 60)
	}
	if c.led <= 0 {
		t.Error("the LED must turn red while somebody is in the cone")
	}
	if g.policeCalled || g.wanted != 0 || g.stealth || g.playerAlert(g.p) || g.playerIsSuspect() {
		t.Error("in the normal game walking past a camera changes nothing")
	}
	if c.alert > 0 {
		t.Error("the cone stays green in the normal game when nobody kills")
	}
}

func TestCameraHiddenPlayerIsNotSeenByTheLED(t *testing.T) {
	g, c := camGame(t)
	g.p.hidden = true
	g.updateCameras(0.016)
	if c.led > 0 {
		t.Error("a hidden player does not light the LED")
	}
	g.witness()
	if g.policeCalled {
		t.Error("a hidden player is not filmed")
	}
}

func TestCameraStealthCountsPassingThroughTheCone(t *testing.T) {
	g, c := camGame(t)
	g.stealth = true
	runStealthCam := func(secs float64) {
		for tt := 0.0; tt < secs; tt += 1.0 / 60 {
			g.time += 1.0 / 60
			g.updateCameras(1.0 / 60)
			g.updateStealth(1.0 / 60)
		}
	}
	runStealthCam(0.2)
	if g.sth.phase != sthCalm {
		t.Error("a glance is not enough")
	}
	runStealthCam(0.8)
	if g.sth.phase != sthRed || g.sth.detections != 1 {
		t.Fatalf("stealth: standing in the cone must fire the red ALERT (phase %d)", g.sth.phase)
	}
	if !g.policeCalled || c.alert <= 0 {
		t.Error("the alert calls the police and the cone shows red")
	}
}

func TestCameraStealthBoxOrBushHidesFromIt(t *testing.T) {
	g, _ := camGame(t)
	g.stealth = true
	g.sth.box = true
	for i := 0; i < 180; i++ {
		g.time += 1.0 / 60
		g.updateCameras(1.0 / 60)
		g.updateStealth(1.0 / 60)
	}
	if g.sth.detections != 0 {
		t.Error("under the cardboard box the camera sees nothing")
	}
}

func TestCameraKillStealthDetects(t *testing.T) {
	g, _ := camGame(t)
	g.stealth = true
	g.witness()
	if g.sth.detections != 1 || g.sth.phase != sthRed {
		t.Error("a kill filmed in stealth mode must detect him")
	}
}

func TestCameraTwoPlayersOnlyTheFilmedOneIsCuffed(t *testing.T) {
	old := twoPlayerMode
	t.Cleanup(func() { twoPlayerMode = old })
	g := twoGame(true)
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	g.snd = &Sounds{muted: true}
	def := &LevelDef{W: 960, H: 540, Start: Vec{480, 500}}
	g.lvl = buildLevel(def)
	g.visK = 1
	c := &secCam{mount: Vec{200, 100}, eye: Vec{200, 106}, base: math.Pi / 2}
	c.ang = c.base
	g.cams = []*secCam{c}
	p1, p2 := g.p, g.p2
	p1.pos, p2.pos = Vec{700, 200}, Vec{200, 200} // only J2 is under the camera
	g.withPlayer(p2, func() { g.witness() })
	if !p2.jail.sus || p1.jail.sus {
		t.Fatalf("the cuffs go to J2 only (j1=%v j2=%v)", p1.jail.sus, p2.jail.sus)
	}
	if !g.policeCalled {
		t.Error("the police are called")
	}
	// and the other way round: J1 kills away from the lens, nothing new happens to him
	g2 := twoGame(true)
	g2.policeCalled, g2.pendingCar, g2.pendingUnit = false, -1, -1
	g2.snd = &Sounds{muted: true}
	g2.lvl = buildLevel(def)
	g2.visK = 1
	c2 := *c
	g2.cams = []*secCam{&c2}
	g2.p.pos, g2.p2.pos = Vec{700, 200}, Vec{200, 200}
	g2.witness()
	if g2.policeCalled || g2.p.jail.sus {
		t.Error("J1 is out of the cone: no cuffs")
	}
}

// ---- placement -------------------------------------------------------------------------------------

func TestCameraPlacementRules(t *testing.T) {
	g := layerGame(t, false)
	_ = g
	withCams := 0
	for i, def := range levelDefs {
		l := buildLevel(def)
		cams := placeCameras(l, camStealthN)
		if len(cams) > camStealthN {
			t.Errorf("L%d %s: %d cameras, at most %d", i, def.Title, len(cams), camStealthN)
		}
		if len(cams) > 0 {
			withCams++
		}
		for a, c := range cams {
			body := camBodyRect(c.mount)
			if body.X < 0 || body.Y < 0 || body.X+body.W > l.W {
				t.Errorf("L%d %s: camera %d outside the level", i, def.Title, a)
			}
			if !l.Free(Vec{c.mount.X, c.mount.Y + 14}) {
				t.Errorf("L%d %s: the floor under camera %d is not free", i, def.Title, a)
			}
			for _, p := range l.Props {
				if dr := propDrawnRel(p.K, p.Var); body.Hits(Rect{p.X + dr.X, p.Y + dr.Y, dr.W, dr.H}) {
					t.Errorf("L%d %s: camera %d covers the prop %s (%.0f,%.0f)", i, def.Title, a, p.Name, p.X, p.Y)
				}
			}
			for _, pf := range def.Performers {
				if body.Grow(20).Has(pf.Pos.X, pf.Pos.Y) {
					t.Errorf("L%d %s: camera %d next to a performer", i, def.Title, a)
				}
			}
			for _, d := range def.Doors {
				if body.Hits(d.R) {
					t.Errorf("L%d %s: camera %d on a door", i, def.Title, a)
				}
			}
			for b := a + 1; b < len(cams); b++ {
				if dist(c.mount, cams[b].mount) < camSpacing {
					t.Errorf("L%d %s: two cameras too close", i, def.Title)
				}
			}
		}
		// cameras are not solids: the level builds identical collisions with and without them
		if n := len(l.solidsP); n != len(buildLevel(def).solidsP) {
			t.Errorf("L%d: solids differ", i)
		}
	}
	if withCams < 10 {
		t.Errorf("stealth: only %d levels take cameras", withCams)
	}
	t.Logf("stealth: %d of %d levels take cameras", withCams, len(levelDefs))
}

func TestCameraPlanLevelsGetTheirCameras(t *testing.T) {
	g := layerGame(t, false)
	_ = g
	got := map[string]int{}
	for i, def := range levelDefs {
		n, ok := cameraPlanNormal[def.Title]
		if !ok {
			continue
		}
		cams := placeCameras(buildLevel(def), n)
		if len(cams) > n {
			t.Errorf("L%d %s: %d cameras, plan %d", i, def.Title, len(cams), n)
		}
		if len(cams) > 0 && i < campaignN {
			got[def.Title] = len(cams)
		}
		if len(cams) == 0 && i < len(campaignLevels) {
			t.Errorf("L%d %s: the plan asks for %d cameras and none fits", i, def.Title, n)
		}
	}
	t.Logf("campaign levels with cameras: %v", got)
	if len(got) < 5 {
		t.Errorf("only %d campaign levels have cameras, want 5-7", len(got))
	}
	for _, def := range levelDefs[:minI(len(levelDefs), len(campaignLevels))] {
		if _, ok := cameraPlanNormal[def.Title]; !ok && def.Title == "MIAMI BEACH" {
			continue
		}
	}
}

func TestCameraLoadLevelAndDrawDoNotPanic(t *testing.T) {
	g := layerGame(t, false)
	idx := -1
	for i, d := range levelDefs {
		if d.Title == "CINE" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("no CINE level")
	}
	check := func(stealth bool) {
		g.stealthNext = stealth
		g.loadLevel(idx)
		g.lvl.graded = true
		g.titleT = 0
		g.mode = ModePlay
		if len(g.cams) == 0 {
			t.Fatalf("stealth=%v: CINE must have cameras", stealth)
		}
		for i := 0; i < 90; i++ {
			g.time += 1.0 / 60
			g.updateCameras(1.0 / 60)
		}
		img := ebiten.NewImage(screenW, screenH)
		g.Draw(img)
		g.cams[0].alert = 2
		g.Draw(img)
		img.Deallocate()
	}
	check(false)
	if g.stealth {
		t.Error("normal load must not be stealth")
	}
	check(true)
	g.stealthNext = false
	g.loadLevel(0)
	if len(g.cams) == 0 { // HOUSE PARTY has one in the plan
		t.Log("HOUSE PARTY got no camera (plan allows it)")
	}
	for i, d := range levelDefs { // levels outside the plan have none in normal mode
		if _, ok := cameraPlanNormal[d.Title]; ok || i > 3 {
			continue
		}
		g.loadLevel(i)
		if len(g.cams) != 0 {
			t.Errorf("%s: not in the plan but has cameras", d.Title)
		}
	}
	g.loadLevel(0)
}

// -camkill: the guest dies in front of the camera, the cone turns red and the cuffs go on.
func TestCameraDebugKillFlag(t *testing.T) {
	g, c := camGame(t)
	c.still = false
	c.ang = c.base
	gu := suspectGuest(g, Vec{600, 400}, DirLeft)
	g.guests = []*Guest{gu}
	g.debugCamKill()
	if !gu.dead {
		t.Fatal("the guest must be dead")
	}
	if c.alert <= 0 || !g.policeCalled || !g.playerIsSuspect() {
		t.Errorf("the lens must catch it: alert=%.1f police=%v suspect=%v", c.alert, g.policeCalled, g.playerIsSuspect())
	}
}
