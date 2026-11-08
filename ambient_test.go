package main

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func ambientLevelIdx(title string) int {
	for i, d := range levelDefs {
		if d.Title == title {
			return i
		}
	}
	return -1
}

func ambientSetup(t *testing.T) *Game {
	g := layerGame(t, false)
	if g.as.glow == nil { // the fog uses it; the audit game has no assets
		g.as.glow = genGlow(96).img()
		t.Cleanup(func() { g.as.glow = nil })
	}
	on, snd := ambientOn, ambientSound
	t.Cleanup(func() { ambientOn, ambientSound = on, snd; bgCur = nil })
	ambientOn = true
	return g
}

func TestAmbientPlanEveryLevelNoPanic(t *testing.T) {
	g := ambientSetup(t)
	n := 0
	for _, e := range ambientPlan {
		if e.Fx != 0 {
			n++
		}
	}
	if n < 10 || n > 14 {
		t.Fatalf("the plan has %d levels with an effect, want 10-14", n)
	}
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	for i, d := range levelDefs {
		g.loadLevel(i)
		g.lvl.graded = true
		fx, _ := ambientFor(d.Title)
		for _, tm := range []float64{0, 3.3, 17, 41.7, 95} {
			g.time = tm
			bgCount = 0
			g.drawBackAnim(scr)
			if fx == 0 && bgCount != 0 {
				t.Errorf("%s: no plan but %d rectangles drawn", d.Title, bgCount)
			}
		}
	}
	for _, e := range ambientPlan {
		if e.Fx != 0 && ambientLevelIdx(e.Title) < 0 && e.Title != "PLAYA" {
			t.Errorf("plan for %q: no such level", e.Title)
		}
	}
}

func TestAmbientDrawsAndSwitch(t *testing.T) {
	g := ambientSetup(t)
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	for _, title := range []string{"MIAMI BEACH", "CRUCERO NOCTURNO", "ESTACIÓN DE ESQUÍ", "CEMENTERIO", "ESTACIÓN ESPACIAL", "PLAZA EUROPEA", "BULL'S EYE"} {
		i := ambientLevelIdx(title)
		if i < 0 {
			t.Fatalf("%s missing", title)
		}
		g.loadLevel(i)
		g.lvl.graded = true
		g.time = 14
		ambientOn = true
		bgCount = 0
		g.drawBackAnim(scr)
		if bgCount == 0 && title != "CRUCERO NOCTURNO" {
			t.Errorf("%s: nothing drawn", title)
		}
		ambientOn = false
		bgCount = 0
		g.drawBackAnim(scr)
		if bgCount != 0 {
			t.Errorf("%s: ambientOn=false still draws", title)
		}
	}
}

func TestAmbientDeterministic(t *testing.T) {
	g := ambientSetup(t)
	for _, title := range []string{"CRUCERO NOCTURNO", "MIAMI BEACH", "ESTACIÓN ESPACIAL"} {
		g.loadLevel(ambientLevelIdx(title))
		bgCur = nil
		a := *bgFor(g.lvl.Def)
		bgCur = nil
		b := *bgFor(g.lvl.Def)
		if !reflect.DeepEqual(a.crests, b.crests) || !reflect.DeepEqual(a.spots, b.spots) || !reflect.DeepEqual(a.stars, b.stars) {
			t.Errorf("%s: the tables differ between two builds", title)
		}
	}
	if bgJumpStart(3) != bgJumpStart(3) {
		t.Error("jump start not deterministic")
	}
}

// A leap every 20-40 s: gaps between consecutive starts.
func TestAmbientJumpSpacing(t *testing.T) {
	for k := 0; k < 200; k++ {
		gap := bgJumpStart(k+1) - bgJumpStart(k)
		if gap < 20 || gap > 40 {
			t.Fatalf("gap %d->%d = %.1f s, want 20-40", k, k+1, gap)
		}
	}
	g := ambientSetup(t)
	g.loadLevel(ambientLevelIdx("CRUCERO NOCTURNO"))
	if len(bgFor(g.lvl.Def).spots) == 0 {
		t.Fatal("the cruise has no water spot for the dolphin")
	}
}

// Decoration only: the effects never touch the level (hitboxes, props, solids).
func TestAmbientDoesNotChangeLevel(t *testing.T) {
	g := ambientSetup(t)
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	g.loadLevel(ambientLevelIdx("MIAMI BEACH"))
	g.lvl.graded = true
	type snap struct {
		foot  []Rect
		sort  []float64
		walls int
		solid int
	}
	take := func() snap {
		s := snap{walls: len(g.lvl.Walls), solid: len(g.lvl.solidsP)}
		for _, p := range g.lvl.Props {
			s.foot = append(s.foot, p.Foot)
			s.sort = append(s.sort, p.Sort)
		}
		return s
	}
	before := take()
	for i := 0; i < 600; i++ {
		g.time = float64(i) / 10
		g.drawBackAnim(scr)
	}
	if !reflect.DeepEqual(before, take()) {
		t.Fatal("drawing the background animation changed the level")
	}
}

func TestAmbientSwayShift(t *testing.T) {
	g := ambientSetup(t)
	g.loadLevel(ambientLevelIdx("MIAMI BEACH"))
	g.lvl.graded = true
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	var palm *Prop
	for _, p := range g.lvl.Props {
		if p.Name == "palm" {
			palm = p
			break
		}
	}
	if palm == nil {
		t.Skip("no palm") // the Miami plan always has palms; guard only
	}
	img := g.as.props["palm"][0].img
	swayed, still := 0, 0
	for i := 0; i < 1200; i++ {
		g.time = float64(i) / 10
		if g.drawSway(scr, palm, img, &ebiten.DrawImageOptions{}) {
			swayed++
		} else {
			still++
		}
	}
	if swayed == 0 || still == 0 {
		t.Fatalf("a palm must sway part of the time: swayed=%d still=%d", swayed, still)
	}
	ambientOn = false
	if g.drawSway(scr, palm, img, &ebiten.DrawImageOptions{}) {
		t.Fatal("ambientOn=false still sways")
	}
	ambientOn = true
	g.loadLevel(ambientLevelIdx("DISCOTECA"))
	g.time = 3
	for i := 0; i < 100; i++ {
		g.time = float64(i)
		if g.drawSway(scr, palm, img, &ebiten.DrawImageOptions{}) {
			t.Fatal("a level without plan sways")
		}
	}
}

func TestAmbientAllocsPerFrame(t *testing.T) {
	g := ambientSetup(t)
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	// Ebiten itself allocates a little per DrawImage; ours must not add anything on top of that.
	base := testing.AllocsPerRun(20, func() {
		for i := 0; i < 100; i++ {
			bgRect(scr, 1, 1, 2, 1, bgWhite(100))
		}
	}) / 100
	for _, title := range []string{"MIAMI BEACH", "CRUCERO NOCTURNO", "ESTACIÓN DE ESQUÍ", "CEMENTERIO", "ESTACIÓN ESPACIAL", "PLAZA EUROPEA"} {
		g.loadLevel(ambientLevelIdx(title))
		g.lvl.graded = true
		g.time = 12.5
		g.drawBackAnim(scr) // builds the table
		bgCount = 0
		allocs := testing.AllocsPerRun(20, func() { g.time += 0.017; g.drawBackAnim(scr) })
		per := allocs / (float64(bgCount) / 21)
		t.Logf("%s: %.0f allocs/frame, %d draws/frame, %.2f per draw (bare DrawImage: %.2f)", title, allocs, bgCount/21, per, base)
		if per > base+0.6 {
			t.Errorf("%s: %.2f allocations per draw, more than a bare DrawImage (%.2f)", title, per, base)
		}
	}
}

func TestAmbientDraw2P(t *testing.T) {
	g := layerGame(t, true)
	on := ambientOn
	t.Cleanup(func() { ambientOn = on; bgCur = nil })
	ambientOn = true
	scr := ebiten.NewImage(screenW, screenH)
	defer scr.Deallocate()
	for _, title := range []string{"MIAMI BEACH", "CRUCERO NOCTURNO"} {
		g.loadLevel(ambientLevelIdx(title))
		g.lvl.graded = true
		g.mode = ModePlay
		g.time = 31
		g.Draw(scr)
	}
}

func TestAmbientSoundLoops(t *testing.T) {
	for _, k := range []string{"waves", "wind", "crickets", "train", "crowd", "hum"} {
		b := buildAmbientLoop(k)
		if len(b) != int(ambLoopSec*sampleRate)*4 {
			t.Errorf("%s: %d bytes", k, len(b))
		}
		var pk int
		for i := 0; i+1 < len(b); i += 4 {
			v := int(int16(uint16(b[i]) | uint16(b[i+1])<<8))
			if v < 0 {
				v = -v
			}
			if v > pk {
				pk = v
			}
		}
		if pk < 2000 || pk > 32000 {
			t.Errorf("%s: peak %d out of range", k, pk)
		}
	}
	if buildAmbientLoop("nope") != nil {
		t.Error("unknown kind should give nil")
	}
	if ambientVolume > 0.5*0.15 {
		t.Errorf("ambient volume %.3f is above 15%% of the music", ambientVolume)
	}
}

func TestAmbientSoundSwitchAndMute(t *testing.T) {
	g := ambientSetup(t)
	g.mode = ModePlay
	g.loadLevel(ambientLevelIdx("MIAMI BEACH"))
	oldSnd := g.snd
	t.Cleanup(func() { g.snd = oldSnd; g.mode = ModeMenu })
	if g.snd == nil {
		g.snd = &Sounds{ctx: audioCtx, sfx: map[string][]byte{}, tracks: map[string][]byte{}, volume: 1}
	}
	m := g.snd.muted
	t.Cleanup(func() { g.snd.muted = m })
	g.snd.muted = false

	ambientSound = false
	if g.ambientWanted() != "" {
		t.Fatal("ambientSound off must be silent")
	}
	ambientSound = true
	if g.ambientWanted() != "waves" {
		t.Fatalf("Miami wants waves, got %q", g.ambientWanted())
	}
	g.snd.muted = true
	if g.ambientWanted() != "" {
		t.Fatal("muted must be silent")
	}
	g.snd.muted = false
	g.mode = ModeMenu
	if g.ambientWanted() != "" {
		t.Fatal("menus must be silent")
	}
	g.mode = ModePlay
	g.helpOv = true
	if g.ambientWanted() != "" {
		t.Fatal("help window must pause it")
	}
	g.helpOv = false
	g.loadLevel(ambientLevelIdx("DISCOTECA"))
	if g.ambientWanted() != "" {
		t.Fatal("a level with no plan has no ambience")
	}
}
