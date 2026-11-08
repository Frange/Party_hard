package main

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

// arrFlow: a game at the moment the letter is over (the story way: not a replay), with the level start recorded.
func arrFlow(two bool) (*Game, *int) {
	if len(levelDefs) < 22 {
		levelDefs = builtinLevels()
	}
	g := firstRunGame()
	g.rng = rand.New(rand.NewSource(3))
	g.progress = Progress{Unlocked: campaignN, Stats: map[string]int{}}
	twoPlayerMode = two
	starts := 0
	testStartLevel = func(g *Game, i int) {
		if i == specialJurassic {
			starts++
		}
		g.levelIdx, g.mode = i, ModePlay
	}
	g.startInvite()
	return g, &starts
}

func arrReset() { twoPlayerMode, testStartLevel = false, nil }

// run the scene for secs seconds; it returns the sounds and whether it ended
func runArrival(a *arrivalState, secs float64, presses ...float64) (sounds []string, done bool) {
	const dt = 1.0 / 60
	next, total := 0, 0.0
	for total < secs {
		press := false
		if next < len(presses) && total >= presses[next] {
			press = true
			next++
		}
		s, d := a.advance(dt, press)
		sounds = append(sounds, s...)
		if d {
			return sounds, true
		}
		total += dt
	}
	return sounds, false
}

// letter -> explosion of the letter -> arrival -> level: the island does not start before the arrival is over
func TestArrivalOrderLetterArrivalLevel(t *testing.T) {
	defer arrReset()
	for _, two := range []bool{false, true} {
		g, starts := arrFlow(two)
		iv := &g.end.x.inv
		runInvite(iv, 5)
		_, done, sounds := runInvite(iv, 30, 0, 3, 6)
		if !done || len(sounds) == 0 {
			t.Fatal("the letter never blew up")
		}
		g.finishInvite(g.end) // what updateInvite does when the letter is over
		if g.end == nil || g.end.stage != endArrival || g.mode != ModeEnding {
			t.Fatalf("2P=%v: after the letter comes the arrival, got stage %v mode %v", two, g.end, g.mode)
		}
		if *starts != 0 {
			t.Fatalf("2P=%v: the island started before the arrival", two)
		}
		if g.progress.Stats["seen_arrival"] != 1 || g.progress.Stats["jp_unlocked"] != 1 {
			t.Fatalf("2P=%v: seen_arrival / jp_unlocked not recorded: %v", two, g.progress.Stats)
		}
		a := &g.end.x.arr
		_, fin := runArrival(a, arrTotal-0.5)
		if fin || *starts != 0 {
			t.Fatalf("2P=%v: the arrival ended too soon", two)
		}
		_, fin = runArrival(a, 2)
		if !fin {
			t.Fatalf("2P=%v: the arrival never ended", two)
		}
		g.finishArrival(g.end)
		if *starts != 1 || g.levelIdx != specialJurassic || g.mode != ModePlay || g.end != nil {
			t.Fatalf("2P=%v: the island did not start once after the arrival (starts %d idx %d mode %v)", two, *starts, g.levelIdx, g.mode)
		}
	}
}

// the arrival is not part of the list of parties nor of DEBUG: those start the island directly, and a replay
// of the letter does not lead to it
func TestArrivalNotRepeatedFromListOrReplay(t *testing.T) {
	defer arrReset()
	g, starts := arrFlow(false)
	g.end = nil
	g.mode = ModeLevelSelect
	g.startLevel(specialJurassic) // ELIGE LA FIESTA / DEBUG
	if *starts != 1 || g.mode != ModePlay || g.end != nil {
		t.Fatal("the island from the list must start directly, with no arrival")
	}
	// the letter watched again from the scenes: back to the list, no arrival, no level
	g, starts = arrFlow(false)
	g.playScene(sceneInvite, ModeScenes)
	g.finishInvite(g.end)
	if *starts != 0 || g.mode != ModeScenes || g.end != nil {
		t.Fatalf("a replayed letter must return to the scenes (mode %v, starts %d)", g.mode, *starts)
	}
	// the arrival watched from the scenes: back to the list, the level does not start
	g, starts = arrFlow(true)
	g.playScene(sceneArrival, ModeScenes)
	if g.end == nil || g.end.stage != endArrival || !g.sceneReplay {
		t.Fatal("playScene(sceneArrival) must start the arrival as a replay")
	}
	g.finishArrival(g.end)
	if *starts != 0 || g.mode != ModeScenes {
		t.Fatalf("a replayed arrival must return to the scenes (mode %v, starts %d)", g.mode, *starts)
	}
}

// the helicopter blows up exactly once, after the players have got out, and nothing is left flying for ever
func TestArrivalExplodesOnceAfterExit(t *testing.T) {
	a := &arrivalState{}
	sounds, done := runArrival(a, arrTotal+1)
	if !done {
		t.Fatal("never ended")
	}
	booms := 0
	for _, s := range sounds {
		if s == "boom" {
			booms++
		}
	}
	if booms != 1 || !a.boomed {
		t.Fatalf("the helicopter must blow up once, got %d", booms)
	}
	if arrBoom <= arrExit2+0.45+0.95 {
		t.Fatalf("the blast (%.1f) comes before the second player is clear (%.1f)", arrBoom, arrExit2+0.45+0.95)
	}
	if arrTotal < 6 || arrTotal > 10 {
		t.Fatalf("the scene lasts %.1f s: 6 to 10 s wanted", arrTotal)
	}
	for _, i := range []int{0, 1} { // both are on the ground, away from the helicopter, when it blows
		ac := arrActorAt(i, 2, arrBoom-0.01)
		if !ac.vis || ac.clip {
			t.Fatalf("player %d is not out of the door before the blast", i)
		}
		if ac.y < arrGroundY+80 || ac.x > arrCX-60 && ac.x < arrCX+60 {
			t.Fatalf("player %d too close to the helicopter when it blows: %.0f,%.0f", i, ac.x, ac.y)
		}
	}
	if arrRPM(arrBoom) != 0 || arrWind(arrBoom+1) != 0 {
		t.Fatal("rotor and wind must stop with the blast")
	}
}

// players: one, or two; each comes out of the door, they never share a place and the second one is in a hurry
func TestArrivalOnePlayerOrTwoAppear(t *testing.T) {
	for _, n := range []int{1, 2} {
		for i := 0; i < n; i++ {
			if arrActorAt(i, n, 1).vis {
				t.Fatalf("n=%d player %d is out in the air", n, i)
			}
			if ac := arrActorAt(i, n, arrDoor+0.3); !ac.vis || !ac.clip {
				t.Fatalf("n=%d player %d must be in the doorway when it opens", n, i)
			}
			ac := arrActorAt(i, n, arrBoom-0.01)
			if !ac.vis || ac.clip || ac.y < arrCY+90 {
				t.Fatalf("n=%d player %d must be on the ground at the end", n, i)
			}
		}
	}
	a, b := arrActorAt(0, 2, arrBoom-0.01), arrActorAt(1, 2, arrBoom-0.01)
	if b.x-a.x < 80 {
		t.Fatalf("the two players are on top of each other: %.0f and %.0f", a.x, b.x)
	}
	if arrActorAt(0, 2, 5.0).anim != AnimWalk || arrActorAt(1, 2, 5.0).anim != AnimRun {
		t.Fatal("the blue one must walk out calmly and the red one run")
	}
	if !arrActorAt(1, 2, arrBoom-0.01).dizzy {
		t.Fatal("the red one must be dizzy")
	}
	if g := arrActorAt(1, 2, arrBoom+0.1); g.soot <= 0 {
		t.Fatal("the blast must leave them sooty")
	}
}

// the jokes: at most 48 characters, everybody has one before and after the blast, nobody talks over another
func TestArrivalLines(t *testing.T) {
	for _, n := range []int{1, 2} {
		ls := arrLines(n)
		after := map[int]bool{}
		for i, l := range ls {
			if utf8.RuneCountInString(l.text) > 48 || strings.TrimSpace(l.text) == "" {
				t.Errorf("bad line %q", l.text)
			}
			if l.who >= n {
				t.Errorf("n=%d: a line of player %d", n, l.who+1)
			}
			if l.from >= arrBoom {
				after[l.who] = true
			}
			if l.from < 0 || l.to > arrTotal+0.01 || l.to <= l.from {
				t.Errorf("bad times of %q", l.text)
			}
			for _, m := range ls[i+1:] {
				if l.from < m.to && m.from < l.to {
					t.Errorf("n=%d: %q and %q overlap in time", n, l.text, m.text)
				}
			}
			if tr(l.text) == "" {
				t.Errorf("no translation of %q", l.text)
			}
		}
		for w := 0; w < n; w++ {
			if !after[w] {
				t.Errorf("n=%d: player %d says nothing after the blast", n, w+1)
			}
		}
	}
	uiLang = "en"
	defer func() { uiLang = "es" }()
	for _, l := range arrLines(2) {
		if tr(l.text) == l.text {
			t.Errorf("%q is not translated into English", l.text)
		}
	}
}

// two presses skip (like the other cinematics); one does not
func TestArrivalSkipNeedsTwoPresses(t *testing.T) {
	a := &arrivalState{}
	if _, done := runArrival(a, 3, 1); done {
		t.Fatal("one press must not skip the arrival")
	}
	b := &arrivalState{}
	if _, done := runArrival(b, 3, 1, 1.5); !done {
		t.Fatal("two presses must skip the arrival")
	}
	c := &arrivalState{}
	if _, done := runArrival(c, 8, 1, 4.5); done || c.t < 7.9 {
		t.Fatal("two presses far apart must not skip")
	}
}

// the story: the scene is in the list, between the letter and the secret ending, with its own funny lock
func TestArrivalSceneEntry(t *testing.T) {
	if sceneArrival != sceneInvite+1 || sceneEnd2 != sceneArrival+1 {
		t.Fatal("the arrival goes after the letter and before the secret ending")
	}
	s := sceneList[sceneArrival]
	if s.key != "seen_arrival" || s.lockTitle == "" || s.hint == "" || s.title == "" {
		t.Fatalf("incomplete scene entry: %+v", s)
	}
	for i, o := range sceneList {
		if i != sceneArrival && (o.lockTitle == s.lockTitle || o.hint == s.hint) {
			t.Fatalf("the locked label of the arrival repeats scene %d", i)
		}
	}
	g := testGame()
	debugSnap = nil
	if g.sceneUnlocked(sceneArrival) {
		t.Fatal("open before being seen")
	}
	g.markSeen("seen_arrival")
	if !g.sceneUnlocked(sceneArrival) {
		t.Fatal("seen but locked")
	}
}

// the island starts with the players somewhere free: no prop, wall or fence on top of the start
func TestArrivalLevelStartIsFree(t *testing.T) {
	fitAllFootprints()
	def := jurassicLevel()
	l := buildLevel(def)
	// where the game puts them (twoplayers.go): the nearest free spot to each side of the start
	a1 := l.NearestFree(Vec{def.Start.X - playerGap/2, def.Start.Y})
	b1 := l.NearestFree(Vec{def.Start.X + playerGap/2, def.Start.Y})
	if !l.Free(def.Start) {
		t.Errorf("the start %v is blocked", def.Start)
	}
	if dist(a1, b1) < 14 {
		t.Errorf("the two players start on top of each other: %v %v", a1, b1)
	}
	for _, p := range []Vec{def.Start, a1, b1} {
		if !l.Free(p) {
			t.Errorf("the spot %v is blocked", p)
		}
		if dist(p, def.Start) > 40 {
			t.Errorf("the spot %v is far from the start %v", p, def.Start)
		}
		for _, pr := range l.Props {
			f := pr.Foot
			if p.X > f.X && p.X < f.X+f.W && p.Y > f.Y && p.Y < f.Y+f.H {
				t.Errorf("%s stands on the start %v", pr.Name, p)
			}
		}
		if l.Path(p, Vec{478, 440}) == nil || l.Path(p, Vec{640, 150}) == nil {
			t.Errorf("no way out of the start %v", p)
		}
	}
	for _, pr := range l.Props { // the helicopter leaves nothing behind: no heli prop is added by the arrival
		if strings.Contains(pr.Name, "heli") && pr.Name != "helipad" {
			t.Errorf("unexpected helicopter object %q in the island", pr.Name)
		}
	}
}
