package main

import (
	"testing"
)

// Rules of the cameras that LOCATE the player (camera.go, 2026-11-09): kill in the cone -> cuffs; a wanted
// player, or one passing by his own fresh knife body, is reported to the police; traps, accidents and
// blackouts never incriminate.

func camStep(g *Game, secs float64) {
	for tt := 0.0; tt < secs; tt += 1.0 / 60 {
		g.time += 1.0 / 60
		g.updateCameras(1.0 / 60)
	}
}

func TestCameraLocateKillInZoneMarksPlayer(t *testing.T) {
	g, c := camGame(t)
	gu := suspectGuest(g, Vec{210, 200}, DirLeft)
	g.killGuest(gu, true) // a real knife kill, right in the cone
	if !g.playerIsSuspect() || !g.policeCalled || g.killPos != g.p.pos {
		t.Fatalf("kill in the cone: suspect=%v police=%v killPos=%v", g.playerIsSuspect(), g.policeCalled, g.killPos)
	}
	if c.alert <= 0 || g.run.camSeen == 0 {
		t.Error("the camera must turn red and count the recording")
	}
}

func TestCameraLocateKillOutOfZoneDoesNothing(t *testing.T) {
	g, _ := camGame(t)
	g.p.pos = Vec{700, 400}
	gu := suspectGuest(g, Vec{710, 400}, DirLeft)
	g.killGuest(gu, true)
	camStep(g, 1)
	if g.playerIsSuspect() || g.policeCalled || g.run.camSeen != 0 {
		t.Error("a kill far from the camera, and nobody near: nothing")
	}
}

func TestCameraLocateBlackoutSeesNothing(t *testing.T) {
	g, c := camGame(t)
	g.blk.t = 5 // no power
	gu := suspectGuest(g, Vec{210, 200}, DirLeft)
	g.killGuest(gu, true)
	if g.playerIsSuspect() || g.policeCalled || c.alert > 0 {
		t.Error("blackout: the camera films nothing")
	}
	// a wanted player walking under it in the dark is not reported either
	g.policeCalled, g.wanted = true, 50
	cop := suspectCop(g, Vec{800, 450}, DirLeft)
	camStep(g, 0.5)
	if cop.state != cSweep || g.run.camSeen != 0 {
		t.Error("blackout: no report to the police")
	}
}

func TestCameraLocateFreshBodyReportsThePlayer(t *testing.T) {
	g, c := camGame(t)
	// the kill happens in the camera's zone but outside its cone (it looks straight down, this is 1 rad off)
	g.p.pos = Vec{284, 160}
	gu := suspectGuest(g, Vec{290, 160}, DirLeft)
	if c.sees(g, g.p.pos) || !c.camZone(g, gu.pos) {
		t.Fatal("setup: the kill must be in the zone and out of the cone")
	}
	g.killGuest(gu, true)
	if g.policeCalled || g.playerIsSuspect() {
		t.Fatal("the cone did not see the kill")
	}
	g.p.pos = Vec{200, 200} // now he walks under the lens with his victim lying there
	camStep(g, 0.1)
	if !g.policeCalled || g.alertPos != (Vec{200, 200}) || g.run.camSeen != 1 {
		t.Fatalf("passing by his fresh body: police=%v alertPos=%v rec=%d", g.policeCalled, g.alertPos, g.run.camSeen)
	}
	if g.playerIsSuspect() {
		t.Error("a location is not a kill on tape: no cuffs")
	}
	camStep(g, 1)
	if g.run.camSeen != 1 {
		t.Error("one report per camera every few seconds, not every frame")
	}
	// the body is old: no more reports
	g2, _ := camGame(t)
	g2.p.pos = Vec{284, 160}
	gu2 := suspectGuest(g2, Vec{290, 160}, DirLeft)
	g2.killGuest(gu2, true)
	g2.time += camRecent + 1
	g2.p.pos = Vec{200, 200}
	camStep(g2, 0.2)
	if g2.policeCalled || g2.run.camSeen != 0 {
		t.Error("an old body is not news for the camera")
	}
}

func TestCameraLocateTrapBodyDoesNotIncriminate(t *testing.T) {
	g, _ := camGame(t)
	gu := suspectGuest(g, Vec{284, 160}, DirLeft)
	gu.dead = true
	g.killByPlayer = true // his trap / poison: the body counts for his criminality, but it is not a knife
	g.spawnBody(gu.pos, gu.cs, gu.dir)
	g.killByPlayer = false
	if !g.bodies[len(g.bodies)-1].byPlayer {
		t.Fatal("setup: the trap body must be byPlayer")
	}
	camStep(g, 0.5)
	if g.policeCalled || g.playerIsSuspect() || g.run.camSeen != 0 {
		t.Error("a trap or accident body must not incriminate the player who walks past")
	}
}

func TestCameraLocateWantedPlayerSendsAnOfficer(t *testing.T) {
	g, _ := camGame(t)
	g.policeCalled, g.wanted = true, 50
	cop := suspectCop(g, Vec{800, 450}, DirLeft)
	camStep(g, 0.1)
	if cop.state != cGoto || g.killPos != g.p.pos || !g.seenNow {
		t.Fatalf("wanted under the lens: cop state=%d killPos=%v seenNow=%v", cop.state, g.killPos, g.seenNow)
	}
	if !g.playerIsSuspect() {
		t.Error("the cuffs stay on")
	}
	// out of the zone: the officer is not called
	g2, _ := camGame(t)
	g2.policeCalled, g2.wanted = true, 50
	g2.p.pos = Vec{700, 400}
	cop2 := suspectCop(g2, Vec{800, 450}, DirLeft)
	camStep(g2, 0.5)
	if cop2.state != cSweep || g2.run.camSeen != 0 {
		t.Error("a wanted player away from the cameras is not reported by them")
	}
}

func TestCameraLocateWantedNoPoliceCallsThemOnce(t *testing.T) {
	g, _ := camGame(t)
	g.wanted = 50 // wanted, no police called yet
	camStep(g, 0.1)
	if !g.policeCalled || g.alertPos != g.p.pos {
		t.Fatalf("no police on the scene: the camera calls them (police=%v alertPos=%v)", g.policeCalled, g.alertPos)
	}
	car := g.pendingCar
	camStep(g, camReportCD+0.5) // a second report must not make a second call
	if g.pendingCar != car {
		t.Error("only one call while the police are coming")
	}
}

func TestCameraFilmsTheKillerKilling(t *testing.T) {
	g, c := camGame(t)
	g.wanted = 40
	g.lastKiller = lastKill{kind: suspPlayer}
	g.slasherWitness(&Slasher{need: 3}, Vec{200, 200})
	if g.lastKiller.kind != suspSlasher || g.wanted != 0 || c.alert <= 0 {
		t.Errorf("the killer filmed killing is the suspect now (kind=%d wanted=%.0f)", g.lastKiller.kind, g.wanted)
	}
	g2, _ := camGame(t)
	g2.lastKiller = lastKill{kind: suspPlayer}
	g2.slasherWitness(&Slasher{need: 3}, Vec{700, 400})
	if g2.lastKiller.kind == suspSlasher {
		t.Error("out of the cone (and nobody around) the killer is not seen")
	}
}
