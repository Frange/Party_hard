package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The rocket interlude for two players: the same roof, the same guard and the same barrel with a
// toilet for a nose, which now has two seats. The blue one (patient) waits his turn and checks his
// list; the red one (nervous) searches all his pockets for a ticket he never had, offers a chewing
// gum, climbs in first and sits on the wrong seat, wants the countdown to go faster and is the first to say
// "I told you so" when it fizzles.
// ---------------------------------------------------------------------------

// rocketHeroPos2: where the red one is (feet) on the roof / in the barrel at time t.
func (g *Game) rocketHeroPos2(t float64) (x, y float64, inside bool) {
	hx, hy, in1 := g.rocketHeroPos(t)
	rest := -64 + 22*math.Sin(t*1.7)
	x, y = hx+rest, hy
	if t >= itBoard0 { // he rushes to the barrel ahead of the blue one and waits stamping
		to := rkX - 150.0
		u := introEase((t - itBoard0) / 1.5)
		x = (hx + rest) + (to-(hx+rest))*u
		if t < rkBoard {
			x += 8 * math.Sin(t*3) // fidgeting
		}
	}
	if t >= rkBoard { // up and into the second seat (the right one)
		u := math.Min(1, (t-rkBoard)/0.9)
		topY := rkBase - 14 - rkTop + 72
		fromX := rkX - 150.0
		x = fromX + (rkX+26-fromX)*introEase(u)
		y = (interGround + 12) + (topY-(interGround+12))*introEase(u) - math.Sin(math.Pi*u)*80
		inside = u >= 1
	}
	_ = in1
	return
}

func (g *Game) rocketHero2(c *ebiten.Image, in *interState, h, oy float64) {
	t := in.t
	cs := in.heroes[1]
	if cs == nil {
		return
	}
	x, y, _ := g.rocketHeroPos2(t)
	anim, ph, dir := AnimWalk, g.interPhase(in)*1.45, DirRight
	lean := 0.0
	if t < itBoard0 && t > 2.2 {
		lean = 0.1
	}
	if t >= itBoard0 {
		ph = t * 10
	}
	switch {
	case t >= rkAsk && t < rkBoard-0.2: // pockets, pockets, pockets...
		anim, ph, dir = AnimPanic, t*9, DirLeft
		lean = 0
	case t >= itBoard0+1.6 && t < rkAsk:
		anim, ph = AnimIdle, 0
	}
	if t >= rkBoard && t < rkBoard+0.9 {
		anim, ph = AnimRun, t*10
	}
	if t >= rkBoard+0.9 {
		anim, ph, dir = AnimIdle, 0, DirDown
	}
	bob := 0.0
	if t < itBoard0 {
		bob = -math.Abs(math.Sin(ph*math.Pi*2/float64(animFrames[AnimWalk]))) * 7
	}
	if t >= rkBoard+0.9 && t < rkFizzle { // his foot goes at twice the speed
		bob = -math.Abs(math.Sin(t*12)) * 3
	}
	if t >= rkFizzle && t < rkKick { // "I told you so": arms up
		anim, ph = AnimWave, t*8
	}
	tint := sceneTint(1, in.heroes[0], in.heroes[1])
	g.sceneChar(c, cs, anim, dir, ph, x, y-h+oy+bob, interScale, 1, 0, lean, tint)
	if t < itBoard0+0.8 && t > 2.2 {
		g.interAnger(c, x, y-h+oy+bob, math.Mod(t, 1.7)+0.2)
	}
}

// the two voices: the blue one calm, the red one hot (the guard and the first one's own lines stay as they were)
func (g *Game) rocketTwoSay(dst *ebiten.Image, in *interState) {
	t := in.t
	x2, y2, _ := g.rocketHeroPos2(t)
	h := rocketHeight(t)
	cam := rocketCam(t)
	// once he is in the barrel the bubble floats above the toilet and the heads (not on them), a bit to
	// the right so it clears the countdown digit
	say := func(s string) {
		g.speechBubble(dst, x2+10+40*b2f(t >= rkBoard), y2-h+cam-34*interScale-4+3*b2f(t >= rkBoard), s)
	}
	hx, hy, _ := g.rocketHeroPos(t)
	say1 := func(s string) { g.speechBubble(dst, hx+10, hy-h+cam-34*interScale-4+70*b2f(t >= rkBoard), s) }
	switch {
	case t > 2 && t < itBoard0:
		const per = 3.8
		k := int((t - 3.9) / per)
		if t >= 3.9 {
			if u := t - 3.9 - float64(k)*per; u < 1.5 {
				say(interTwoLines[1][k%len(interTwoLines[1])])
			}
		}
	case t >= rkAsk+0.3 && t < rkTicket+0.5:
		say("¡Lo tenía por aquí!")
	case t >= rkTicket+0.9 && t < rkBoard:
		say("¿Vale un chicle?")
	case t >= rkBoard+0.9 && t < rkBoard+2.0:
		say("¡Yo me pido delante!")
	case t >= rkCount0+0.2 && t < rkCount0+1.3:
		say("¡¡Más deprisa!!")
	case t >= rkFizzle+0.6 && t < rkFizzle+1.5:
		say1("Paciencia.")
	case t >= rkFizzle+0.05 && t < rkFizzle+0.55:
		say("¡¡LO SABÍA!!")
	case t >= rkKick+0.5 && t < rkCount1:
		say1("Otra vez. Con calma.")
	case t >= rkCount1+0.3 && t < rkCount1+1.6:
		say("¡¡Venga, venga!!")
	}
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

var _ = ebiten.NewImage
