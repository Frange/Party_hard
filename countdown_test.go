package main

import (
	"math/rand"
	"testing"
)

// cdTestGame: the shared game in a party of level idx, with the countdown globals put back afterwards.
func cdTestGame(t *testing.T, idx int, two bool) *Game {
	t.Helper()
	g := partyGame(t)
	prev := [...]any{cdRng, cdSeed, cdForce, cdForceDur, g.fromEditor}
	t.Cleanup(func() {
		cdRng, cdSeed, cdForce, cdForceDur = prev[0].(*rand.Rand), prev[1].(int64), prev[2].(bool), prev[3].(float64)
		g.fromEditor = prev[4].(bool)
		g.cd = cdState{}
	})
	g = peLoad(t, idx, two)
	return g
}

// The closing time grows with the people of the party, from 7 to 10 minutes.
func TestCountdownDurationByGuests(t *testing.T) {
	cases := map[int]float64{0: 420, 30: 420, 40: 420, 50: 480, 60: 540, 65: 570, 70: 600, 95: 600, 140: 600}
	for n, want := range cases {
		if got := cdDuration(n); got != want {
			t.Errorf("cdDuration(%d) = %.0f, want %.0f", n, got, want)
		}
	}
	for n := 1; n < 150; n++ {
		if cdDuration(n) < cdDuration(n-1) || cdDuration(n) < 420 || cdDuration(n) > 600 {
			t.Fatalf("cdDuration(%d) = %.0f", n, cdDuration(n))
		}
	}
}

// About one party in twelve (fixed seed, 24000 rolls: between 7 % and 9.7 %).
func TestCountdownChanceAboutOneInTwelve(t *testing.T) {
	r := rand.New(rand.NewSource(1212))
	yes, n := 0, 24000
	for i := 0; i < n; i++ {
		if cdRoll(r) {
			yes++
		}
	}
	f := float64(yes) / float64(n)
	t.Logf("countdown in %.2f %% of the parties", f*100)
	if f < 0.070 || f > 0.097 {
		t.Errorf("countdown in %.2f %% of the parties, want about 8.3 %%", f*100)
	}
}

// In real parties (loadLevel), with the real dice: some parties get it, most do not, and the length follows the
// kill target of each one.
func TestCountdownInRealParties(t *testing.T) {
	g := cdTestGame(t, 0, false)
	cdRng, cdRollFrozen = rand.New(rand.NewSource(77)), true
	t.Cleanup(func() { cdRollFrozen = false })
	on := 0
	for i := 0; i < 48; i++ {
		idx := i % 6
		g.loadLevel(idx)
		if g.cd.on {
			on++
			if g.cd.dur != cdDuration(g.total) || g.cd.end != g.time+g.cd.dur {
				t.Errorf("level %d: length %.0f for a target of %d", idx, g.cd.dur, g.total)
			}
		}
	}
	if on == 0 || on > 14 {
		t.Errorf("%d of 48 parties had a countdown", on)
	}
}

// Never in DEBUG (progress frozen) unless forced, nor in STEALTH, the RETO DE HOY, the island or a level from the editor.
func TestCountdownNotWhereItDoesNotBelong(t *testing.T) {
	g := cdTestGame(t, 0, false)
	cdRng = rand.New(rand.NewSource(1))
	progressFrozen = true // DEBUG / capture flags
	for i := 0; i < 60; i++ {
		g.loadLevel(0)
		if g.cd.on {
			t.Fatal("a countdown in DEBUG")
		}
	}
	cdForce = true
	g.loadLevel(0)
	if !g.cd.on {
		t.Fatal("forced, no countdown")
	}
	for _, c := range []struct {
		name string
		set  func()
		undo func()
	}{
		{"stealth", func() { g.stealth = true }, func() { g.stealth = false }},
		{"daily", func() { g.dc = &dailyChallenge{} }, func() { g.dc = nil }},
		{"editor", func() { g.fromEditor = true }, func() { g.fromEditor = false }},
		{"island", func() { g.levelIdx = specialJurassic }, func() { g.levelIdx = 0 }},
	} {
		c.set()
		g.cdLevelReset()
		if g.cd.on {
			t.Errorf("a countdown in %s", c.name)
		}
		c.undo()
	}
}

// It runs on the game time: it stops while the game is paused (the PAUSE window) and goes on after.
func TestCountdownStopsWithThePause(t *testing.T) {
	g := cdTestGame(t, 0, false)
	cdForce = true
	g.loadLevel(0)
	g.mode = ModePlay
	l0, ok := g.cdLeft()
	if !ok {
		t.Fatal("no countdown")
	}
	g.pause = pauseState{open: true}
	t.Cleanup(func() { g.pause = pauseState{} })
	for i := 0; i < 120; i++ { // Update stops at updatePause while it is open: updatePlay (g.time, cdTick) does not run
		if !g.updatePause() {
			t.Fatal("the pause window let the party run")
		}
	}
	if l1, _ := g.cdLeft(); l1 != l0 {
		t.Errorf("it ran while paused: %.2f -> %.2f", l0, l1)
	}
	g.pause = pauseState{}
	g.time += 10
	if l2, _ := g.cdLeft(); l2 > l0-9.9 {
		t.Errorf("it does not run with the game: %.2f -> %.2f", l0, l2)
	}
}

// At 0 the police come in force (wanted, three stars, two more cars on top of the first) and the party is closed: no
// more guests. One clock for two players. Finished before 0: nothing happens.
func TestCountdownEndsWithThePolice(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := cdTestGame(t, 0, two)
		cdForce, cdForceDur = true, 5
		g.loadLevel(0)
		g.mode = ModePlay
		if !g.cd.on || g.cd.dur != 5 {
			t.Fatal("no short countdown")
		}
		g.dbg.noPolice = false
		for i := 0; i < 60*4; i++ {
			g.time += 1.0 / 60
			g.cdTick()
		}
		if g.cd.fired || g.policeCalled {
			t.Fatal("it fired early")
		}
		room := g.guestRoom()
		for i := 0; i < 60*20; i++ {
			g.time += 1.0 / 60
			g.cdTick()
			g.updateVehicles(1.0 / 60)
		}
		if !g.cd.fired || !g.policeCalled || g.stars < cdStars || g.wanted <= 0 {
			t.Fatalf("at 0: fired %v police %v stars %d wanted %.0f", g.cd.fired, g.policeCalled, g.stars, g.wanted)
		}
		if g.cd.extra != 0 {
			t.Errorf("%d extra cars not sent", g.cd.extra)
		}
		if room > 0 && g.guestRoom() != 0 || !g.arrivalsClosed() {
			t.Error("the party is not closed to new guests")
		}
		if g.p.dead || two && g.p2 != nil && g.p2.dead {
			t.Error("the countdown killed a player")
		}
	}
}

// The line goes under the HUD panel, with the margins of the rules (12 px from the panel, 16 px from the left edge,
// inside the screen in Spanish and English), and the notes under it (OBJETO) move down so nothing touches.
func TestCountdownLineMargins(t *testing.T) {
	g := cdTestGame(t, 0, false)
	cdForce = true
	g.loadLevel(0)
	g.mode = ModePlay
	g.p.item = "poison"
	prevLang := uiLang
	t.Cleanup(func() { uiLang = prevLang })
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, fired := range []bool{false, true} {
			g.cd.fired = fired
			x, y, w, h := g.cdRect()
			panelBottom := g.hud.bottom - 3
			if y-panelBottom < 12 || x < 16 || x+w > screenW-16 {
				t.Errorf("%s fired=%v: line at %.0f,%.0f w %.0f (panel ends at %.0f)", lang, fired, x, y, w, panelBottom)
			}
			obj, _, _, _ := g.hudSide()
			if obj-(y+h) < 4 {
				t.Errorf("%s: the OBJETO line (y %.0f) touches the countdown (ends %.0f)", lang, obj, y+h)
			}
			if x != float64(int(x)) || y != float64(int(y)) {
				t.Error("the line is not on whole pixels")
			}
		}
	}
	g.cd = cdState{}
	if g.cdLineH() != 0 {
		t.Error("the line takes room without a countdown")
	}
}
