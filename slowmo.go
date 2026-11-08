package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// SLOW MOTION OF THE LAST KILL. When the last person of the party falls (g.aliveCount() reaches 0 in checkDone),
// the game does not end at once: for slowTotal REAL seconds the time runs slower (dt x slowFactor, with a quick
// ease in, a hold and a smooth ramp back to 1), a white flash with a dark vignette marks the moment and a deep
// "whoosh" plays (sfx "slowmo", audio.go). When the time is up, completeLevel runs as it always did.
//   - Only once per level (slowState.fired): a chain of explosions cannot start it again.
//   - Only in ModePlay of a normal party: never in STEALTH, interludes or endings (those are other modes).
//   - The countdown runs in real frames (1/60 each), so the end screen is never later than slowTotal seconds.
//   - slowmoEnabled switches it off (tests do, see slowmo_flag_test.go; the -walkbot and the selftest too).
//   - If somebody new shows up during the slow motion (aliveCount > 0 again) the party simply goes on.
// ---------------------------------------------------------------------------

const (
	slowFactor = 0.25 // dt multiplier at the deepest point
	slowIn     = 0.08 // seconds to get there
	slowHold   = 0.55 // seconds at slowFactor (counted from the kill)
	slowTotal  = 1.0  // real seconds from the last kill to the end of the party
	slowFlashT = 0.30 // length of the white flash
)

// slowmoEnabled: the whole effect can be turned off (tests, benchmarks).
var slowmoEnabled = true

type slowState struct {
	fired  bool    // already used in this level
	active bool    // running now
	t      float64 // real seconds since the last kill
}

// slowCurve is the dt multiplier t real seconds after the last kill.
func slowCurve(t float64) float64 {
	switch {
	case t <= 0:
		return 1
	case t < slowIn:
		return lerp(1, slowFactor, t/slowIn)
	case t < slowHold:
		return slowFactor
	case t < slowTotal:
		u := (t - slowHold) / (slowTotal - slowHold)
		return lerp(slowFactor, 1, u*u*(3-2*u))
	}
	return 1
}

// slowScale is the multiplier the update loop applies to dt.
func (g *Game) slowScale() float64 {
	if !g.slow.active {
		return 1
	}
	return slowCurve(g.slow.t)
}

// slowBegin is called when the last person has fallen. It returns true when the end of the party is postponed.
func (g *Game) slowBegin() bool {
	s := &g.slow
	if s.active {
		return true
	}
	if s.fired || !slowmoEnabled || g.stealth || g.mode != ModePlay || g.walkBot || g.selftest > 0 {
		return false
	}
	s.fired, s.active, s.t = true, true, 0
	if g.snd != nil {
		g.snd.Play("slowmo")
	}
	return true
}

// slowTick counts the real time of the effect (once per frame, after updatePlay) and ends the party when it is over.
func (g *Game) slowTick() {
	s := &g.slow
	if !s.active {
		return
	}
	if g.mode != ModePlay { // lost, left or restarted meanwhile
		s.active = false
		return
	}
	s.t += 1.0 / 60
	if s.t >= slowTotal {
		s.active = false
		if g.aliveCount() == 0 {
			g.completeLevel()
		}
	}
}

// slowFlashAlpha: strength (0..255) of the white flash t real seconds after the kill.
func slowFlashAlpha(t float64) uint8 {
	if t < 0 || t >= slowFlashT {
		return 0
	}
	u := 1 - t/slowFlashT
	return uint8(255 * u * u * 0.85)
}

// slowVignetteAlpha: darkness of the i-th edge band (0 = outermost) for a slow-motion depth k (0..1).
func slowVignetteAlpha(k float64, i int) uint8 { return uint8(k * float64(120-i*17)) }

// drawSlowFx: the white flash and the dark edges while the time is slow.
func (g *Game) drawSlowFx(dst *ebiten.Image) {
	s := &g.slow
	if !s.active || g.mode != ModePlay {
		return
	}
	k := (1 - slowCurve(s.t)) / (1 - slowFactor) // 0 = normal speed, 1 = deepest
	for i := 0; i < 7; i++ {
		a := slowVignetteAlpha(k, i)
		in, th := float64(i*10), 10.0
		c := color.RGBA{0, 0, 0, a}
		fillRect(dst, in, in, screenW-2*in, th, c)
		fillRect(dst, in, screenH-in-th, screenW-2*in, th, c)
		fillRect(dst, in, in+th, th, screenH-2*in-2*th, c)
		fillRect(dst, screenW-in-th, in+th, th, screenH-2*in-2*th, c)
	}
	if a := slowFlashAlpha(s.t); a > 0 {
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{a, a, a, a})
	}
}
