package main

import (
	"fmt"
	"testing"
)

func testGame() *Game {
	progressFrozen = true // nothing in these tests may touch the real progress.json
	return &Game{progress: Progress{Unlocked: 1}}
}

func TestScenesLockedUntilSeen(t *testing.T) {
	g := testGame()
	debugSnap = nil
	// a brand new game, even with every trophy: nothing but the intro-for-players-that-played is open
	g.progress.Trophies = map[string]bool{}
	for _, tr := range trophyList {
		g.progress.Trophies[tr.id] = true
	}
	for i := 0; i < sceneCount; i++ {
		if g.sceneUnlocked(i) {
			t.Errorf("scene %d open without having been seen", i)
		}
		if g.sceneName(i) != sceneList[i].lockTitle || g.sceneText(i) != sceneList[i].hint {
			t.Errorf("scene %d shows its secrets while locked", i)
		}
	}
	// locked labels are all different, funny and have a hint
	seen := map[string]bool{}
	for i, s := range sceneList {
		if s.lockTitle == "" || s.hint == "" || seen[s.lockTitle] {
			t.Errorf("scene %d: bad locked label %q / hint %q", i, s.lockTitle, s.hint)
		}
		seen[s.lockTitle] = true
	}
	if sceneList[sceneEnd1].lockTitle != "¿ FINAL ?" || sceneList[sceneEnd2].lockTitle != "FINAL: ???" {
		t.Error("the endings must read ¿ FINAL ? and FINAL: ???")
	}
	// seeing one for real opens it, and shows the real title
	g.markSeen("seen_end2")
	if !g.sceneUnlocked(sceneEnd2) || g.sceneName(sceneEnd2) != "FINAL SECRETO" {
		t.Error("the secret ending stays locked after being seen")
	}
	if g.sceneUnlocked(sceneInvite) || g.sceneUnlocked(sceneEnd1) {
		t.Error("seeing one scene opened another")
	}
}

func TestScenesDebugOpensEverything(t *testing.T) {
	g := testGame()
	debugSnap = &Progress{}
	defer func() { debugSnap = nil }()
	for i := 0; i < sceneCount; i++ {
		if !g.sceneUnlocked(i) {
			t.Errorf("scene %d locked in DEBUG", i)
		}
	}
}

func TestIntroForPlayersWhoAlreadyPlayed(t *testing.T) {
	g := testGame()
	debugSnap = nil
	if g.sceneUnlocked(sceneIntro) {
		t.Fatal("a new player must not have the intro unlocked")
	}
	g.progress.Unlocked = 4 // an old save, from before the intro existed
	if !g.sceneUnlocked(sceneIntro) {
		t.Error("somebody who already played should be able to see the intro")
	}
}

func TestOldProgressWithoutSeenKeys(t *testing.T) {
	g := testGame()
	g.progress = Progress{Unlocked: 3} // Stats nil, like an old progress.json
	if g.seen("seen_intro") {
		t.Fatal("nothing seen in an old progress")
	}
	g.markSeen("seen_intro") // must not panic on a nil map
	if !g.seen("seen_intro") {
		t.Fatal("the mark is lost")
	}
}

func TestNewGameKeepsSeenMarks(t *testing.T) {
	old := Progress{Stats: map[string]int{"seen_intro": 1, "seen_end1": 1, "kills": 99}}
	n := Progress{Unlocked: 1}
	keepSeen(&n, old)
	if n.Stats["seen_intro"] != 1 || n.Stats["seen_end1"] != 1 || n.Stats["kills"] != 0 {
		t.Errorf("stats after a reset: %v", n.Stats)
	}
}

func TestIntroShownOnlyTheFirstTime(t *testing.T) {
	g := testGame()
	if g.seen("seen_intro") {
		t.Fatal("seen at the start")
	}
	g.markSeen("seen_intro") // startIntro does this
	if g.seen("seen_intro") == false {
		t.Fatal("not marked")
	}
	// beginNewGame chooses by this very flag
	g2 := testGame()
	g2.progress.Stats = map[string]int{"seen_intro": 1}
	if !g2.seen("seen_intro") {
		t.Fatal("a seen intro must not show again")
	}
}

func TestInterludesAfterLevels5And10(t *testing.T) {
	g := testGame()
	if k, ok := g.interludeAfter(4); !ok || k != interCar {
		t.Error("after the bull ring there must be the car interlude")
	}
	if k, ok := g.interludeAfter(9); !ok || k != interBoat {
		t.Error("after the disco there must be the boat interlude")
	}
	for _, i := range []int{0, 3, 5, 8, 10, 20} {
		if _, ok := g.interludeAfter(i); ok {
			t.Errorf("level %d should not have an interlude", i)
		}
	}
	g.markSeen("seen_interlude5")
	g.markSeen("seen_interlude10")
	if _, ok := g.interludeAfter(4); ok {
		t.Error("the first interlude shows again")
	}
	if _, ok := g.interludeAfter(9); ok {
		t.Error("the second interlude shows again")
	}
}

func TestInterludeTimeline(t *testing.T) {
	if len(interludePlaces(interCar)) != 5 || len(interludePlaces(interBoat)) != 0 || len(interludePlaces(interBalloon)) != 0 {
		t.Fatal("five memories in the car; the boat and the balloon have their own scenes")
	}
	// the street never goes backwards and the vehicle is at its place when it stops
	prev := -1.0
	for x := 0.0; x <= interLen; x += 0.05 {
		s := interScroll(x)
		if s < prev-1e-9 {
			t.Fatalf("the street goes backwards at %.2f", x)
		}
		prev = s
	}
	if d := interVehicleX(itStop) - interStopX; d > 0.001 || d < -0.001 {
		t.Errorf("the vehicle is not at its place when the street stops: %v", d)
	}
	if interLen > 20 || interLen < 10 {
		t.Errorf("the car lasts %.1f s: between 10 and 20", interLen)
	}
	if interVehicleX(0) < screenW {
		t.Error("the vehicle must start out of the screen, far to the right")
	}
	if !(itDecel0 < itStop && itStop < itBoard0 && itBoard0 < itOpen && itOpen < itIn && itIn < itDepart && itDepart < itFadeOut && itFadeOut < interLen) {
		t.Error("interlude timeline out of order")
	}
}

func TestIntroTimelineAndProportions(t *testing.T) {
	if !(inGlassSlide < inGlassFall && inGlassFall < inGlassLand && inGlassLand < inSheepStart && inSheepStart < inBurst &&
		inBurst < inStandUp && inStandUp < inKnife && inKnife < inWalk0 && inWalk0 < inWalk1 && inWalk1 < inFadeOut && inFadeOut < inCard && inCard < introLen) {
		t.Fatal("intro timeline out of order")
	}
	// the door is about 1.35 times as tall as the hero is at that depth
	_, _, s := introHeroPos(inWalk1)
	heroH := 32 * s // the visible height of the hero sprite
	if r := doorH / heroH; r < 1.3 || r > 1.45 {
		t.Errorf("the door is %.2f times the hero (want 1.3-1.4)", r)
	}
	// the door is closed at the start, opens when he arrives and closes again after he goes
	if doorOpen(0) != 0 || doorOpen(inWalk1+1) < 0.99 || doorOpen(inWalk1+2.4) > 0.01 {
		t.Error("door animation wrong")
	}
	// the glass breaks only once and the beat falls off
	if beatKick(0) < 0.99 || beatKick(thumpBeat-0.01) > 0.2 {
		t.Error("beat kick")
	}
}

func opaqueCount(b *pixBuf) (opaque, soft int) {
	for i := 3; i < len(b.p); i += 4 {
		switch {
		case b.p[i] == 255:
			opaque++
		case b.p[i] > 0:
			soft++
		}
	}
	return
}

// the hero of the cinematics must be solid (not a transparent body with only a knife floating about)
func TestHeroWithKnifeIsSolid(t *testing.T) {
	ca, err := loadCharAsset("assets/sprites/main_character/darius")
	if err != nil {
		t.Skip("no darius asset:", err)
	}
	base := newRigCharSet(ca)
	knife := knifeHeroSet(base)
	if knife.rig == nil || knife.rig.carry != HoldKnife || base.rig.carry == HoldKnife {
		t.Fatal("the knife must be set on a copy of the rig, not on the original")
	}
	for _, vw := range []int{vFront, vSide, vBack} {
		for _, anim := range []int{AnimIdle, AnimWalk} {
			b0, _ := opaqueCount(base.rigFrame(anim, vw, 1))
			b1, soft := opaqueCount(knife.rigFrame(anim, vw, 1))
			if b0 < 80 || b1 < b0 {
				t.Errorf("view %d anim %d: body %d px opaque, with the knife %d", vw, anim, b0, b1)
			}
			if soft != 0 {
				t.Errorf("view %d anim %d: %d semi transparent pixels", vw, anim, soft)
			}
		}
	}
	// a procedural character gets the knife through its spec, keeping its colours
	sp := playerSpec()
	proc := newCharSet(&sp)
	pk := knifeHeroSet(proc)
	if pk.Spec == proc.Spec || pk.Spec.Carry != HoldKnife || pk.Spec.Skin != sp.Skin {
		t.Error("procedural hero: knife not set on a copy of the spec")
	}
}

func TestScenesListFormat(t *testing.T) {
	g := testGame()
	debugSnap = nil
	g.progress.Stats = map[string]int{"seen_intro": 1, "seen_interlude5": 1}
	if got := g.sceneRowName(sceneIntro); got != "1.- INTRO: NADIE TE DEJÓ DORMIR" || g.sceneRowCount(sceneIntro) != "" {
		t.Errorf("intro row: %q %q", got, g.sceneRowCount(sceneIntro))
	}
	if got := g.sceneRowName(sceneInter1); got != "2.- INTERLUDIO: OTRA FIESTA MÁS" || g.sceneRowCount(sceneInter1) != fmt.Sprintf("(5/%d)", len(campaignLevels)) {
		t.Errorf("seen row: %q %q", got, g.sceneRowCount(sceneInter1))
	}
	if got := g.sceneRowName(sceneInter2); got != "3.- ¿?" || g.sceneRowCount(sceneInter2) != "" {
		t.Errorf("locked row reveals: %q %q", got, g.sceneRowCount(sceneInter2))
	}
	// every row fits between the left margin and the SEEN/LOCKED column, in both languages
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	g.progress.Stats = map[string]int{"seen_intro": 1, "seen_interlude5": 1, "seen_interlude10": 1, "seen_interlude15": 1, "seen_interlude20": 1, "seen_end1": 1, "seen_end2": 1, "seen_invite": 1, "seen_arrival": 1}
	for _, l := range []string{"es", "en"} {
		uiLang = l
		for i := 0; i < sceneCount; i++ {
			row := g.sceneRowName(i)
			if n := len([]rune(row + " " + g.sceneRowCount(i))); n > 48 {
				t.Errorf("%s row %d has %d chars", l, i, n)
			}
		}
	}
}

// the balloon is scene 4 of the list: key, counter (15/N) and name, locked until seen
func TestSceneBalloonRow(t *testing.T) {
	old := debugSnap
	debugSnap = nil
	t.Cleanup(func() { debugSnap = old })
	g := testGame()
	g.progress.Stats = map[string]int{}
	if got := g.sceneRowName(sceneInter15); got != "4.- ¿?" || g.sceneRowCount(sceneInter15) != "" {
		t.Errorf("locked balloon row: %q %q", got, g.sceneRowCount(sceneInter15))
	}
	g.progress.Stats["seen_interlude15"] = 1
	if got := g.sceneRowName(sceneInter15); got != "4.- INTERLUDIO: GLOBO A LA DERIVA" || g.sceneRowCount(sceneInter15) != fmt.Sprintf("(15/%d)", len(campaignLevels)) {
		t.Errorf("seen balloon row: %q %q", got, g.sceneRowCount(sceneInter15))
	}
}

// each interlude row replays its own interlude, as a replay (no level starts after it)
func TestScenePlaysRightInterlude(t *testing.T) {
	old := debugSnap
	debugSnap = nil
	t.Cleanup(func() { debugSnap = old })
	for _, c := range []struct{ scene, kind, next int }{
		{sceneInter1, interCar, 5}, {sceneInter2, interBoat, 10}, {sceneInter15, interBalloon, 15}, {sceneInter3, interRocket, len(campaignLevels) - 1},
	} {
		g := testGame()
		g.snd, g.as = newSounds(true), &Assets{}
		g.playScene(c.scene, ModeScenes)
		if g.inter == nil || g.inter.kind != c.kind || g.inter.next != c.next || !g.sceneReplay {
			t.Errorf("scene %d does not replay interlude kind %d", c.scene+1, c.kind)
		}
	}
}
