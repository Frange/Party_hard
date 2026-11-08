package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The title: what the program shows when it opens, before the main menu.
//
// Fade in from black, two spotlights sweeping the dark, PARTY slides in, HARDY drops with
// a slam (flash, shake, shock rings, coloured sparks), the neon flickers, the subtitle and the
// credits appear and, after a few seconds, a fade to black leads to the main menu or, the
// very first time (nothing saved, intro not seen), to the intro of the room.
// Enter, Esc or a click skip it. It plays once per run of the program (never when coming
// back to the menu from a party) and none of the debug flags show it unless -title is given.
// ---------------------------------------------------------------------------

type titleSpark struct {
	x, y, vx, vy float64
	life, max    float64
	col          color.RGBA
}

type titleState struct {
	t        float64
	leaving  bool
	leaveT   float64
	fadeLen  float64 // how long the fade to black lasts (shorter when skipped)
	slammed  bool
	sparks   []titleSpark
	shakeT   float64
	flashT   float64
	ringT    float64
	seedBase uint32
}

// the timeline, in seconds
const (
	tiFadeIn   = 0.55
	tiParty    = 1.2  // PARTY slides in
	tiSlam     = 1.75 // HARDY hits the ground
	tiSub      = 2.9  // the subtitle
	tiHome     = 3.5  // "homenaje a Party Hard"
	tiCredits  = 4.1  // the credits
	tiEnd      = 6.0  // the fade to black starts
	tiFadeOut  = 0.55
	tiSkipFade = 0.3
	titleY     = 162.0 // where HARDY lands
)

// titleDuration is how long the title lasts when nobody skips it.
const titleDuration = tiEnd + tiFadeOut

func (g *Game) startTitle() {
	g.title = &titleState{seedBase: 0x9e3779b9}
	g.mode = ModeTitle
	g.snd.StopMusic()
	g.snd.StartScene("base", "base:title", ambientBuilder(ambTitle), "", true)
}

// finishTitle ends the title: the intro of the room the very first time, the main menu otherwise.
func (g *Game) finishTitle() {
	g.title = nil
	g.snd.StopScene()
	if !noAssets && g.firstRun() {
		g.bootIntro()
		return
	}
	g.mode = ModeMenu
	g.mainSel = 0
	g.snd.StartMusic()
}

// skipTitle is what Enter, Esc or a click do: a quick fade to black and on.
func (g *Game) skipTitle() {
	if ti := g.title; ti != nil && !ti.leaving {
		ti.leaving, ti.leaveT, ti.fadeLen = true, 0, tiSkipFade
	}
}

func (g *Game) updateTitle() {
	ti := g.title
	if ti == nil {
		g.finishTitle()
		return
	}
	const dt = 1.0 / 60
	ti.t += dt
	g.time = ti.t
	if actConfirm.just() || actBack.just() || ebitenClick() {
		g.skipTitle()
	}
	if !ti.leaving && ti.t >= tiEnd {
		ti.leaving, ti.leaveT, ti.fadeLen = true, 0, tiFadeOut
	}
	// the music: up with the lights, down with the fade
	vol := smooth01(ti.t / 0.6)
	if ti.leaving {
		vol *= 1 - smooth01(ti.leaveT/ti.fadeLen)
	}
	g.snd.SetSceneVolume("base", vol)
	if ti.leaving {
		ti.leaveT += dt
		if ti.leaveT >= ti.fadeLen {
			g.finishTitle()
			return
		}
	}
	// the impact of HARDY: shake, flash, rings and sparks
	if !ti.slammed && ti.t >= tiSlam {
		ti.slammed = true
		ti.shakeT, ti.flashT, ti.ringT = 0.45, 0.28, 0
		g.snd.Play("thud")
		ti.addSparks(screenW/2, titleY+70)
	}
	if ti.slammed {
		ti.ringT += dt
	}
	if ti.shakeT > 0 {
		ti.shakeT -= dt
	}
	if ti.flashT > 0 {
		ti.flashT -= dt
	}
	for i := 0; i < len(ti.sparks); {
		s := &ti.sparks[i]
		s.life -= dt
		if s.life <= 0 {
			ti.sparks = append(ti.sparks[:i], ti.sparks[i+1:]...)
			continue
		}
		s.x += s.vx * dt
		s.y += s.vy * dt
		s.vy += 260 * dt
		i++
	}
}

// addSparks bursts coloured sparks (pink, cyan, yellow, violet: never white) from a point.
func (ti *titleState) addSparks(x, y float64) {
	cols := []color.RGBA{rgb(0xff3070), rgb(0x60d0f0), rgb(0xf0c030), rgb(0x9040e0)}
	h := ti.seedBase
	rnd := func() float64 { // a small deterministic generator: the same burst every time
		h ^= h << 13
		h ^= h >> 17
		h ^= h << 5
		return float64(h%10000) / 10000
	}
	for i := 0; i < 90; i++ {
		a := rnd() * 2 * math.Pi
		v := 60 + rnd()*260
		max := 0.6 + rnd()*0.8
		ti.sparks = append(ti.sparks, titleSpark{
			x: x + (rnd()-0.5)*220, y: y + (rnd()-0.5)*20,
			vx: math.Cos(a) * v, vy: math.Sin(a)*v*0.7 - 90,
			life: max, max: max, col: cols[i%len(cols)],
		})
	}
}

// ---- drawing ------------------------------------------------------------------------------------

func titleEase(x float64) float64 { return 1 - math.Pow(1-clampF(x, 0, 1), 3) }

// neonOn: the flicker of the neon (a few quick drops of light), the same on every run.
func neonOn(t float64) bool {
	switch {
	case t > 2.25 && t < 2.62:
		return int((t-2.25)*30)%3 != 1
	case t > 3.62 && t < 3.78:
		return int((t-3.62)*40)%2 == 0
	}
	return true
}

// drawCone paints a soft spotlight from the top of the screen.
func drawCone(dst *ebiten.Image, x0, angle, alpha, spread float64) {
	for y := 0; y < screenH; y += 2 {
		fy := float64(y)
		cx := x0 + fy*math.Tan(angle)
		half := 10 + fy*spread
		fade := 1 - fy/float64(screenH)
		a := alpha * fade * fade
		for k := 0; k < 3; k++ { // a soft edge: three nested bands
			w := half * (1 - float64(k)*0.28)
			fillRect(dst, math.Round(cx-w), fy, math.Round(2*w), 2, color.RGBA{uint8(255 * a * 0.1), uint8(230 * a * 0.1), uint8(180 * a * 0.1), uint8(255 * a * 0.1)})
		}
	}
}

func (g *Game) drawTitle(dst *ebiten.Image) {
	ti := g.title
	if ti == nil {
		return
	}
	t := ti.t
	dst.Fill(rgb(0x05030a))
	// a very dark gradient of the night, no dots
	for y := 0; y < screenH; y += 6 {
		k := float64(y) / screenH
		fillRect(dst, 0, float64(y), screenW, 6, mixc(rgb(0x05030a), rgb(0x160a2a), k*0.9))
	}
	// the spotlights: they sweep, then settle crossed behind the logo
	lights := smooth01((t - 0.3) / 0.8)
	settle := smooth01((t - tiSlam) / 0.8)
	a1 := lerp(0.55*math.Sin(t*1.7), 0.22, settle)
	a2 := lerp(-0.55*math.Sin(t*1.7+0.9), -0.22, settle)
	drawCone(dst, 150, a1, 0.9*lights, 0.24)
	drawCone(dst, screenW-150, a2, 0.9*lights, 0.24)

	// the shake of the impact moves the whole logo block
	ox, oy := 0.0, 0.0
	if ti.shakeT > 0 {
		k := ti.shakeT / 0.45
		ox = math.Round(math.Sin(t*90) * 6 * k)
		oy = math.Round(math.Cos(t*77) * 5 * k)
	}
	cx := float64(screenW) / 2

	// the glow behind the logo
	if t >= tiParty {
		glow := smooth01((t - tiSlam) / 0.5)
		for i := 7; i >= 1; i-- {
			pad := float64(i) * 14
			a := uint8(7 * glow)
			fillRect(dst, cx-280-pad+ox, titleY-10-pad/2+oy, 560+2*pad, 160+pad, color.RGBA{uint8(0x90 * a / 255), uint8(0x20 * a / 255), uint8(0xa0 * a / 255), a})
		}
	}

	// PARTY slides in from the left
	if t >= tiParty {
		k := titleEase((t - tiParty) / 0.45)
		px := lerp(-200, cx, k)
		col := rgb(0x60d0f0)
		if !neonOn(t) {
			col = rgb(0x2a6a80)
		}
		g.gradText(dst, "PARTY", math.Round(px)+ox, titleY-66+oy, col, col, rgb(0x0a1a2a), 5)
	}
	// HARDY falls and bounces
	if t >= tiSlam-0.34 {
		y := titleY
		if t < tiSlam {
			k := (t - (tiSlam - 0.34)) / 0.34
			y = lerp(-170, titleY, k*k)
		} else {
			d := t - tiSlam
			y = titleY - 16*math.Abs(math.Sin(d*11))*math.Exp(-d*5.5)
		}
		a, b := rgb(0xff3070), rgb(0x9040e0)
		if !neonOn(t) {
			a, b = rgb(0x70203a), rgb(0x3a2060)
		}
		g.gradText(dst, "HARDY", cx+ox, math.Round(y)+oy, a, b, rgb(0x300050), 10)
	}
	// the shock rings of the slam
	if ti.slammed {
		r := ti.ringT * 520
		ringAt(dst, cx, titleY+80, r, 3, rgba(0xff3070, uint8(clampF(200-ti.ringT*380, 0, 200))))
		ringAt(dst, cx, titleY+80, r*0.7, 2, rgba(0x60d0f0, uint8(clampF(160-ti.ringT*320, 0, 160))))
	}
	// the sparks
	for _, s := range ti.sparks {
		a := uint8(255 * clampF(s.life/s.max, 0, 1))
		fillRect(dst, math.Round(s.x), math.Round(s.y), 3, 3, color.RGBA{s.col.R, s.col.G, s.col.B, a})
	}

	// the subtitle, spaced out, and the thin lines that open under the logo
	if t >= tiSub {
		k := smooth01((t - tiSub) / 0.5)
		w := 330 * titleEase((t-tiSub)/0.7)
		fillRect(dst, cx-w, titleY+182, 2*w, 2, mixc(rgb(0x05030a), rgb(0x9040e0), k))
		spaced(g, dst, "FIESTA NOCTURNA", cx, titleY+194, 14, mixc(rgb(0x05030a), rgb(0xff7aa8), k), 2)
	}
	if t >= tiHome {
		k := smooth01((t - tiHome) / 0.5)
		g.drawTextC(dst, "homenaje a Party Hard", cx, titleY+236, mixc(rgb(0x05030a), rgb(0x9a90c0), k), 1)
	}
	if t >= tiCredits {
		k := smooth01((t - tiCredits) / 0.5)
		g.drawTextC(dst, "Un juego de Frange", cx, screenH-46, mixc(rgb(0x05030a), rgb(0xe8e0a0), k), 1)
	}

	// the flash of the impact, the fade in and the fade out
	if ti.flashT > 0 {
		a := uint8(80 * ti.flashT / 0.28)
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{a, a * 7 / 10, a, a})
	}
	black := 0.0
	if t < tiFadeIn {
		black = 1 - t/tiFadeIn
	}
	if ti.leaving {
		black = math.Max(black, clampF(ti.leaveT/ti.fadeLen, 0, 1))
	}
	if black > 0 {
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{0, 0, 0, uint8(255 * black)})
	}
}

// spaced writes a line with a fixed gap between the letters, centred on cx, at whole pixels.
func spaced(g *Game, dst *ebiten.Image, s string, cx, y, gap float64, c color.RGBA, scale float64) {
	runes := []rune(s)
	total := 0.0
	for _, r := range runes {
		total += g.textW(string(r), scale) + gap
	}
	total -= gap
	x := math.Round(cx - total/2)
	for _, r := range runes {
		g.drawText(dst, string(r), x, math.Round(y), c, scale)
		x += math.Round(g.textW(string(r), scale) + gap)
	}
}
