package main

import (
	"flag"
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func blackoutGame(t *testing.T) *Game {
	g, _ := camGame(t)
	g.snd = &Sounds{muted: true}
	return g
}

func TestBlackoutChanceIsRareAndDeterministic(t *testing.T) {
	if blackoutChance > 1.0/15 {
		t.Fatalf("too frequent: %v", blackoutChance)
	}
	hits := 0
	const n = 20000
	r := rand.New(rand.NewSource(42))
	for i := 0; i < n; i++ {
		if b, ok := blackoutPlan(r, false); ok {
			hits++
			if b.dur < blackoutMin || b.dur > blackoutMax || b.at < blackoutAtMin || b.at > blackoutAtMax {
				t.Fatalf("plan out of range: %+v", b)
			}
		}
	}
	if f := float64(hits) / n; math.Abs(f-blackoutChance) > 0.01 {
		t.Errorf("frequency %.4f, want about %.4f", f, blackoutChance)
	}
	a, oka := blackoutPlan(rand.New(rand.NewSource(7)), false)
	b, okb := blackoutPlan(rand.New(rand.NewSource(7)), false)
	if a != b || oka != okb {
		t.Error("same seed, different plan")
	}
}

func TestBlackoutNeverInStealth(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 5000; i++ {
		if _, ok := blackoutPlan(r, true); ok {
			t.Fatal("stealth must never have a blackout")
		}
	}
	g := blackoutGame(t)
	g.stealth = true
	g.blk = blackoutState{planned: true, at: 0, dur: 9}
	g.updateBlackout(0.1)
	if g.blackoutOn() {
		t.Error("it started in stealth")
	}
	old := dbgBlackout
	dbgBlackout = true
	t.Cleanup(func() { dbgBlackout = old })
	g.blackoutLevelReset()
	if g.blk.planned {
		t.Error("the flag must not force it in stealth")
	}
}

func TestBlackoutDurationAndEffects(t *testing.T) {
	g := blackoutGame(t)
	g.visK = 1
	g.blk = blackoutState{planned: true, at: 5, dur: 10}
	g.time = 1
	g.updateBlackout(0.1)
	if g.blackoutOn() {
		t.Fatal("started too early")
	}
	g.time = 6
	g.updateBlackout(0.1)
	if !g.blackoutOn() {
		t.Fatal("did not start")
	}
	// cameras off: the player is in the cone but nobody sees him
	c := g.cams[0]
	if c.sees(g, g.p.pos) {
		t.Error("a camera saw during the blackout")
	}
	g.updateCameras(0.1)
	if c.led > 0 {
		t.Error("the LED lit during the blackout")
	}
	// sight halved
	if g.blackoutSight() != blackoutVisK {
		t.Error("sight not reduced")
	}
	from := Vec{300, 300}
	to := Vec{300 + 0.7*guestRange, 300}
	g.blk.t = 5
	if g.sees(from, DirRight, to, guestRange, guestDot) {
		t.Error("seen at 70% range in the dark")
	}
	g.blk.t = 0
	if !g.sees(from, DirRight, to, guestRange, guestDot) {
		t.Error("not seen at 70% range with light")
	}
	g.blk.t = 10
	for i := 0; i < 200 && g.blackoutOn(); i++ {
		g.updateBlackout(0.1)
	}
	if g.blackoutOn() || !g.blk.done {
		t.Fatal("did not end")
	}
	if !c.sees(g, g.p.pos) {
		t.Error("the camera must work again")
	}
	g.blk.started = false
	g.updateBlackout(0.1)
	if g.blackoutOn() {
		t.Error("it must happen once per party")
	}
}

func TestBlackoutFadeAndDraw(t *testing.T) {
	g := auditGame(t)
	auditReset(g)
	g.blk = blackoutState{planned: true, started: true, dur: 10, t: 5}
	if k := g.blackoutK(); k != 1 {
		t.Errorf("k=%v", k)
	}
	g.blk.t = 10
	if k := g.blackoutK(); k != 0 {
		t.Errorf("k at start=%v", k)
	}
	g.blk.t = 5
	if blackoutHaloR < 48 {
		t.Error("the halo must be big enough to play")
	}
	scr := ebiten.NewImage(screenW, screenH)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic drawing: %v", r)
			}
		}()
		g.drawBlackout(scr)
		g.blk.t = 0.1
		g.drawBlackout(scr)
		g.blk.t = 0
		g.drawBlackout(scr)
	}()
}

func TestBlackoutFlagIsFrozen(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Bool("blackout", false, "")
	_ = fs.Parse([]string{"-blackout"})
	if !debugFlagsActive(fs) {
		t.Error("-blackout must freeze the progress")
	}
}

func TestMusicReactTransitionAndSwitch(t *testing.T) {
	g := blackoutGame(t)
	g.musicReact = 0
	if g.musicReactTarget() != 0 {
		t.Error("calm party must not react")
	}
	g.wanted = 5
	if g.musicReactTarget() != 1 {
		t.Error("wanted must react")
	}
	k := 0.0
	steps := 0
	for k < 1 {
		k = musicReactStep(k, 1, 1.0/60)
		steps++
	}
	if math.Abs(float64(steps)/60-musicReactFade) > 0.1 {
		t.Errorf("transition took %v s", float64(steps)/60)
	}
	for i := 0; i < 130; i++ {
		k = musicReactStep(k, 0, 1.0/60)
	}
	if k != 0 {
		t.Errorf("did not return to normal: %v", k)
	}
	if v := musicReactVolume(0.5, 1); v > 0.5*1.12 || v <= 0.5 {
		t.Errorf("boost out of the soft range: %v", v)
	}
	g.policeCalled, g.wanted = true, 0
	if g.musicReactTarget() != 1 {
		t.Error("police called must react")
	}
	g.policeCalled = false
	g.stealth, g.sth.phase = true, sthRed
	if g.musicReactTarget() != 1 {
		t.Error("red phase must react")
	}
	g.stealth = false
	g.wanted = 5
	g.mode = ModeMenu
	if g.musicReactTarget() != 0 {
		t.Error("only party music")
	}
	g.mode = ModePlay
	old := musicReactEnabled
	musicReactEnabled = false
	t.Cleanup(func() { musicReactEnabled = old })
	if g.musicReactTarget() != 0 {
		t.Error("the switch must disable it")
	}
	(&Sounds{}).SetReact(1) // no player: must not panic
}
