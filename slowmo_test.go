package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func slowOn(t *testing.T) {
	t.Helper()
	prev := slowmoEnabled
	slowmoEnabled = true
	t.Cleanup(func() { slowmoEnabled = prev })
}

func TestSlowCurveShape(t *testing.T) {
	if slowCurve(0) != 1 || slowCurve(slowTotal) != 1 || slowCurve(slowTotal+5) != 1 {
		t.Error("the curve must start and end at normal speed")
	}
	if slowCurve(slowHold-0.01) != slowFactor || slowCurve(slowIn) != slowFactor {
		t.Error("the curve must hold at slowFactor")
	}
	prev := 1.0
	for tt := 0.0; tt <= slowHold; tt += 1.0 / 60 { // goes down...
		if c := slowCurve(tt); c > prev+1e-12 {
			t.Fatalf("up at %.3f: %.3f > %.3f", tt, c, prev)
		} else {
			prev = c
		}
	}
	for tt := slowHold; tt <= slowTotal; tt += 1.0 / 60 { // ...and back up, smoothly (never jumps by more than 0.2 in a frame)
		c := slowCurve(tt)
		if c < prev-1e-12 || c-prev > 0.2 {
			t.Fatalf("bad ramp at %.3f: %.3f after %.3f", tt, c, prev)
		}
		prev = c
	}
}

// The last kill starts the slow motion once; the party ends within slowTotal real seconds; nothing starts it again.
func TestSlowmoLastKillOnceAndEnds(t *testing.T) {
	slowOn(t)
	g := hudPlay(t, false)
	keep := leaveAlive(g, 1)
	keep[0].dead = true // as killGuest would leave him
	g.checkDone()
	if !g.slow.active || g.mode != ModePlay {
		t.Fatalf("slow motion not started (active=%v mode=%v)", g.slow.active, g.mode)
	}
	if g.slowScale() != 1 {
		t.Errorf("scale at t=0 is %v", g.slowScale())
	}
	frames := 0
	minScale := 1.0
	for g.mode == ModePlay && frames < 200 {
		g.checkDone() // an explosion chain asking again must change nothing
		if s := g.slowScale(); s < minScale {
			minScale = s
		}
		g.slowTick()
		frames++
	}
	if g.mode == ModePlay {
		t.Fatal("the party never ended")
	}
	if secs := float64(frames) / 60; secs > slowTotal+0.1 || secs < slowTotal-0.1 {
		t.Errorf("ended after %.2f real seconds, want ~%.1f (and never above 1.5)", secs, slowTotal)
	}
	if minScale != slowFactor {
		t.Errorf("deepest scale %.2f, want %.2f", minScale, slowFactor)
	}
	if g.slow.active || !g.slow.fired {
		t.Errorf("state after the end: %+v", g.slow)
	}
	// once per level: a second "last kill" in the same level ends it at once
	g.mode = ModePlay
	g.checkDone()
	if g.slow.active {
		t.Error("slow motion fired twice in one level")
	}
}

func TestSlowmoOffAndExcluded(t *testing.T) {
	g := hudPlay(t, false)
	leaveAlive(g, 0)
	prev := slowmoEnabled
	t.Cleanup(func() { slowmoEnabled = prev; g.stealth = false; g.walkBot = false })
	slowmoEnabled = false
	g.checkDone()
	if g.slow.active || g.mode == ModePlay {
		t.Errorf("disabled: active=%v mode=%v (the party must end at once)", g.slow.active, g.mode)
	}
	slowmoEnabled = true
	auditPlay(g, false)
	leaveAlive(g, 0)
	g.stealth = true
	if g.slowBegin() {
		t.Error("slow motion in STEALTH")
	}
	g.stealth = false
	g.walkBot = true
	if g.slowBegin() {
		t.Error("slow motion with the walkbot")
	}
}

// If somebody turns up during the slow motion the party goes on; if the player is caught the effect is dropped.
func TestSlowmoCancelledByNewcomerOrLoss(t *testing.T) {
	slowOn(t)
	g := hudPlay(t, false)
	leaveAlive(g, 0)
	g.checkDone()
	if !g.slow.active {
		t.Fatal("not started")
	}
	g.guests = append(g.guests, &Guest{pos: Vec{100, 100}}) // a latecomer
	for i := 0; i < 80; i++ {
		g.slowTick()
	}
	if g.mode != ModePlay || g.slow.active {
		t.Errorf("a newcomer should keep the party going (mode %v, active %v)", g.mode, g.slow.active)
	}
	auditPlay(g, false)
	leaveAlive(g, 0)
	g.checkDone()
	g.mode = ModeLose
	g.slowTick()
	if g.slow.active {
		t.Error("slow motion kept running after losing")
	}
	g.mode = ModePlay
}

func TestSlowmoDrawNoPanicAndFlash(t *testing.T) {
	slowOn(t)
	g := hudPlay(t, true)
	leaveAlive(g, 0)
	g.checkDone()
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.drawSlowFx(img) // flash + vignette at t=0
	g.slow.t = slowHold
	g.drawSlowFx(img) // vignette only
	g.slow.t = slowTotal - 0.01
	g.drawSlowFx(img)
	if slowFlashAlpha(0) < 150 || slowFlashAlpha(slowFlashT) != 0 || slowFlashAlpha(slowFlashT/2) >= slowFlashAlpha(0) {
		t.Errorf("flash alpha: %d at 0, %d at half, %d at the end", slowFlashAlpha(0), slowFlashAlpha(slowFlashT/2), slowFlashAlpha(slowFlashT))
	}
	if slowVignetteAlpha(1, 0) == 0 || slowVignetteAlpha(0, 0) != 0 || slowVignetteAlpha(1, 6) >= slowVignetteAlpha(1, 0) {
		t.Error("the vignette must be dark at the edge, fade inwards and vanish when the time is normal")
	}
	g.mode = ModePlay
}
