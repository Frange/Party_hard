package main

import "testing"

func TestPlayersFlagResolve(t *testing.T) {
	for _, c := range []struct {
		flag  int
		saved bool
		want  bool
	}{
		{1, false, false}, {1, true, false}, {2, false, true}, {2, true, true},
		{0, false, false}, {0, true, true}, {3, true, true}, {-1, false, false},
	} {
		if got := resolvePlayers(c.flag, c.saved); got != c.want {
			t.Errorf("-players %d saved=%v: got %v want %v", c.flag, c.saved, got, c.want)
		}
	}
}

// -leveldonedemo 2 -players 2 with no -level goes straight to the results of level 1 (it used to wait on the beds cinematic).
func TestLevelDoneDemoFlagStartsALevel(t *testing.T) {
	if demoLevel(0, 2, false) != 1 || demoLevel(0, 1, false) != 1 {
		t.Fatal("no level given: must start level 1")
	}
	if demoLevel(5, 2, false) != 5 {
		t.Fatal("an explicit -level must be kept")
	}
	if demoLevel(0, 0, false) != 0 || demoLevel(0, 2, true) != 0 {
		t.Fatal("no demo (or stealth) must not start a level")
	}
}

// The demo setup lands on the newspaper (the only end screen) for two players once a party runs.
func TestLevelDoneDemoTwoPlayersNewspaper(t *testing.T) {
	g := hudPlay(t, true)
	prevP, prevF, prevD := cloneProgress(g.progress), progressFrozen, levelDoneDemo
	progressFrozen = true
	t.Cleanup(func() {
		g.progress, progressFrozen, levelDoneDemo = prevP, prevF, prevD
		g.mode = ModeMenu
	})
	levelDoneDemo = 2
	g.time = 1
	g.levelDoneDemoSetup()
	if g.mode != ModeLevelDone {
		t.Fatalf("mode=%v, want the newspaper (ModeLevelDone)", g.mode)
	}
}
