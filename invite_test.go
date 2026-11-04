package main

import (
	"testing"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// textGame is a game that can measure text.
func textGame() *Game {
	g := testGame()
	g.face = text.NewGoXFace(bitmapfont.Face)
	return g
}

// run the letter for secs seconds, pressing at the given times
func runInvite(iv *inviteState, secs float64, presses ...float64) (phases []int, done bool, sounds []string) {
	const dt = 1.0 / 60
	next := 0
	total := 0.0
	for total < secs {
		press := false
		if next < len(presses) && total >= presses[next] {
			press = true
			next++
		}
		s, d := iv.step(press, dt)
		sounds = append(sounds, s...)
		if len(phases) == 0 || phases[len(phases)-1] != iv.phase {
			phases = append(phases, iv.phase)
		}
		if d {
			return phases, true, sounds
		}
		total += dt
	}
	return phases, false, sounds
}

// nothing moves on by itself: the closed envelope and the open letter wait for the player for ever
func TestInviteNeverAdvancesAlone(t *testing.T) {
	iv := &inviteState{}
	phases, done, _ := runInvite(iv, 600) // ten minutes without touching anything
	if done || iv.phase != invWait {
		t.Fatalf("the closed envelope moved on by itself: phase %d done %v (%v)", iv.phase, done, phases)
	}
	// a press opens it; then it waits again, with the text on show
	phases, done, _ = runInvite(iv, 600, 0)
	if done || iv.phase != invRead {
		t.Fatalf("the open letter moved on by itself: phase %d done %v (%v)", iv.phase, done, phases)
	}
}

// the order is: wait, open, read, count down, explode, and only then the island
func TestInviteOrderAndExplosionBeforeLevel(t *testing.T) {
	iv := &inviteState{}
	runInvite(iv, 5) // arrives, waits
	_, done, sounds := runInvite(iv, 30, 0, 3, 6)
	if !done {
		t.Fatal("the letter never finished after the three presses")
	}
	boom := -1
	for i, s := range sounds {
		if s == "boom" {
			boom = i
		}
	}
	if boom < 0 {
		t.Fatal("no explosion before the level")
	}
	beeps := 0
	for _, s := range sounds[:boom] {
		if s == "alert" {
			beeps++
		}
	}
	if beeps != invSteps {
		t.Fatalf("the countdown beeped %d times, want %d (5-4-3-2-1)", beeps, invSteps)
	}
}

// a double press by accident (within 0.3 s) does not skip a phase
func TestInviteDebounce(t *testing.T) {
	iv := &inviteState{}
	runInvite(iv, 5)
	runInvite(iv, 0.1, 0, 0.05) // two presses in a row: only the first counts
	if iv.phase != invOpen {
		t.Fatalf("after a double press it should be opening, phase %d", iv.phase)
	}
	// pressing while it opens hurries that phase to its end, but not the next one
	iv2 := &inviteState{phase: invOpen}
	runInvite(iv2, 0.05, 0)
	if iv2.phase == invCount || iv2.phase == invBoom {
		t.Fatalf("a press while opening skipped the reading: phase %d", iv2.phase)
	}
}

// the countdown only starts with a press after the whole letter is on show
func TestInviteDeviceInactiveWhileReading(t *testing.T) {
	iv := &inviteState{phase: invRead}
	runInvite(iv, 120)
	if iv.phase != invRead {
		t.Fatalf("the device started counting alone: phase %d", iv.phase)
	}
}

// the text of the letter fits on the paper at a whole text scale
func TestInviteTextFits(t *testing.T) {
	g := textGame()
	sc := g.inviteScale()
	if sc != 1 && sc != 2 {
		t.Fatalf("the text scale is not a whole number: %v", sc)
	}
	lines, h := g.inviteLayout(sc)
	if h > invPH-20 {
		t.Fatalf("the text is %v high and the paper %v", h, invPH)
	}
	for _, l := range lines {
		if l.x < 0 || l.x+g.textW(l.s, l.scale) > invPW {
			t.Fatalf("line %q goes out of the paper", l.s)
		}
		if l.x != float64(int(l.x)) || l.y != float64(int(l.y)) {
			t.Fatalf("line %q is not on whole pixels", l.s)
		}
	}
}

// the island appears in the list only after the invitation has unlocked it
func TestIslandListedOnlyWhenUnlocked(t *testing.T) {
	old := progressFrozen
	progressFrozen = true // never write the real progress
	defer func() { progressFrozen = old }()
	g := &Game{}
	if g.jpListed() {
		t.Fatal("the island is listed before it is unlocked")
	}
	g.unlockJP()
	if !g.jpListed() {
		t.Fatal("the island is not listed after the invitation")
	}
	// a new game keeps it
	var fresh Progress
	keepSeen(&fresh, g.progress)
	if fresh.Stats["jp_unlocked"] == 0 {
		t.Fatal("a new game forgot that the island was unlocked")
	}
}

// the two endings are different pieces and both end the same way: bed, then silence
func TestEndingsAreDifferent(t *testing.T) {
	order := func(secret bool) []int {
		var o []int
		for s := endWalk; s != endFinal; s = nextEndStage(s, secret) {
			o = append(o, s)
		}
		return append(o, endFinal)
	}
	n, s := order(false), order(true)
	if len(s) != len(n)+1 {
		t.Fatalf("the secret ending should have one more scene (the stinger): %v vs %v", s, n)
	}
	for _, o := range [][]int{n, s} {
		hasBed := false
		for _, st := range o {
			if st == endBed {
				hasBed = true
			}
		}
		if !hasBed || o[len(o)-1] != endFinal {
			t.Fatalf("an ending without the bed or the end card: %v", o)
		}
	}
	if endLenFor(endWalk, true) <= endLenFor(endWalk, false) || endLenFor(endCredits, true) <= endLenFor(endCredits, false) {
		t.Fatal("the secret ending is not longer")
	}
	nr, sr := endCreditRows(false), endCreditRows(true)
	if len(sr) <= len(nr) {
		t.Fatal("the secret credits are not longer than the normal ones")
	}
	g := textGame()
	for _, tt := range []float64{1, 7, 12, 20} {
		a := g.walkCaption(&endState{stage: endWalk, t: tt})
		b := g.walkCaption(&endState{stage: endWalk, t: tt, trueEnd: true})
		if a == b {
			t.Fatalf("the same caption at %vs in both endings: %q", tt, a)
		}
	}
	if g.bedCaption(&endState{stage: endBed, t: 12}) == g.bedCaption(&endState{stage: endBed, t: 12, trueEnd: true}) {
		t.Fatal("the same bed caption in both endings")
	}
	// the music dies away in the bed (the silence) in both
	for _, secret := range []bool{false, true} {
		end := endLenFor(endBed, secret)
		if v := endMusicVol(&endState{stage: endBed, t: end - 0.5, trueEnd: secret}); v > 0.02 {
			t.Fatalf("the bed is not silent at its end (secret %v): %v", secret, v)
		}
	}
}

// the scenes list follows the story: intro, interludes 1 to 3, normal ending, letter, secret ending
func TestScenesInStoryOrder(t *testing.T) {
	want := []string{"seen_intro", "seen_interlude5", "seen_interlude10", "seen_interlude20", "seen_end1", "seen_invite", "seen_end2"}
	if sceneCount != len(want) {
		t.Fatalf("%d scenes, want %d", sceneCount, len(want))
	}
	for i, k := range want {
		if sceneList[i].key != k {
			t.Errorf("scene %d is %q, want %q", i+1, sceneList[i].key, k)
		}
	}
	if sceneIntro != 0 || sceneInter1 != 1 || sceneInter2 != 2 || sceneInter3 != 3 || sceneEnd1 != 4 || sceneInvite != 5 || sceneEnd2 != 6 {
		t.Fatal("the constants of the scenes are not in story order")
	}
}
