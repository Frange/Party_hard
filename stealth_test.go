package main

import (
	"flag"
	"strings"
	"testing"
)

func stealthGame(t *testing.T) *Game {
	t.Helper()
	oldF, oldS, oldP := progressFrozen, debugSnap, debugPrevFrozen
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen = oldF, oldS, oldP })
	g := suspectGame()
	g.policeCalled = false
	g.mode = ModePlay
	g.stealth = true
	g.p.pos = Vec{200, 300}
	return g
}

func guardAt(g *Game, at Vec, face Dir) *Guard {
	gd := &Guard{pos: at, dir: face, state: gdWait}
	g.guards = append(g.guards, gd)
	return gd
}

func runStealth(g *Game, secs float64) {
	for t := 0.0; t < secs; t += 1.0 / 60 {
		g.time += 1.0 / 60
		g.updateStealth(1.0 / 60)
	}
}

func TestStealthOffEverythingIsInert(t *testing.T) {
	g := stealthGame(t)
	g.stealth = false
	guardAt(g, Vec{260, 300}, DirLeft)
	runStealth(g, 3)
	if g.sth.phase != sthCalm || g.sth.detections != 0 || g.wanted != 0 {
		t.Error("outside the stealth mode nothing of it may act")
	}
	if g.stealthInteract(g.p) || g.stealthNoAttack() || g.stealthSpeedMul(g.p) != 1 || g.stealthBoxed(g.p) {
		t.Error("the box hooks must do nothing outside the stealth mode")
	}
}

func TestStealthSeenByAGuardIsDetectedAfterAMoment(t *testing.T) {
	g := stealthGame(t)
	guardAt(g, Vec{260, 300}, DirLeft) // looks at the player, 60 px away
	runStealth(g, 0.2)
	if g.sth.phase != sthCalm {
		t.Error("detected too fast: a glance is not enough")
	}
	runStealth(g, 0.5)
	if g.sth.phase != sthRed || g.sth.detections != 1 {
		t.Fatalf("phase %d detections %d: seen for over %.2fs and still calm", g.sth.phase, g.sth.detections, sthSeeTime)
	}
	if !g.policeCalled || g.wanted < 45 {
		t.Error("the alert must call the police and hold the search up")
	}
	if g.sth.exT <= 0 {
		t.Error("the big exclamation is not shown")
	}
}

func TestStealthBehindTheWatcherIsSafe(t *testing.T) {
	g := stealthGame(t)
	guardAt(g, Vec{260, 300}, DirRight) // looks the other way
	suspectGuest(g, Vec{140, 300}, DirLeft)
	runStealth(g, 3)
	if g.sth.detections != 0 {
		t.Error("nobody has him in front: he must not be detected")
	}
}

func TestStealthPhasesRedYellowCalm(t *testing.T) {
	g := stealthGame(t)
	g.stealthDetect()
	if g.sth.phase != sthRed {
		t.Fatal("red first")
	}
	runStealth(g, sthRedTime-1)
	if g.sth.phase != sthRed || g.wanted < 45 {
		t.Errorf("still red with the search held (phase %d wanted %.0f)", g.sth.phase, g.wanted)
	}
	runStealth(g, 1.3)
	if g.sth.phase != sthYellow {
		t.Fatalf("after %.0f s it must be yellow, phase %d", sthRedTime, g.sth.phase)
	}
	if g.wanted != 0 {
		t.Error("yellow: the chase is over, the cuffs are not held")
	}
	runStealth(g, sthYellowTime+0.3)
	if g.sth.phase != sthCalm {
		t.Error("after the yellow phase it goes calm")
	}
	if g.sth.detections != 1 {
		t.Errorf("detections %d", g.sth.detections)
	}
}

func TestStealthSeenAgainInRedResetsTimerAndInYellowGoesRed(t *testing.T) {
	g := stealthGame(t)
	g.stealthDetect()
	runStealth(g, 8)
	g.stealthDetect()
	if g.sth.t < sthRedTime-0.01 || g.sth.detections != 1 {
		t.Errorf("seen again in red: timer %.1f detections %d", g.sth.t, g.sth.detections)
	}
	g.sth.phase, g.sth.t = sthYellow, 3
	g.stealthDetect()
	if g.sth.phase != sthRed || g.sth.detections != 2 {
		t.Errorf("seen in caution goes back to red (phase %d, detections %d)", g.sth.phase, g.sth.detections)
	}
}

func TestStealthKillSeenIsDetection(t *testing.T) {
	g := stealthGame(t)
	suspectGuest(g, Vec{230, 300}, DirLeft)
	g.witness()
	if g.sth.phase != sthRed {
		t.Error("a witnessed kill is a detection")
	}
}

func TestStealthBoxHidesFromEveryone(t *testing.T) {
	g := stealthGame(t)
	guardAt(g, Vec{260, 300}, DirLeft)
	suspectGuest(g, Vec{140, 300}, DirRight)
	suspectCop(g, Vec{200, 380}, DirUp)
	if !g.stealthInteract(g.p) || !g.sth.box {
		t.Fatal("E must put the box on")
	}
	g.p.moving = true
	runStealth(g, 4)
	if g.sth.detections != 0 {
		t.Error("under the box nobody detects him")
	}
	if g.stealthSpeedMul(g.p) >= 1 {
		t.Error("the box must be slow")
	}
	if !g.stealthNoAttack() {
		t.Error("no attacks from inside the box")
	}
	g.stealthInteract(g.p)
	if g.sth.box {
		t.Error("E again takes the box off")
	}
}

func TestStealthBoxFailsOnBumpAndWhenStillInFrontOfAnOfficer(t *testing.T) {
	g := stealthGame(t)
	g.stealthInteract(g.p)
	suspectGuest(g, Vec{g.p.pos.X + sthBoxBump - 2, g.p.pos.Y}, DirLeft)
	runStealth(g, 0.1)
	if g.sth.phase != sthRed || g.sth.box {
		t.Error("bumping into the box blows it")
	}
	// standing still in front of an officer
	g2 := stealthGame(t)
	g2.stealthInteract(g2.p)
	suspectCop(g2, Vec{g2.p.pos.X + 40, g2.p.pos.Y}, DirLeft)
	g2.p.moving = true
	runStealth(g2, sthBoxStill+1)
	if g2.sth.detections != 0 {
		t.Error("walking in front of an officer under the box is fine")
	}
	g2.p.moving = false
	runStealth(g2, sthBoxStill-0.5)
	if g2.sth.detections != 0 {
		t.Error("blown too early")
	}
	runStealth(g2, 1)
	if g2.sth.phase != sthRed {
		t.Errorf("%.0f s still in front of an officer must blow the box", sthBoxStill)
	}
	// far from the officers standing still is fine
	g3 := stealthGame(t)
	g3.stealthInteract(g3.p)
	suspectCop(g3, Vec{g3.p.pos.X + 200, g3.p.pos.Y}, DirLeft)
	runStealth(g3, 6)
	if g3.sth.detections != 0 {
		t.Error("standing still far from the officers must be safe")
	}
}

func TestStealthBoxCannotBePutOnDuringTheAlert(t *testing.T) {
	g := stealthGame(t)
	g.stealthDetect()
	g.stealthInteract(g.p)
	if g.sth.box {
		t.Error("no box during the red phase")
	}
	g.sth.phase, g.sth.t = sthYellow, 5
	g.stealthInteract(g.p)
	if !g.sth.box {
		t.Error("the box is allowed during the caution")
	}
}

func TestStealthDetectedInTheBoxRunsTheSameCycle(t *testing.T) {
	g := stealthGame(t)
	g.stealthInteract(g.p)
	g.stealthDetect()
	if g.sth.box || g.sth.phase != sthRed || g.sth.detections != 1 {
		t.Error("detected in the box: box off and the usual red phase")
	}
}

func TestStealthEndUnseenOrSeen(t *testing.T) {
	g := stealthGame(t)
	g.snd = &Sounds{muted: true}
	if g.stealthDoneTitle("X") != "SIN SER VISTO" {
		t.Error("never detected: SIN SER VISTO")
	}
	g.stealthWin()
	if g.mode != ModeLevelDone || len(g.toasts) == 0 || g.toasts[0].name != "SIN SER VISTO" {
		t.Error("the unseen win shows the badge")
	}
	g2 := stealthGame(t)
	g2.snd = &Sounds{muted: true}
	g2.sth.detections = 2
	g2.stealthWin()
	if len(g2.toasts) != 0 || g2.stealthDoneTitle("X") != "X" {
		t.Error("a seen win has no badge")
	}
}

func TestStealthAlertSoundIsShortAndNotSilent(t *testing.T) {
	b := stealthAlertSfx()
	secs := float64(len(b)) / 4 / sampleRate
	if secs < 0.3 || secs > 1.2 {
		t.Errorf("the alert lasts %.2f s: it must be short", secs)
	}
	loud := false
	for i := 1; i < len(b); i += 2 {
		if b[i] != 0 && b[i] != 0xff {
			loud = true
			break
		}
	}
	if !loud {
		t.Error("the alert sound is silent")
	}
}

// The stealth layer: more space (fewer guests, hedges) and the original level is untouched.
func TestStealthLayerHedgesDoNotBreakTheLevels(t *testing.T) {
	layerGame(t, false) // builds levelDefs
	prev := noAssets
	noAssets = true
	t.Cleanup(func() { noAssets = prev })
	total := 0
	for i := 0; i < campaignN && i < len(levelDefs); i++ {
		def := levelDefs[i]
		n0 := len(def.Props)
		sd := stealthDef(def)
		if len(def.Props) != n0 {
			t.Fatalf("level %d: the original definition was changed", i)
		}
		total += len(sd.Props) - n0
		base, st := buildLevel(def), buildLevel(sd)
		count := func(l *Level) int {
			n := 0
			for _, r := range l.reachFrom(l.Def.Start) {
				if r {
					n++
				}
			}
			return n
		}
		b, s := count(base), count(st)
		if float64(s) < float64(b)*0.93 {
			t.Errorf("level %d (%s): hedges cut the walkable area %d -> %d", i, def.Title, b, s)
		}
		for _, p := range st.Props {
			if p.Name != "hedge" || originalProp(def, p) {
				continue
			}
			for _, w := range append([]Vec{def.Start}, flattenRoutes(def)...) {
				if p.Foot.Grow(12).Has(w.X, w.Y) {
					t.Errorf("level %d: a hedge sits on the start or a guard route point %v", i, w)
				}
			}
			for _, o := range st.Props {
				if o != p && !o.K.NoSolid && rectsTouch(p.Foot, o.Foot) {
					t.Errorf("level %d: hedge at %.0f,%.0f overlaps %s", i, p.X, p.Y, o.Name)
				}
			}
		}
	}
	if total == 0 {
		t.Error("no hedge was added to any level")
	}
}

func originalProp(def *LevelDef, p *Prop) bool {
	for _, in := range def.Props {
		if in.Kind == p.Name && in.X == p.X && in.Y == p.Y {
			return true
		}
	}
	return false
}

func flattenRoutes(def *LevelDef) []Vec {
	var out []Vec
	for _, r := range def.Guards {
		out = append(out, r...)
	}
	return out
}

func rectsTouch(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func TestStealthStartsAndLeavesTheMode(t *testing.T) {
	oldF, oldS, oldP := progressFrozen, debugSnap, debugPrevFrozen
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen = oldF, oldS, oldP })
	g := layerGame(t, true) // the 2 players switch is on: the mode drops the second player, nothing else
	progressFrozen = true
	g.startStealth(0)
	if !g.stealth || g.mode != ModePlay || g.is2P() || !twoPlayerMode {
		t.Fatalf("stealth=%v mode=%v 2P=%v", g.stealth, g.mode, g.is2P())
	}
	n := 0
	for _, gu := range g.guests {
		if gu.perf == "" && !gu.out() {
			n++
		}
	}
	if n > sthGuests {
		t.Errorf("%d guests: the stealth party has at most %d", n, sthGuests)
	}
	if g.slasherT < 1e8 || g.arrivalT < 1e8 {
		t.Error("random killers and arrivals must be off in the stealth mode")
	}
	g.startLevel(g.levelIdx) // retrying stays in the mode
	if !g.stealth {
		t.Error("retry left the stealth mode")
	}
	g.stealthExit()
	if g.stealth || !g.is2P() {
		t.Errorf("after leaving: stealth=%v 2P=%v (the next normal party has two players again)", g.stealth, g.is2P())
	}
}

func TestStealthDebugRowAndFlags(t *testing.T) {
	oldF, oldS, oldP := progressFrozen, debugSnap, debugPrevFrozen
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen = oldF, oldS, oldP })
	g := layerGame(t, false)
	found := false
	for _, r := range g.debugRows() {
		if r.label == stealthMenuLabel {
			found = true
		}
	}
	if !found {
		t.Error("the DEBUG menu has no stealth row")
	}
	names := strings.Join(mainFlagNames(t), " ")
	for _, f := range []string{"copmemory", "stealth"} {
		if !strings.Contains(" "+names+" ", " "+f+" ") {
			t.Errorf("flag -%s missing in main.go", f)
			continue
		}
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.Bool(f, false, "")
		fs.Parse([]string{"-" + f})
		if !debugFlagsActive(fs) {
			t.Errorf("-%s must freeze the progress", f)
		}
	}
}

// The HUD, the cones and the texts of the mode fit on the screen in both languages, in every phase.
func TestStealthHUDTextAudit(t *testing.T) {
	snapshotLang(t)
	prevLang, prevEd := gameLang, editorLang
	oldF := progressFrozen
	t.Cleanup(func() { gameLang, editorLang, progressFrozen = prevLang, prevEd, oldF })
	progressFrozen = true
	g := layerGame(t, false)
	for _, lang := range []string{"es", "en"} {
		for _, phase := range []int{sthCalm, sthYellow, sthRed} {
			for _, box := range []bool{false, true} {
				ph, bx, lang := phase, box, lang
				sh := auditShot{name: "stealth HUD", panels: false, setup: func(g *Game) {
					auditPlay(g, false)
					g.stealthNext = true
					g.loadLevel(0)
					g.lvl.graded = true
					g.titleT, g.mode = 0, ModePlay
					g.sth.phase, g.sth.t, g.sth.box, g.sth.meter = ph, 5, bx, 0.5
					if ph == sthRed {
						g.sth.exT, g.sth.exMax = 1, 1.6
					}
					g.msgs = nil
				}}
				rec, errs := auditRender(g, sh, lang)
				if errs != "" {
					t.Fatal(errs)
				}
				for _, f := range auditCheck("stealth HUD ["+lang+"]", rec, false, nil, nil) {
					t.Errorf("phase %d box %v: %s", ph, bx, f.msg)
				}
			}
		}
	}
	g.stealthNext = false
	g.loadLevel(0)
}
