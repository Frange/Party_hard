package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// preGame: a game that can run the story of the endings without touching anything real.
func preGame(t *testing.T, two bool) *Game {
	old, oldTwo, oldStart := progressFrozen, twoPlayerMode, testStartLevel
	t.Cleanup(func() { progressFrozen, twoPlayerMode, testStartLevel = old, oldTwo, oldStart })
	g := firstRunGame()
	g.progress = Progress{Unlocked: campaignN, Stats: map[string]int{}}
	twoPlayerMode = two
	testStartLevel = func(g *Game, i int) { g.levelIdx, g.mode = i, ModePlay }
	return g
}

// runStages plays the ending at 60 Hz until it reaches a stage in stops (or maxSecs pass); it returns the stages
// it went through and the seconds spent in each.
func runStages(g *Game, maxSecs float64, stops ...int) (order []int, secs map[int]float64) {
	secs = map[int]float64{}
	for k := 0; float64(k) < maxSecs*60 && g.end != nil; k++ {
		e := g.end
		if len(order) == 0 || order[len(order)-1] != e.stage {
			order = append(order, e.stage)
		}
		for _, s := range stops {
			if e.stage == s {
				return
			}
		}
		secs[e.stage] += 1.0 / 60
		g.updateEnding()
	}
	return
}

func sameStages(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The action of the last scene comes BEFORE each ending and before the letter, and each one leads where it must.
func TestPreludePlaysBeforeEachEnding(t *testing.T) {
	for _, two := range []bool{false, true} {
		// the road without every trophy: action, then a short epilogue
		g := preGame(t, two)
		g.startEndingPre(false)
		if g.end.stage != endPrelude || g.end.x.pre.kind != preBad || g.progress.Stats["seen_end1"] != 1 {
			t.Fatalf("2P=%v: the normal ending must start with its prelude (stage %d)", two, g.end.stage)
		}
		order, secs := runStages(g, 60, endFinal)
		if !sameStages(order, []int{endPrelude, endWalk, endDoor, endFinal}) {
			t.Fatalf("2P=%v normal ending: stages %v", two, order)
		}
		if ep := secs[endWalk] + secs[endDoor]; ep > 15 || ep < 6 {
			t.Errorf("2P=%v: the epilogue of the normal ending lasts %.1fs (must be short: 6..15)", two, ep)
		}
		if secs[endPrelude] < preBadLen-0.1 {
			t.Errorf("prelude cut short: %.1fs", secs[endPrelude])
		}
		// the road with every trophy: action, then the letter exactly as it was
		g = preGame(t, two)
		g.startInvitePre()
		if g.end.stage != endPrelude || g.end.x.pre.kind != preLetter || g.progress.Stats["seen_invite"] != 1 {
			t.Fatalf("2P=%v: the letter must start with its prelude", two)
		}
		order, _ = runStages(g, 60, endInvite)
		if !sameStages(order, []int{endPrelude, endInvite}) {
			t.Fatalf("2P=%v letter: stages %v", two, order)
		}
		if g.end.x.inv.phase != invArrive || g.end.x.inv.t > 0.1 {
			t.Errorf("the letter must begin at its beginning after the prelude: phase %d t %.2f", g.end.x.inv.phase, g.end.x.inv.t)
		}
		// the island: action, then the party, the credits, the scene after them and the end card
		g = preGame(t, two)
		g.startEndingPre(true)
		if g.end.stage != endPrelude || g.end.x.pre.kind != preIsland || g.progress.Stats["seen_end2"] != 1 {
			t.Fatalf("2P=%v: the secret ending must start with its prelude", two)
		}
		order, secs = runStages(g, 260, endFinal)
		if !sameStages(order, []int{endPrelude, endParty, endCredits, endStinger, endPaper, endHome, endFinal}) {
			t.Fatalf("2P=%v secret ending: stages %v", two, order)
		}
		if f := secs[endParty] + secs[endCredits]; f < 35 {
			t.Errorf("2P=%v: the good ending (party + credits) lasts only %.1fs, it must be long (>=35)", two, f)
		}
	}
}

// Skipping the prelude (Enter or Esc) goes on with what comes next; it does not skip the whole ending.
func TestPreludeSkipGoesOnToTheEnding(t *testing.T) {
	for _, c := range []struct {
		name  string
		start func(*Game)
		want  int
	}{
		{"normal", func(g *Game) { g.startEndingPre(false) }, endWalk},
		{"letter", func(g *Game) { g.startInvitePre() }, endInvite},
		{"secret", func(g *Game) { g.startEndingPre(true) }, endParty},
	} {
		g := preGame(t, false)
		c.start(g)
		g.end.t = 1
		g.endLeavePrelude(g.end) // what Esc does in a prelude
		if g.end.stage != c.want || g.end.t != 0 {
			t.Errorf("%s: after the prelude comes stage %d, got %d", c.name, c.want, g.end.stage)
		}
	}
}

// The story entries (list of scenes, DEBUG) use the preludes too, and only the story's keys are recorded.
func TestScenesReplayIncludesPrelude(t *testing.T) {
	for i, kind := range map[int]int{sceneEnd1: preBad, sceneInvite: preLetter, sceneEnd2: preIsland} {
		g := preGame(t, false)
		g.progress.Stats = map[string]int{}
		g.playScene(i, ModeScenes)
		if g.end == nil || g.end.stage != endPrelude || g.end.x.pre.kind != kind || !g.sceneReplay {
			t.Errorf("scene %d: a replay must start with the prelude of kind %d", i, kind)
		}
	}
}

// Each prelude and each ending is its own thing: different captions, different music, different pace.
func TestEndingsAreDifferent(t *testing.T) {
	if preBadLen > 15 || preludeLen(preLetter) < 10 || preludeLen(preIsland) < 10 {
		t.Fatal("the preludes are the action: not too short (>=10 s) and the bad one is not long either")
	}
	bad := endLenFor(endWalk, false) + endLenFor(endDoor, false)
	if bad > 15 {
		t.Fatalf("the normal ending lasts %.1f s: it is a short epilogue (<= 15 s)", bad)
	}
	if good := endLenFor(endParty, true) + endLenFor(endCredits, true); good < 35 || good < 3*bad {
		t.Fatalf("the good ending lasts %.1f s: it has to be long (>= 35 s and far longer than the normal one)", good)
	}
	if nextEndStage(endDoor, false) != endFinal || nextEndStage(endPrelude, true) != endParty || nextEndStage(endParty, true) != endCredits {
		t.Fatal("the stages of the endings are not chained as designed")
	}
	g := textGame()
	for _, tt := range []float64{1, 4, 7} {
		if g.walkCaption(&endState{stage: endWalk, t: tt}) == g.walkCaption(&endState{stage: endWalk, t: tt, trueEnd: true}) {
			t.Errorf("same caption at %.0fs in both endings", tt)
		}
	}
	for _, tt := range []float64{2, 5, 9} {
		if preBadCaption(tt) == preLetterCaption(tt) || preLetterCaption(tt) == preIslandCaption(tt) || preBadCaption(tt) == preIslandCaption(tt) {
			t.Errorf("two preludes say the same at %.0fs", tt)
		}
	}
	// the tunes: the bad one is quiet, the good one is the fanfare; the party is louder than anything of the bad one
	nt, _ := endTune(&endState{stage: endWalk})
	gt, _ := endTune(&endState{stage: endParty, trueEnd: true})
	if nt == gt || gt != ambSecret {
		t.Errorf("the endings share their music: %q vs %q", nt, gt)
	}
	if endMusicVol(&endState{stage: endParty, t: 10, trueEnd: true}) <= endMusicVol(&endState{stage: endWalk, t: 5}) {
		t.Error("the party must be louder than the quiet walk")
	}
	// the three preludes put the actors in different places and poses
	same := 0
	for tt := 0.0; tt < 12; tt += 0.25 {
		a1, _, _ := preActors(preLetter, tt, false)
		b1, _, _ := preActors(preBad, tt, false)
		c1, _, _ := preActors(preIsland, tt, false)
		if a1 == b1 || b1 == c1 || a1 == c1 {
			same++
		}
	}
	if same > 3 {
		t.Errorf("the heroes do the same in two preludes at %d moments", same)
	}
}

// every balloon and caption of the new pieces fits a balloon (<= 48 characters) and is translated.
func TestPreludeTextsFit(t *testing.T) {
	defer func(l string) { uiLang = l }(uiLang)
	var all []string
	for _, ls := range [][]preLine{preLetterLines, preBadLines, preIslandLines} {
		for _, l := range ls {
			all = append(all, l.s)
			if l.b <= l.a {
				t.Errorf("balloon %q has no time", l.s)
			}
		}
	}
	for _, l := range partyLines {
		all = append(all, l.s)
	}
	for _, s := range []string{preAlarm(1), preAlarm(4), "Foto de familia", "Nadie fue devorado*", "*Casi nadie.", "¡FELIZ FIESTA!", "¡SORPRESA!",
		"Aterrizaje de emergencia. En su calle, casi.", "Los fantasmas de las fiestas le siguen, flotando y sin pagar.",
		"Los fantasmas de las fiestas les siguen. El azul saluda; el rojo huye."} {
		all = append(all, s)
	}
	for tt := 0.0; tt < 40; tt += 0.5 {
		all = append(all, preLetterCaption(tt), preBadCaption(tt), preIslandCaption(tt), partyCaption(tt))
	}
	for _, r := range endCreditRows(true) {
		all = append(all, r[0], r[1])
	}
	for _, s := range all {
		if n := utf8.RuneCountInString(s); n > 72 && !strings.Contains(s, "  ") { // captions of the narrator may be longer; balloons not
			t.Errorf("%q has %d characters", s, n)
		}
	}
	for _, ls := range [][]preLine{preLetterLines, preBadLines, preIslandLines} {
		for _, l := range ls {
			if n := utf8.RuneCountInString(l.s); n > 48 {
				t.Errorf("balloon %q has %d characters (max 48)", l.s, n)
			}
		}
	}
	for _, l := range partyLines {
		if n := utf8.RuneCountInString(l.s); n > 48 {
			t.Errorf("balloon %q has %d characters (max 48)", l.s, n)
		}
	}
	uiLang = "en"
	done := map[string]bool{"Frange": true, "2026": true, "Guybrush Treepwood": true, "Diego": true, "César": true, "Samuel": true, "Director": true, "Testers": true, "Fiesta Nocturna": true, "PARTY HARDY": true} // names and numbers
	for _, s := range all {
		if s == "" || done[s] {
			continue
		}
		done[s] = true
		if tr(s) == s && !strings.HasPrefix(s, "RAWR") { // (the roar of the dinosaur is the same in both languages)
			t.Errorf("no English text for %q", s)
		}
	}
}

// Every piece draws without panicking with 1 and 2 players; the backgrounds are painted before the people.
func TestPreludesAndPartyLayerOrderAndNoPanic(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := layerGame(t, two)
		dst := ebiten.NewImage(screenW, screenH)
		for _, c := range []struct {
			name  string
			start func()
			stage int
			times []float64
		}{
			{"letter", func() { g.startInvitePre() }, endPrelude, []float64{0.1, 3.5, 8, 10.5, 13.5, 16.8}},
			{"bad", func() { g.startEndingPre(false) }, endPrelude, []float64{0.1, 2, 5, 7.9, 9, 11, 12.5}},
			{"island", func() { g.startEndingPre(true) }, endPrelude, []float64{0.1, 3.4, 6.5, 9, 10.5, 13, 15.4}},
			{"party", func() { g.startEndingPre(true) }, endParty, []float64{0.5, 5, 8.5, 12, 15, 19.9, 22, 26.8, 29, 31.3, 33, 35.9}},
		} {
			c.start()
			e := g.end
			e.stage = c.stage
			for _, tt := range c.times {
				e.t, e.total = tt, tt
				var rec []string
				layerRec = &rec
				g.drawEnding(dst)
				layerRec = nil
				firstActor, bgs := -1, 0
				for i, l := range rec {
					if strings.HasPrefix(l, "actor:") && firstActor < 0 {
						firstActor = i
					}
					if strings.HasPrefix(l, "bg:") {
						bgs++
						if firstActor >= 0 {
							t.Errorf("2P=%v %s t=%.1f: %q is drawn after an actor (%q)", two, c.name, tt, l, rec[firstActor])
						}
					}
				}
				if firstActor < 0 || bgs == 0 {
					t.Errorf("2P=%v %s t=%.1f: no background or no actor was drawn: %v", two, c.name, tt, rec)
				}
			}
		}
		// the scene after the credits and the credits of the party
		g.startEndingPre(true)
		for _, st := range []int{endCredits, endStinger, endFinal} {
			g.end.stage = st
			for _, tt := range []float64{0.5, 3, 7} {
				g.end.t, g.end.total = tt, tt
				g.drawEnding(dst)
			}
		}
		dst.Deallocate()
	}
}

// The people of the party move without teleporting in front of the player's eyes.
func TestPartyCrowdMovesSmoothly(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := layerGame(t, two)
		g.startEndingPre(true)
		g.partyBuild(g.end)
		p := &g.end.x.party
		if len(p.cast) < 6 {
			t.Fatalf("only %d guests at the party", len(p.cast))
		}
		prev := g.partyCrowd(p, 0, two)
		if want := len(p.cast) + 1 + bi(two); len(prev) != want {
			t.Fatalf("2P=%v: %d people, want %d", two, len(prev), want)
		}
		for tt := 1.0 / 60; tt < partyLen; tt += 1.0 / 60 {
			cur := g.partyCrowd(p, tt, two)
			for i := range cur {
				m, q := cur[i], prev[i]
				if m.x < -400 || m.x > endCW+400 || m.y < 150 || m.y > 270 {
					t.Fatalf("%s at %.2fs is off the island: %.0f,%.0f", m.name, tt, m.x, m.y)
				}
				if dx := m.x - q.x; dx > 12 || dx < -12 {
					off := func(x float64) bool { return x < -24 || x > endCW+24 }
					if !(off(m.x) && off(q.x)) && m.alpha > 0.2 && q.alpha > 0 {
						t.Fatalf("%s jumps %.0f px at %.2fs while on screen (%.0f -> %.0f)", m.name, dx, tt, q.x, m.x)
					}
				}
			}
			prev = cur
		}
	}
}

// The conga: during its time the people really cross the whole island from one side to the other.
func TestPartyCongaCrossesTheIsland(t *testing.T) {
	g := layerGame(t, false)
	g.startEndingPre(true)
	g.partyBuild(g.end)
	p := &g.end.x.party
	seenLeft := map[int]bool{}
	seenRight := map[int]bool{}
	for tt := partyActConga; tt < partyActNight; tt += 0.05 {
		for i, m := range g.partyCrowd(p, tt, false) {
			if m.x > 20 && m.x < 80 && m.dir == DirRight {
				seenLeft[i] = true
			}
			if m.x > 400 && m.x < 460 && m.dir == DirRight {
				seenRight[i] = true
			}
		}
	}
	n := len(p.cast) + 1
	if len(seenLeft) != n || len(seenRight) != n {
		t.Fatalf("only %d/%d people enter and %d/%d leave in the conga", len(seenLeft), n, len(seenRight), n)
	}
}

// The credits of the party have time to roll all of them in.
func TestPartyCreditsFitTheirTime(t *testing.T) {
	rows := endCreditRows(true)
	h := 0.0
	for _, r := range rows {
		switch {
		case r[0] == "":
			h += 28
		case r[1] == "Fiesta Nocturna":
			h += 98
		default:
			h += 38
		}
	}
	reach := endCreditSpeed(true) * endLenFor(endCredits, true)
	if 560+h-reach > 460 {
		t.Errorf("the last row of the credits is still at y=%.0f when the credits end (rows %.0f px, scroll %.0f px)", 560+h-reach, h, reach)
	}
}

// with the progress frozen the whole story of the endings writes nothing to assets/progress.json.
func TestEndingsWithFrozenProgressWriteNothing(t *testing.T) {
	oldFrozen, oldNo := progressFrozen, noAssets
	wd, _ := os.Getwd()
	t.Cleanup(func() { progressFrozen, noAssets = oldFrozen, oldNo; os.Chdir(wd) })
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0o755)
	os.Chdir(dir)
	noAssets = false
	progressFrozen = true
	for _, start := range []func(*Game){func(g *Game) { g.startEndingPre(false) }, func(g *Game) { g.startEndingPre(true) }, func(g *Game) { g.startInvitePre() }} {
		g := firstRunGame()
		progressFrozen = true
		g.progress = Progress{Unlocked: campaignN, Stats: map[string]int{}}
		start(g)
		runStages(g, 25, endWalk, endParty, endInvite)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "progress.json")); err == nil {
		t.Fatal("the endings wrote a progress.json while frozen")
	}
}
