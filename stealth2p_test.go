package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// two-player stealth game: level 1, both players, no stray guests or guards.
func stealth2Game(t *testing.T) *Game {
	t.Helper()
	oldF, oldS, oldP := progressFrozen, debugSnap, debugPrevFrozen
	pm, pd := twoPlayerMode, dbgTwoPlayers
	t.Cleanup(func() {
		progressFrozen, debugSnap, debugPrevFrozen = oldF, oldS, oldP
		twoPlayerMode, dbgTwoPlayers = pm, pd
	})
	progressFrozen = true
	twoPlayerMode, dbgTwoPlayers = false, false // the stealth 2P does not depend on the switch
	g := layerGame(t, false)
	oldSnd := g.snd
	t.Cleanup(func() {
		g.snd = oldSnd
		g.stealth, g.stealth2, g.stealthNext, g.stealth2Next, g.sth = false, false, false, false, stealthState{}
	})
	g.snd = &Sounds{muted: true, sfx: map[string][]byte{}}
	g.startStealth2P(0)
	g.guards, g.cops, g.cams = nil, nil, nil
	g.policeCalled = false
	g.p.pos = Vec{200, 300}
	g.p2.pos = Vec{200, 420}
	for _, gu := range g.guests {
		gu.gone = true
	}
	return g
}

func TestStealth2PStartsWithTwoPlayersWithoutTheSwitch(t *testing.T) {
	g := stealth2Game(t)
	if !g.stealth || !g.stealth2 || !g.is2P() || g.p2 == nil {
		t.Fatal("stealth 2P must have two players")
	}
	// the normal stealth drops the second one even with the switch on
	twoPlayerMode, dbgTwoPlayers = true, true
	g.startStealth(0)
	if g.stealth2 || g.is2P() {
		t.Error("the one-player stealth must stay with one player")
	}
}

func TestStealth2PDetectionIsPerPlayerAndAlertIsShared(t *testing.T) {
	g := stealth2Game(t)
	guardAt(g, Vec{260, 300}, DirLeft) // only player 1 is in front
	runStealth(g, 1.2)
	if g.sth.phase != sthRed || g.sth.by[0] != 1 || g.sth.by[1] != 0 || g.sth.culprit != 0 {
		t.Fatalf("phase %d by %v culprit %d: only player 1 was seen", g.sth.phase, g.sth.by, g.sth.culprit)
	}
	if g.sth.two.meter != 0 {
		t.Error("player 2's meter must stay empty")
	}
	if !g.policeCalled {
		t.Error("the alert calls the police for both")
	}
}

func TestStealth2PSecondPlayerSeenIsHisFault(t *testing.T) {
	g := stealth2Game(t)
	guardAt(g, Vec{260, 420}, DirLeft)
	runStealth(g, 1.2)
	if g.sth.by[1] != 1 || g.sth.by[0] != 0 || g.sth.culprit != 1 {
		t.Fatalf("by %v culprit %d", g.sth.by, g.sth.culprit)
	}
	if g.stealthCulpritText() != "La alerta la provocó el J2" {
		t.Error("the panel must name player 2")
	}
}

func TestStealth2PBoxIsPerPlayer(t *testing.T) {
	g := stealth2Game(t)
	guardAt(g, Vec{260, 300}, DirLeft)
	guardAt(g, Vec{260, 420}, DirLeft)
	g.sth.two.box = true // player 2 hides, player 1 does not
	if !g.stealthBoxed(g.p2) || g.stealthBoxed(g.p) {
		t.Fatal("the box belongs to player 2 only")
	}
	runStealth(g, 1.2)
	if g.sth.by[1] != 0 || g.sth.by[0] != 1 {
		t.Errorf("by %v: the boxed one must not be seen", g.sth.by)
	}
	// the key: E as player 2 toggles HIS box
	g.sth.box, g.sth.two.box, g.sth.phase = false, false, sthCalm
	g.asPlayer(1, func() {
		if !g.stealthInteract(g.p) {
			t.Error("E of player 2 must work")
		}
	})
	if !g.sth.two.box || g.sth.box {
		t.Error("player 2's E put the wrong box on")
	}
}

func TestStealth2PNoiseAttractsWithoutDetecting(t *testing.T) {
	g := stealth2Game(t)
	gd := guardAt(g, Vec{200, 380}, DirDown) // 80 px below player 1, looking away from him
	gd.route = []Vec{gd.pos}
	gd.home = gd.pos
	g.p.dir = DirDown
	g.time = 10
	if !g.stealthWhistle(g.p) {
		t.Fatal("the decoy key must be taken")
	}
	if g.sth.noiseT <= 0 || g.sth.whistles[0] != 1 {
		t.Fatal("no noise was made")
	}
	if g.sth.detections != 0 || g.sth.phase != sthCalm {
		t.Error("the noise itself is not a detection")
	}
	// a guard far from the noise but inside the range goes to it
	far := guardAt(g, Vec{g.sth.noisePos.X + 90, g.sth.noisePos.Y}, DirUp)
	far.route, far.home = []Vec{far.pos}, far.pos
	for i := 0; i < 60*5; i++ {
		g.time += 1.0 / 60
		g.updateStealthNoise(1.0 / 60)
		g.updateGuard(far, 1.0/60)
	}
	if d := dist(far.pos, g.sth.noisePos); d > 40 {
		t.Errorf("the guard did not go to the noise: %.0f px away", d)
	}
	if g.sth.detections != 0 {
		t.Error("attracting guards must not detect anyone")
	}
	// cool-down
	g.time = 11
	if g.stealthWhistle(g.p); g.sth.whistles[0] != 1 {
		t.Error("a second noise at once must not work")
	}
}

func TestStealth2PNoiseNeedsAnAuthority(t *testing.T) {
	g := stealth2Game(t)
	if !g.stealthWhistle(g.p) || g.sth.noiseT > 0 {
		t.Error("with nobody near the key is taken but makes no noise")
	}
	if g.stealth2 = false; g.stealthWhistle(g.p) {
		t.Error("outside the 2P mode the key must dance as usual")
	}
}

func TestStealth2PWinTogether(t *testing.T) {
	g := stealth2Game(t)
	g.stealthWin()
	if g.mode != ModeLevelDone || len(g.toasts) == 0 {
		t.Fatal("clean party: the win and the badge")
	}
	if g.stealthDoneTitle("x") != "SIN SER VISTO" || g.stealthDoneSub("x") != "Fantasmas perfectos: nadie os detectó" {
		t.Error("two players unseen: wrong texts")
	}
	g.sth.detections = 1
	if g.stealthDoneTitle("x") == "SIN SER VISTO" {
		t.Error("detected: no badge")
	}
}

func TestStealth2PProgressFrozenAndDebugFlag(t *testing.T) {
	g := stealth2Game(t)
	if !progressFrozen {
		t.Error("progress must be frozen")
	}
	_ = g
	found := false
	for _, r := range g.debugRows() {
		if r.label == stealth2pMenuLabel {
			found = true
		}
	}
	if !found {
		t.Error("the DEBUG menu has no row for the stealth 2P")
	}
	for _, id := range g.menuList() {
		if menuLabel(id) == stealth2pMenuLabel {
			t.Error("not in the main menu")
		}
	}
	if rows := g.debugRows(); debugRow0+float64(len(rows))*debugRowH > 496 {
		t.Errorf("%d DEBUG rows do not fit", len(rows))
	}
}

func TestStealth2PDrawsWithoutPanic(t *testing.T) {
	g := stealth2Game(t)
	guardAt(g, Vec{260, 300}, DirLeft)
	g.sth.phase, g.sth.t, g.sth.culprit, g.sth.exT, g.sth.exMax = sthRed, 5, 1, 1, 1.6
	g.sth.meter, g.sth.two.meter, g.sth.two.box = 0.5, 0.8, true
	g.sth.noiseFx, g.sth.noisePos = 0.6, Vec{300, 300}
	img := ebiten.NewImage(screenW, screenH)
	g.drawStealth(img)
	g.sth.phase = sthCalm
	g.drawStealth(img)
}
