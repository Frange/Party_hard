package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The two player version of the car interlude (the boat and the balloon have their own, see
// interlude_boat.go and interlude_balloon.go).
//
// The same angry walk, now with the two heroes and their characters (twoscenes.go): the blue one is
// the patient one: he walks upright at his own pace, never looks where he is going and always
// gets away with it, reads the map (upside down) and rows calmly; the red one is the nervous one:
// he stomps, runs ahead, kicks the first thing he sees, walks into a lamp post, is afraid of a
// little dog and wants to drive. In the car they fight over the wheel; in the boat the red one
// is seasick over a bucket while the blue one rows with an oar although the boat has a motor.
// ---------------------------------------------------------------------------

// the little accidents of the street, in seconds of the scene
const (
	i2CanT  = 2.3 // the blue one walks past a tin can without looking...
	i2CanKT = 3.1 // ...and the red one kicks it
	i2LampT = 5.6 // a lamp post the red one walks into while he looks back at the blue one
)

// interTwoLines: what each of them mutters in the street (the first is the patient one).
var interTwoLines = [2][]string{
	{"Otra fiesta más.", "Con calma. Siempre hay un siguiente.", "Respira, que es gratis.", "Todo se arregla andando."},
	{"¡¡Qué harta me tiene!!", "¡Ya vale de fiestas!", "¡Yo así no aguanto!", "¡Más rápido, que llegamos tarde!"},
}

// interTwoOffset: where the red one is relative to the blue one (pixels; negative = behind). He is
// restless: he lags, rushes, comes back; and he has his own business with the can, the lamp and the dog.
func interTwoOffset(t float64) float64 {
	off := -64 + 22*math.Sin(t*1.7)
	// he wants to kick the can: he rushes to it...
	off += 40 * (introEase((t-(i2CanKT-1.2))/0.8) - introEase((t-(i2CanKT+0.6))/0.8))
	// ...he overtakes the blue one to see whether he follows (and hits the lamp post)
	off += 150 * (introEase((t-(i2LampT-2.4))/1.2) - introEase((t-(i2LampT+1.6))/1.0))
	return off
}

// interTwoGagX: the screen x of a thing of the street that is at the blue one's x at time t0 (it comes
// from the right and goes by with the street) and 'ahead' pixels in front of him.
func interTwoGagX(t, t0, ahead float64) float64 {
	return interHeroX + ahead - (interScroll(t) - interScroll(t0))
}

// interTwoHero2 is where the red one stands (and how he moves) at time t, as the actor of the scene.
func (g *Game) interTwoHero2(in *interState) (x, y float64, anim int, ph float64, lean float64, hop float64) {
	t := in.t
	hx, hy := g.interHeroPos(in)
	x, y = hx+interTwoOffset(math.Min(t, itBoard0)), hy
	anim, ph = AnimWalk, g.interPhase(in)*1.45 // out of step, and faster
	if t > 2.2 && t < itDecel0 {
		lean = 0.1
	}
	if t >= itBoard0 { // he rushes to the vehicle ahead of the blue one
		to := interStopX + 100
		u := introEase((t - itBoard0) / (itOpen - itBoard0 - 0.3))
		x = (hx + interTwoOffset(itBoard0)) + (to-(hx+interTwoOffset(itBoard0)))*u
		ph = t * 10
		lean = 0
		if t > itOpen+0.5 {
			anim, ph = AnimIdle, 0
		}
	}
	// the kick: a leg swing as he reaches the can
	if t > i2CanKT-0.25 && t < i2CanKT+0.35 {
		anim, ph = AnimPunch, (t-(i2CanKT-0.25))*10
	}
	// the lamp post: he stops dead
	if t > i2LampT+0.3 && t < i2LampT+1.6 {
		anim, ph = AnimFall, 0
	}
	return
}

// the dog sits on the pavement and barks at the red one; the blue one pats it
func (g *Game) interTwoStreetGags(c *ebiten.Image, in *interState) {
	t := in.t
	if t > itBoard0 {
		return
	}
	gy := interGround + 14
	// the tin can: the blue one steps round it; the red one kicks it up the street
	if t < i2CanKT+2.5 {
		x := interTwoGagX(t, i2CanT, 60)
		y := gy
		if t >= i2CanKT {
			u := t - i2CanKT
			x += 200*u + 40*u*u
			y = gy - math.Max(0, 120*u-170*u*u)
			if y > gy {
				y = gy
			}
		}
		fillRect(c, x-9, y-26, 18, 26, rgb(0x6a6e7a))
		fillRect(c, x-11, y-30, 22, 5, rgb(0x8a8e9a))
		fillRect(c, x-9, y-18, 18, 2, rgb(0x4a4e5a))
		if t >= i2CanKT && t < i2CanKT+0.6 {
			g.sceneBang(c, "¡CLANG!", x, y-44, (t-i2CanKT)/0.6, 0xffe060, 1.8)
		}
	}
	// the lamp post: it is where the red one runs ahead to
	if t > i2LampT-5 && t < i2LampT+5 {
		bx, _ := g.interHeroPos(in)
		x := interTwoGagX(t, i2LampT+1.0, 150) + (bx - interHeroX)
		fillRect(c, x-3, interGround-122, 7, 136, rgb(0x2a2a36))
		fillRect(c, x-22, interGround-126, 30, 6, rgb(0x2a2a36))
		fillRect(c, x-20, interGround-121, 18, 3, rgb(0xfff0b0))
		if t > i2LampT+0.3 && t < i2LampT+1.5 {
			g.sceneBang(c, "¡BONG!", x, interGround-150, (t-i2LampT-0.3)/1.2, 0xff8a6a, 2)
			ax, ay, _, _, _, _ := g.interTwoHero2(in)
			g.sceneStars(c, ax, ay-34*interScale-6, t, 1)
		}
	}
}

// the speech of the street: one of them at a time, each in his own voice
func (g *Game) interTwoSay(dst *ebiten.Image, in *interState) bool {
	t := in.t
	if t < 2 {
		return true
	}
	hx, hy := g.interHeroPos(in)
	rx, ry, _, _, _, _ := g.interTwoHero2(in)
	say := func(x, y float64, s string) { g.speechBubble(dst, x+10, y-34*interScale-4, s) }
	switch {
	case t < itBoard0:
		blue := [][2]float64{{1.3, 0}, {7.0, 1}}
		red := [][2]float64{{4.2, 0}}
		for _, b := range blue {
			if sceneIn(t, b[0], b[0]+1.7) && !sceneIn(t, i2CanKT-0.5, i2CanKT+1.2) {
				say(hx, hy, interTwoLines[0][int(b[1])])
				return true
			}
		}
		switch {
		case sceneIn(t, i2CanT, i2CanT+0.9):
			say(hx, hy, "Ni lo miro.")
		case sceneIn(t, i2CanKT+0.5, i2CanKT+1.5):
			say(rx, ry, "¡Toma lata!")
		case sceneIn(t, i2LampT-0.9, i2LampT+0.1):
			say(rx, ry, "¿Venís o no venís?")
		case sceneIn(t, i2LampT+1.5, i2LampT+2.5):
			say(hx, hy, "Te lo dije.")
		}
		for _, r := range red {
			if sceneIn(t, r[0], r[0]+1.5) && !sceneIn(t, i2CanKT-0.5, i2LampT+3.2) {
				say(rx, ry, interTwoLines[1][int(r[1])])
				return true
			}
		}
	}
	return true
}

// interDrawHero2 draws the red one (at 4x like the blue one).
func (g *Game) interDrawHero2(c *ebiten.Image, in *interState, alpha float64) {
	if alpha <= 0 || in.heroes[1] == nil {
		return
	}
	x, y, anim, ph, lean, hop := g.interTwoHero2(in)
	tint := sceneTint(1, in.heroes[0], in.heroes[1])
	g.sceneChar(c, in.heroes[1], anim, DirRight, ph, x, y, interScale, alpha, hop, lean, tint)
	if in.t < itBoard0 && in.t > 2.2 { // steam, as the blue one has: he fumes more
		g.interAnger(c, x, y, math.Mod(in.t, 1.7)+0.2)
	}
}

// the two heads in the windows of the car: the red one at the wheel, the blue one with the map
func (g *Game) interTwoCarHeads(c *ebiten.Image, in *interState, x, y float64) {
	t := in.t
	// the blue one, in the back seat window, behind an open map held upside down
	filledEllipse(c, x+92, y-84, 8, 9, rgb(0xe0b088))
	fillRect(c, x+85, y-94, 14, 5, rgb(0x14141a))
	fillRect(c, x+70, y-80, 26, 20, rgb(0xf0ead0)) // the map
	fillRect(c, x+70, y-80, 26, 2, rgb(0xc8c0a0))
	fillRect(c, x+78, y-72, 2, 8, rgb(0xd02a2a)) // a red cross and a road
	fillRect(c, x+76, y-70, 6, 2, rgb(0xd02a2a))
	fillRect(c, x+84, y-76, 8, 1, rgb(0x6a6a74))
	g.drawText(c, "N", x+86, y-66, rgb(0x2a2a3a), 0.8) // north, at the bottom: upside down
	// the red one at the wheel, shaking
	jx := math.Round(math.Sin(t*31) * 2)
	filledEllipse(c, x+128+jx, y-84, 8, 9, rgb(0xe0b088))
	fillRect(c, x+121+jx, y-94, 14, 5, rgb(0x14141a))
	fillRect(c, x+120+jx, y-66, 18, 4, rgb(0xc42a2a)) // his red shoulder
	fillRect(c, x+138, y-74, 2, 12, rgb(0x14141a))    // the wheel
	if t < itDepart {
		g.sceneStars(c, x+128, y-100, t, 0.9)
	}
}

// boarding: who says what (the red one wants the wheel, the blue one has the map)
func (g *Game) interTwoBoardSay(dst *ebiten.Image, in *interState) {
	t := in.t
	vx := interVehicleX(t)
	x2, y2, _, _, _, _ := g.interTwoHero2(in)
	hx, hy := g.interHeroPos(in)
	switch {
	case sceneIn(t, itBoard0+0.1, itBoard0+0.9):
		g.speechBubble(dst, x2+10, y2-34*interScale-4, "¡¡Yo conduzco!!")
	case sceneIn(t, itBoard0+0.9, itOpen+0.4):
		g.speechBubble(dst, hx+10, hy-34*interScale-4, "Yo tengo el mapa.")
	case sceneIn(t, itOpen+0.5, itOpen+1.2):
		g.speechBubble(dst, vx+105, interGround-90, "¡Quita las manos!")
	case sceneIn(t, itOpen+1.2, itDepart-0.1):
		g.speechBubble(dst, vx+92, interGround-90, "El mapa está del revés")
	}
}
