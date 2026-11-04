package main

import (
	"flag"
	"testing"
)

// run the title for `secs` seconds of game time
func runTitle(g *Game, secs float64) {
	for i := 0; i < int(secs*60) && g.mode == ModeTitle; i++ {
		g.updateTitle()
	}
}

// the title opens the program and, after a few seconds, goes on to the main menu by itself
func TestTitleGoesToMenuByItself(t *testing.T) {
	g := firstRunGame()
	g.progress.Stats = map[string]int{"played": 3, "seen_intro": 1} // somebody who has played before
	g.startTitle()
	if g.mode != ModeTitle || g.title == nil {
		t.Fatal("the title must open the program")
	}
	runTitle(g, tiEnd-1)
	if g.mode != ModeTitle {
		t.Fatal("the title must last a few seconds")
	}
	runTitle(g, 3)
	if g.mode != ModeMenu || g.title != nil {
		t.Fatalf("after the title: the main menu (mode %v)", g.mode)
	}
	if titleDuration < 4 || titleDuration > 8 {
		t.Fatalf("the title must last about 4-8 seconds, not %.1f", titleDuration)
	}
}

// Enter (or Esc, or a click) skips it fast
func TestTitleCanBeSkipped(t *testing.T) {
	g := firstRunGame()
	g.progress.Stats = map[string]int{"played": 1}
	g.startTitle()
	runTitle(g, 0.5)
	g.skipTitle()
	runTitle(g, tiSkipFade+0.1)
	if g.mode != ModeMenu {
		t.Fatalf("skipping the title must lead to the menu (mode %v)", g.mode)
	}
}

// the very first time the title is followed by the intro of the room, and the intro by the menu without CONTINUAR
func TestTitleThenIntroTheFirstTime(t *testing.T) {
	g := firstRunGame()
	if !g.firstRun() {
		t.Fatal("an empty game must be a first run")
	}
	g.startTitle()
	g.skipTitle()
	runTitle(g, 1)
	if g.mode != ModeIntro || !g.introBoot {
		t.Fatalf("the first time the title must lead to the intro (mode %v)", g.mode)
	}
	g.finishIntro()
	if g.mode != ModeMenu || hasRow(g.menuList(), menuContinue) {
		t.Fatal("after the intro: the menu without CONTINUAR")
	}
	// wiping everything shows the intro again, not the title
	g.wipeProgress()
	if g.mode != ModeIntro || g.title != nil {
		t.Fatalf("after BORRA TODO the intro of the room plays, not the title (mode %v)", g.mode)
	}
}

// only the program opened as a player does shows the title: any debug flag skips it, except -title
func TestPlainStartOnlyWithoutDebugFlags(t *testing.T) {
	mk := func(set ...string) *flag.FlagSet {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		for _, n := range []string{"mute", "lang", "noassets", "title", "notitle", "level", "shot", "intro", "firstrun"} {
			fs.String(n, "", "")
		}
		for _, n := range set {
			fs.Set(n, "1")
		}
		return fs
	}
	if !plainStartFor(mk()) || !plainStartFor(mk("mute", "lang", "noassets")) {
		t.Fatal("a normal start (or only mute / language / noassets) must show the title")
	}
	for _, f := range []string{"level", "shot", "intro", "firstrun"} {
		if plainStartFor(mk(f)) {
			t.Fatalf("-%s must skip the title", f)
		}
	}
}

// the music of the title exists, is short and bright
func TestTitleMusic(t *testing.T) {
	pcm := ambientPCM(ambTitle)
	if len(pcm) == 0 {
		t.Fatal("no music for the title")
	}
	secs := float64(len(pcm)) / (sampleRate * 4) // 16-bit stereo
	if secs < 4 || secs > 8 {
		t.Fatalf("the title tune lasts %.1f s", secs)
	}
	if sd := songs[ambTitle]; sd.bpm < 110 {
		t.Fatalf("the title tune must be lively, not %.0f BPM", sd.bpm)
	}
}

// the sparks are coloured, never white dots
func TestTitleSparksAreNotWhite(t *testing.T) {
	ti := &titleState{seedBase: 1}
	ti.addSparks(480, 200)
	if len(ti.sparks) == 0 {
		t.Fatal("no sparks")
	}
	for _, s := range ti.sparks {
		if int(s.col.R)+int(s.col.G)+int(s.col.B) > 600 {
			t.Fatalf("a nearly white spark: %v", s.col)
		}
	}
}
