package main

import (
	"math/rand"
	"testing"
)

// flowGame: a game with real sound and frozen progress; the level that is "just finished" is idx.
func flowGame(idx int, two bool) *Game {
	if len(levelDefs) <= len(campaignLevels) { // the whole campaign (and the random pool) as in the real game
		levelDefs = builtinLevels()
	}
	if campaignN != len(campaignLevels) {
		panic("every party of the campaign must build")
	}
	g := firstRunGame()
	g.rng = rand.New(rand.NewSource(7))
	g.progress = Progress{Unlocked: campaignN, Stats: map[string]int{}}
	twoPlayerMode = two
	g.levelIdx = idx
	g.mode = ModeLevelDone
	testStartLevel = func(g *Game, i int) { // what loadLevel + startLevel leave behind
		g.levelIdx, g.mode, g.inter = i, ModePlay, nil
	}
	return g
}

// leave: what the player does at the "party over" screen, and where it leads.
func leaveLevel(g *Game) { g.afterLevel() }

func flowReset() { twoPlayerMode, testStartLevel = false, nil }

func TestInterludeFlowCampaign(t *testing.T) {
	defer flowReset()
	for _, two := range []bool{false, true} {
		for idx := 0; idx < campaignN-1; idx++ {
			g := flowGame(idx, two)
			leaveLevel(g)
			wantInter := map[int]int{4: interCar, 9: interBoat, 14: interBalloon, campaignN - 2: interRocket}
			kind, isInter := wantInter[idx]
			if isInter {
				if g.mode != ModeInterlude || g.inter == nil || g.inter.kind != kind || g.inter.next != idx+1 || g.inter.two != two {
					t.Fatalf("2P=%v level %d: expected interlude %d before level %d (mode %v)", two, idx+1, kind, idx+2, g.mode)
				}
				if g.levelIdx != idx {
					t.Fatalf("level %d: the next party was loaded before the interlude", idx+1)
				}
				// the interlude ends: the NEXT party is loaded and played
				g.finishInterlude()
				if g.mode != ModePlay || g.levelIdx != idx+1 || g.inter != nil {
					t.Fatalf("2P=%v interlude after %d: next party not loaded (mode %v idx %d)", two, idx+1, g.mode, g.levelIdx)
				}
			} else if g.mode != ModePlay || g.levelIdx != idx+1 || g.inter != nil {
				t.Fatalf("2P=%v level %d: no interlude expected, went to mode %v idx %d", two, idx+1, g.mode, g.levelIdx)
			}
		}
	}
}

// the rocket goes between the 70s disco and the space station (the last two of the list), only once
func TestRocketBeforeSpaceStationOnce(t *testing.T) {
	defer flowReset()
	for _, two := range []bool{false, true} {
		g := flowGame(campaignN-2, two)
		leaveLevel(g)
		if g.mode != ModeInterlude || g.inter.kind != interRocket || g.inter.next != campaignN-1 {
			t.Fatal("beating party 23 must show the rocket, then party 24")
		}
		if levelDefs[campaignN-2].Title != "DISCO SETENTERA" || levelDefs[campaignN-1].Title != "ESTACIÓN ESPACIAL" {
			t.Fatalf("level order changed: %q -> %q", levelDefs[campaignN-2].Title, levelDefs[campaignN-1].Title)
		}
		if !g.seen("seen_interlude20") {
			t.Error("marked only at the end: a retry after dying would repeat it")
		}
		// dying in party 24 and retrying, or beating 23 again: no rocket
		g.finishInterlude()
		g.mode, g.levelIdx = ModeLevelDone, campaignN-2
		leaveLevel(g)
		if g.mode != ModePlay || g.levelIdx != campaignN-1 {
			t.Fatalf("2P=%v: the rocket repeated (mode %v)", two, g.mode)
		}
		g.startLevel(g.levelIdx) // retry after losing
		if g.mode != ModePlay || g.inter != nil {
			t.Fatal("a retry must not show an interlude")
		}
	}
}

// choosing a party from the list, losing and retrying, random mode, the end: no rocket
func TestRocketNeverElsewhere(t *testing.T) {
	defer flowReset()
	g := flowGame(0, false)
	for _, i := range []int{18, 19, 20, 21, 22, campaignN - 1, 1000} {
		if _, ok := g.interludeAfter(i); ok {
			t.Errorf("level index %d must not trigger an interlude", i)
		}
	}
	// the list (and random mode) call startLevel directly: never an interlude
	g.startLevel(campaignN - 1)
	if g.mode != ModePlay || g.inter != nil {
		t.Fatal("picking party 24 in the list must not play the rocket")
	}
	if campaignN < 24 {
		return
	}
	// random-mode parties (beyond the campaign) never trigger anything, even index 19 mod something
	for i := campaignN; i < len(levelDefs) && i < campaignN+3; i++ {
		g = flowGame(i, false)
		g.progress.Stats = map[string]int{}
		leaveLevel(g)
		if g.mode == ModeInterlude {
			t.Fatalf("random level %d showed an interlude", i)
		}
	}
	// already seen: not again (all three)
	g = flowGame(campaignN-2, false)
	g.progress.Stats["seen_interlude20"] = 1
	leaveLevel(g)
	if g.mode != ModePlay || g.levelIdx != campaignN-1 {
		t.Fatal("a seen rocket must not play again")
	}
}

// the 21st party closes the story: ending or invitation, never the rocket
func TestRocketNotAfterLastParty(t *testing.T) {
	defer flowReset()
	g := flowGame(campaignN-1, false)
	g.progress.Stats = map[string]int{}
	g.mode = ModePlay
	g.unlockNext()
	if _, ok := g.interludeAfter(campaignN - 1); ok {
		t.Fatal("the rocket must not come after the last party")
	}
}
