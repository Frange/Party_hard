package main

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// The anthem made of the chorus repeated, its dip under the end of the title, and ONE music from the
// first instant of the program to the menu (no other song first, no switch, no cut).
// ---------------------------------------------------------------------------

// rmsOf: the RMS of a slice of samples.
func rmsOf(x []float64) float64 {
	e := 0.0
	for _, v := range x {
		e += v * v
	}
	return math.Sqrt(e / float64(len(x)))
}

// the anthem is only the chorus of the intro, repeated: every section but the one-bar dip is partyHook over
// A A C G, and the same section always sounds the same (no invented melodies or chords any more)
func TestAnthemIsTheChorusRepeated(t *testing.T) {
	l := thumpLeft(ambientPCM(ambAnthem))
	if len(l) != anthemBars*anthemBarSamples {
		t.Fatalf("anthem length %d, want %d", len(l), anthemBars*anthemBarSamples)
	}
	secs := float64(len(l)) / sampleRate
	if secs < 60 {
		t.Fatalf("the anthem lasts only %.1f s: it must stay long", secs)
	}
	checkLoopClean(t, "anthem", l)
	dips := 0
	bar := func(b0, nb int) []float64 { return l[b0*anthemBarSamples : (b0+nb)*anthemBarSamples] }
	corr := func(a, b []float64) float64 {
		var ab, aa, bb float64
		for i := range a {
			ab += a[i] * b[i]
			aa += a[i] * a[i]
			bb += b[i] * b[i]
		}
		return ab / math.Sqrt(aa*bb)
	}
	first := bar(0, 4)
	firstOf := map[int][]float64{}
	lay := anthemLayout()
	for i, k := range anthemForm {
		b0 := lay[i]
		if k == secDip {
			dips++
			if b0 != anthemDipBar {
				t.Errorf("the dip is at bar %d, not %d", b0, anthemDipBar)
			}
			continue
		}
		s := bar(b0, 4)
		c := corr(first, s)
		t.Logf("bars %2d-%2d (%5.1f-%5.1f s) section %d: correlation with the first chorus %.2f", b0, b0+3,
			float64(b0*anthemBarSamples)/sampleRate, float64((b0+4)*anthemBarSamples)/sampleRate, k, c)
		if c < 0.5 {
			t.Errorf("bars %d-%d are not the chorus (correlation %.2f with bars 0-3)", b0, b0+3, c)
		}
		if p, ok := firstOf[k]; ok {
			if c2 := corr(p, s); c2 < 0.9 {
				t.Errorf("the repeat of section %d at bar %d is not the same music (%.2f)", k, b0, c2)
			}
		} else {
			firstOf[k] = s
		}
	}
	if dips != 1 || anthemForm[0] != secHook {
		t.Fatalf("the anthem must open with the chorus and have one dip (%d)", dips)
	}
	if parseNotes(partyHook)[0].midi[0] != 69 {
		t.Error("the hook changed")
	}
}

// the dip of the anthem falls on the fade to black of the title: the drums and the bass stop, the music
// gets clearly quieter but never silent, and the drop of the next chorus comes in without a click
func TestAnthemDipUnderTheEndOfTheTitle(t *testing.T) {
	if math.Abs(tiEnd-anthemDipAt) > 1e-9 {
		t.Fatalf("the fade to black starts at %.3f s, the dip at %.3f s", tiEnd, anthemDipAt)
	}
	if d := titleDuration - (6.1 + 0.55); d < 1.5 || d > 2.5 {
		t.Fatalf("the title must last a little longer (+1.5..2.5 s), not %+.2f s", d)
	}
	l := thumpLeft(ambientPCM(ambAnthem))
	at := func(s float64) int { return int(s * sampleRate) }
	dip0, dip1 := anthemDipBar*anthemBarSamples, (anthemDipBar+1)*anthemBarSamples
	hook := rmsOf(l[0:dip0])
	dip := rmsOf(l[dip0:dip1])
	drop := rmsOf(l[dip1 : dip1+4*anthemBarSamples])
	t.Logf("rms: chorus bars 0-3 %.4f, dip (bar 4, %.2f-%.2f s) %.4f (%.0f%%), drop bars 5-8 %.4f",
		hook, float64(dip0)/sampleRate, float64(dip1)/sampleRate, dip, 100*dip/hook, drop)
	if dip > hook*0.6 || dip < hook*0.12 {
		t.Errorf("the dip must be clearly quieter but not silent: %.0f%% of the chorus", 100*dip/hook)
	}
	if drop < hook*0.9 {
		t.Errorf("the drop after the dip does not come back up (%.4f vs %.4f)", drop, hook)
	}
	if s := subBassShare(toPCM(l[dip0+thumpBeatSamples/2 : dip1])); s > subBassShare(toPCM(l[0:dip0]))/3 {
		t.Errorf("the dip still has the kick (%.0f%% sub-bass)", s*100)
	}
	// 50 ms windows over the end of the title (7.0 s .. 9.9 s): never a hole of silence
	minW, minAt := 1.0, 0.0
	for s := 7.0; s < float64(dip1)/sampleRate+0.2; s += 0.05 {
		r := rmsOf(l[at(s):at(s+0.05)])
		if r < minW {
			minW, minAt = r, s
		}
	}
	t.Logf("quietest 50 ms between 7.0 and %.1f s: rms %.4f at %.2f s (%.0f%% of the chorus)", float64(dip1)/sampleRate+0.2, minW, minAt, 100*minW/hook)
	if minW < hook*0.05 {
		t.Errorf("a hole of silence at %.2f s (rms %.4f)", minAt, minW)
	}
	// no click where the dip starts or where the drop comes in
	steps := func(a, b int) float64 {
		m := 0.0
		for i := a; i < b; i++ {
			m = math.Max(m, math.Abs(l[i]-l[i-1]))
		}
		return m
	}
	all := steps(1, len(l))
	for _, b := range []int{dip0, dip1} {
		if s := steps(b-480, b+480); s > all*0.999+1e-9 && s > 0.2 {
			t.Errorf("a jump of %.3f at sample %d", s, b)
		}
	}
	t.Logf("biggest step: whole song %.3f, at the dip %.3f, at the drop %.3f", all, steps(dip0-480, dip0+480), steps(dip1-480, dip1+480))
}

// menuFlowGame: an empty (new) game with a party level loaded behind the menu, as the real program has
// (g.lvl is never nil) and no mp3 library: the synthesised party song is what could sneak in.
func menuFlowGame(t *testing.T) *Game {
	g := firstRunGame()
	g.snd = newSounds(true) // muted: the track that is asked for still changes, nothing is heard
	g.rng = rand.New(rand.NewSource(1))
	g.lvl = &Level{Def: &LevelDef{Title: "LA PRIMERA FIESTA"}}
	g.lib = &musicLib{level: map[int]string{}, bands: map[string][]string{}}
	g.nextSong = 1e9
	// the party song ready, as it is a second after the program opens
	sp := playlistFor(g.lvl.Def.Title)[0]
	if _, isGenre := genreSongs[sp.style]; isGenre {
		genreWarm([]songSpec{sp})
		for i := 0; i < 400 && genreReadyPCM(sp) == nil; i++ {
			time.Sleep(25 * time.Millisecond)
		}
	}
	ambientPCM(ambAnthem)
	t.Cleanup(func() { g.snd.StopMusic() })
	return g
}

// a new game: title, intro of the room, menu. From the first frame of the title to 6 s into the menu only
// the menu's track is ever asked for, and in the menu it starts at once (it is not replaced 2 s later).
func TestNewGameMenuMusicNeverSwitches(t *testing.T) {
	g := menuFlowGame(t)
	if !g.firstRun() {
		t.Fatal("an empty progress must be a first run")
	}
	var log []string
	note := func(where string) {
		if len(log) == 0 || log[len(log)-1] != where+" "+g.snd.cur {
			log = append(log, where+" "+g.snd.cur)
		}
	}
	g.startTitle()
	for i := 0; i < 60*12 && g.mode == ModeTitle; i++ {
		g.syncMusic() // the order of Game.Update: syncMusic, then the title
		if g.snd.cur != menuKeyAnthem && !(i == 0 && g.snd.cur == "") {
			t.Fatalf("title, %.2f s: the music is %q, not the menu's anthem", float64(i)/60, g.snd.cur)
		}
		g.updateTitle()
		note("title")
		if g.snd.cur != menuKeyAnthem {
			t.Fatalf("title, %.2f s: the music is %q after the frame, not the menu's anthem", float64(i)/60, g.snd.cur)
		}
	}
	if g.mode != ModeIntro {
		t.Fatalf("a new game goes from the title to the intro (mode %v)", g.mode)
	}
	for i := 0; i < 60*6; i++ { // the intro: the party of the level must not start behind it
		g.syncMusic()
		note("intro")
		if g.snd.cur != menuKeyAnthem {
			t.Fatalf("intro, %.2f s: %q started behind the intro (the menu would open with it)", float64(i)/60, g.snd.cur)
		}
	}
	g.finishIntro()
	if g.mode != ModeMenu {
		t.Fatalf("after the intro: the menu (mode %v)", g.mode)
	}
	for i := 0; i < 60*6; i++ {
		g.syncMusic()
		note("menu")
		if g.snd.cur != menuKeyAnthem || g.snd.fadePend != "" {
			t.Fatalf("menu, %.2f s: the music is %q (pending %q): it switches", float64(i)/60, g.snd.cur, g.snd.fadePend)
		}
	}
	t.Logf("tracks asked for: %v", log)
}

// somebody who has played: the title and the menu are ONE song on ONE player; the menu does not restart it,
// does not fade it out and in, and the music is at full level when the menu appears
func TestTitleMusicGoesOnInTheMenu(t *testing.T) {
	g := menuFlowGame(t)
	g.progress.Stats = map[string]int{"played": 3, "seen_intro": 1}
	g.startTitle()
	g.syncMusic()
	if g.snd.music != nil {
		t.Fatal("the music must start with the first frame of the title, not before")
	}
	g.updateTitle()
	p := g.snd.music
	if p == nil || g.snd.cur != menuKeyAnthem {
		t.Fatalf("the title must start the menu's anthem (%q)", g.snd.cur)
	}
	prevGain, maxJump := g.snd.fadeGain(), 0.0
	frames := 1
	for g.mode == ModeTitle && frames < 60*12 {
		g.syncMusic()
		g.updateTitle()
		frames++
		gn := g.snd.fadeGain()
		maxJump = math.Max(maxJump, math.Abs(gn-prevGain))
		prevGain = gn
	}
	t.Logf("title lasted %.2f s (%d frames); music gain at the menu %.3f, biggest gain step per frame %.4f", float64(frames)/60, frames, g.snd.fadeGain(), maxJump)
	if g.mode != ModeMenu {
		t.Fatalf("the title must lead to the menu (mode %v)", g.mode)
	}
	if math.Abs(float64(frames)/60-titleDuration) > 0.05 {
		t.Errorf("the title lasted %.2f s, not %.2f", float64(frames)/60, titleDuration)
	}
	for i := 0; i < 60*6; i++ {
		g.syncMusic()
		gn := g.snd.fadeGain()
		maxJump = math.Max(maxJump, math.Abs(gn-prevGain))
		prevGain = gn
		if g.snd.music != p || g.snd.cur != menuKeyAnthem {
			t.Fatalf("menu, %.2f s: the music player changed (%q): the song restarts", float64(i)/60, g.snd.cur)
		}
		if gn < 0.999 {
			t.Fatalf("menu, %.2f s: the music dips (gain %.3f)", float64(i)/60, gn)
		}
	}
	if maxJump > 1.0/60/menuFadeSecs*1.6 {
		t.Errorf("the music level jumps by %.3f in one frame", maxJump)
	}
	// where the song is when the menu appears: inside the dip, before the drop
	if at := titleDuration; at < anthemDipAt || at > anthemDipAt+float64(anthemBarSamples)/sampleRate {
		t.Errorf("the menu appears at %.2f s, outside the dip (%.2f s + 1 bar)", at, anthemDipAt)
	}
}
