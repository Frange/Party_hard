package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The two player version of the car and boat interludes.
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
	i2CanT  = 7.0  // the blue one walks past a tin can without looking...
	i2CanKT = 7.78 // ...and the red one kicks it
	i2LampT = 12.5 // a lamp post the red one walks into while he looks back at the blue one
	i2DogT  = 16.0 // a dog: the red one runs away from it, the blue one pats it
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
	// the dog: he jumps back
	off -= 95 * (introEase((t-i2DogT)/0.4) - introEase((t-(i2DogT+1.8))/0.9))
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
		if in.kind == interBoat {
			to = interStopX - 70
		}
		u := introEase((t - itBoard0) / (itOpen - itBoard0 - 0.3))
		x = (hx + interTwoOffset(itBoard0)) + (to-(hx+interTwoOffset(itBoard0)))*u
		ph = t * 10
		lean = 0
		if in.kind == interBoat && t > itOpen-0.2 { // a jump into the boat that is too big
			v := math.Min(1, (t-(itOpen-0.2))/(itIn-itOpen+0.2))
			x += 95 * v
			y += 14*v - math.Sin(math.Pi*v)*90
		}
		if t > itOpen+0.5 && in.kind != interBoat {
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
	// the dog: a panicky shuffle
	if t > i2DogT && t < i2DogT+1.8 {
		anim, ph = AnimPanic, t*9
	}
	return
}

// the dog sits on the pavement and barks at the red one; the blue one pats it
func (g *Game) interTwoStreetGags(c *ebiten.Image, in *interState) {
	t := in.t
	if t > 30 || in.kind == interBoat && t > itDecel0 {
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
	// the dog
	if t > i2DogT-5 && t < i2DogT+5 {
		x := interTwoGagX(t, i2DogT, 40)
		wag := math.Sin(t*18) * 3
		filledEllipse(c, x, gy-12, 16, 10, rgb(0x8a6a44))
		filledEllipse(c, x+16, gy-20, 9, 8, rgb(0x8a6a44))
		fillRect(c, x+22, gy-20, 8, 5, rgb(0x6a4a2c))
		fillRect(c, x+20, gy-27, 4, 7, rgb(0x4a3220))
		fillRect(c, x+17, gy-23, 2, 2, rgb(0x14100c))
		fillRect(c, x-20+wag, gy-18, 12, 4, rgb(0x8a6a44)) // the tail
		for _, lx := range []float64{-8, -2, 8, 14} {
			fillRect(c, x+lx, gy-4, 4, 6, rgb(0x6a4a2c))
		}
		if t > i2DogT && t < i2DogT+1.2 {
			g.sceneBang(c, "¡GUAU!", x+30, gy-48, (t-i2DogT)/1.2, 0xffe060, 1.8)
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
		blue := [][2]float64{{3.0, 0}, {9.0, 1}, {14.2, 2}, {19.0, 3}}
		red := [][2]float64{{5.4, 0}, {10.8, 1}, {17.6, 2}, {21.5, 3}}
		for _, b := range blue {
			if sceneIn(t, b[0], b[0]+2.3) && !sceneIn(t, i2CanKT-0.5, i2CanKT+1.2) {
				say(hx, hy, interTwoLines[0][int(b[1])])
				return true
			}
		}
		switch {
		case sceneIn(t, i2CanT, i2CanT+0.9):
			say(hx, hy, "Ni lo miro.")
		case sceneIn(t, i2CanKT+0.5, i2CanKT+1.7):
			say(rx, ry, "¡Toma lata!")
		case sceneIn(t, i2LampT-0.6, i2LampT+0.3):
			say(rx, ry, "¿Venís o no venís?")
		case sceneIn(t, i2LampT+1.7, i2LampT+3.0):
			say(hx, hy, "Te lo dije.")
		case sceneIn(t, i2DogT+0.2, i2DogT+1.6):
			say(rx, ry, "¡¡QUITA, QUITA!!")
		case sceneIn(t, i2DogT+1.6, i2DogT+3.0):
			say(hx, hy, "Buen chico.")
		}
		for _, r := range red {
			if sceneIn(t, r[0], r[0]+2.3) && !sceneIn(t, i2CanKT-0.5, i2LampT+3.2) && !sceneIn(t, i2DogT-0.5, i2DogT+3.2) {
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
	if in.kind == interBoat && in.t > itIn { // seasick: green in the face and all over
		tint = [3]float32{0.62, 1, 0.62}
	}
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

// the boat: an oar the blue one rows with calmly and a bucket the red one is sick in
func (g *Game) interTwoBoatProps(c *ebiten.Image, in *interState, x, y float64) {
	t := in.t
	if t < itIn {
		return
	}
	// the oar: it goes through the water on the beat of nothing in particular
	sw := math.Sin(t * 2.2)
	fillRect(c, x+60+sw*6, y-46, 3, 50, rgb(0x8a6a44))
	fillRect(c, x+54+sw*10, y-2, 16, 6, rgb(0x6a4a2c))
	// the bucket under the red one's face
	fillRect(c, x+168, y-46, 18, 14, rgb(0x6a6e7a))
	fillRect(c, x+166, y-48, 22, 4, rgb(0x8a8e9a))
	if int(t*2)%2 == 0 {
		g.drawText(c, "buaaag", x+150, y-96, rgba(0x9ae09a, 230), 1.1)
	}
}

// boarding: who says what (the red one wants the wheel, the blue one has the map)
func (g *Game) interTwoBoardSay(dst *ebiten.Image, in *interState) {
	t := in.t
	vx := interVehicleX(t)
	x2, y2, _, _, _, _ := g.interTwoHero2(in)
	hx, hy := g.interHeroPos(in)
	switch {
	case sceneIn(t, itBoard0+0.2, itBoard0+1.6):
		g.speechBubble(dst, x2+10, y2-34*interScale-4, "¡¡Yo conduzco!!")
	case sceneIn(t, itBoard0+1.6, itOpen+0.6):
		g.speechBubble(dst, hx+10, hy-34*interScale-4, "Yo tengo el mapa.")
	case in.kind != interBoat && sceneIn(t, itOpen+0.8, itOpen+2.2):
		g.speechBubble(dst, vx+105, interGround-90, "¡Quita las manos!")
	case in.kind != interBoat && sceneIn(t, itOpen+2.2, itDepart-0.1):
		g.speechBubble(dst, vx+92, interGround-90, "El mapa está del revés")
	case in.kind == interBoat && sceneIn(t, itOpen+0.6, itOpen+2.0):
		g.speechBubble(dst, vx+60, interGround-70, "Remo, que es gratis.")
	case in.kind == interBoat && sceneIn(t, itOpen+2.0, itDepart-0.1):
		g.speechBubble(dst, vx+168, interGround-80, "¡Uuugh! Mareo...")
	}
}
