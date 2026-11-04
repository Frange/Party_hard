package main

import (
	"os"
	"strings"
	"testing"
)

var firstRunSnd *Sounds // the audio context can only be created once per process

func firstRunGame() *Game {
	if firstRunSnd == nil {
		firstRunSnd = newSounds(true)
	}
	progressFrozen = true // nothing here may touch the real progress.json
	debugSnap = nil
	return &Game{progress: Progress{Unlocked: 1}, snd: firstRunSnd, as: &Assets{}}
}

func hasRow(l []int, id int) bool {
	for _, x := range l {
		if x == id {
			return true
		}
	}
	return false
}

// a fresh game: the intro opens the program and the menu has no CONTINUAR
func TestFirstRunMenuHasNoContinue(t *testing.T) {
	g := firstRunGame()
	if !g.firstRun() {
		t.Fatal("an empty progress must be a first run")
	}
	l := g.menuList()
	if hasRow(l, menuContinue) {
		t.Fatal("CONTINUAR must not be there the first time")
	}
	if !hasRow(l, menuNewGame) || !hasRow(l, menuScenes) {
		t.Fatal("NUEVO JUEGO and ESCENAS must be in the menu")
	}
	g.bootIntro()
	if g.mode != ModeIntro || !g.introBoot || g.sceneReplay {
		t.Fatalf("the program must open with the intro (mode %v boot %v replay %v)", g.mode, g.introBoot, g.sceneReplay)
	}
	// the intro seen (or skipped): the main menu, on NUEVO JUEGO, still without CONTINUAR
	g.finishIntro()
	if g.mode != ModeMenu || g.introBoot || g.intro != nil {
		t.Fatal("after the intro the game must be in the main menu")
	}
	if l := g.menuList(); hasRow(l, menuContinue) || l[g.mainSel] != menuNewGame {
		t.Fatal("the menu must start on NUEVO JUEGO without CONTINUAR")
	}
	if g.firstRun() {
		t.Fatal("the intro has been seen: it must not play again at the next start")
	}
}

// NUEVO JUEGO goes straight to the first party (no intro) and then CONTINUAR is there
func TestNewGameSkipsIntroAndOpensContinue(t *testing.T) {
	src, _ := os.ReadFile("game.go")
	i := strings.Index(string(src), "func (g *Game) beginNewGame()")
	if i < 0 {
		t.Fatal("beginNewGame not found")
	}
	body := string(src)[i : i+300]
	if strings.Contains(body, "startIntro") {
		t.Fatal("beginNewGame must not play the intro again")
	}
	g := firstRunGame()
	g.seen("seen_intro")
	g.markSeen("seen_intro")
	g.progress.Stats["played"] = 1 // what startLevel records
	if !g.hasSave() || !hasRow(g.menuList(), menuContinue) {
		t.Fatal("once the first party is started CONTINUAR must show up")
	}
	if g.firstRun() {
		t.Fatal("not a first run any more")
	}
}

// BORRA TODO: everything goes (the cinematics seen too) and the cycle starts again
func TestWipeRepeatsTheFirstRun(t *testing.T) {
	g := firstRunGame()
	g.progress = Progress{Unlocked: 7, Wallet: 900, Owned: []string{"x"}, Stats: map[string]int{"played": 12, "seen_intro": 1, "seen_end1": 1, "kills": 80}}
	if g.firstRun() || !hasRow(g.menuList(), menuContinue) {
		t.Fatal("a player with a game must not see the first run")
	}
	g.wipeProgress()
	if g.progress.Unlocked != 1 || g.progress.Wallet != 0 || len(g.progress.Owned) != 0 || g.progress.Stats["kills"] != 0 || g.progress.Stats["seen_end1"] != 0 {
		t.Fatalf("the wipe left something: %+v", g.progress)
	}
	if g.mode != ModeIntro || !g.introBoot {
		t.Fatal("after the wipe the intro must play")
	}
	g.finishIntro()
	if g.mode != ModeMenu || hasRow(g.menuList(), menuContinue) {
		t.Fatal("after the wipe the menu has no CONTINUAR")
	}
}

// somebody who already played before the intro existed does not get it at start
func TestOldPlayersSkipTheOpeningIntro(t *testing.T) {
	g := firstRunGame()
	g.progress = Progress{Unlocked: 5, Stats: map[string]int{"played": 20}}
	if g.firstRun() {
		t.Fatal("a player with a game must go straight to the menu")
	}
}
