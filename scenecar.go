package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The car of the cinematics (interlude 5 and its 2 player version). It is the same model as the
// cars of the game (vaSedanSide of the vehicle art kit, vehart.go) with the same finish as the prop
// sprites, only with two extras the scenes need: the driver's door opening and a notch on the rims
// so the wheels can be seen turning. Drawn at an integer scale (sceneCarK), nose to the right.
// ---------------------------------------------------------------------------

const (
	sceneCarK      = 3 // screen pixels per design pixel
	sceneCarW      = 68
	sceneCarH      = 38
	sceneCarDoors  = 8 // door positions: 0 closed .. sceneCarDoors open
	sceneCarSpinFr = 8 // wheel notch positions
)

// sceneCarBody: the paint of the scene car (orange, as it always was in this scene).
var sceneCarBody = c(0xd8742a)

var sceneCarCache = map[[2]int]*ebiten.Image{}

// sceneCarUse counts the draws (tests check that the scenes go through drawSceneCar).
var sceneCarUse int

// sceneCarPix paints the car (nose LEFT, like every side sprite) with the driver's door open door/
// sceneCarDoors and the wheel notch in frame spin (spin < 0: no notch, the car is parked).
func sceneCarPix(door, spin int) *pixBuf {
	b := newPix(sceneCarW, sceneCarH)
	b.ox, b.oy = 1, 1
	vaBuild(b, 66, func(t *pixBuf) { vaSedanSide(t, vaCar{body: sceneCarBody}) }, nil)
	if door > 0 {
		sceneCarDoor(b, float64(min(door, sceneCarDoors))/sceneCarDoors)
	}
	if spin >= 0 {
		// the car goes right once mirrored: in these (unmirrored) coordinates the wheels turn the other way
		a := -float64(spin%sceneCarSpinFr) * 2 * math.Pi / sceneCarSpinFr
		for _, cx := range []float64{14, 51} {
			b.set(int(math.Round(cx+math.Cos(a)*2.6)), int(math.Round(30+math.Sin(a)*2.6)), vaInk)
		}
	}
	return stylizeSprite(b.scaled(wk).detail(int64(len("car")*31)+int64('c')*977, 0.035).outlinedSel(0.45))
}

// sceneCarDoor: the front (driver's) door, hinged at the nose side (x 19), swung open by o (0..1):
// the dark inside and the seat show and the leaf narrows towards the hinge as it turns to the camera.
func sceneCarDoor(b *pixBuf, o float64) {
	inside, seat := c(0x241a14), c(0x6a4a3a)
	for y := 9; y <= 26; y++ {
		x0 := 19
		if y <= 15 {
			fx, _ := vaSedanEdges(y, false)
			x0 = max(x0, fx+2)
		}
		x1 := 33
		if y <= 15 {
			x1 = 31
		}
		vaRow(b, y, x0, x1, inside)
	}
	vaRow(b, 16, 19, 33, inside)
	if o > 0.05 {
		b.rect(21, 18, 11, 5, seat)
		b.rect(21, 18, 11, 1, shade(seat, 1.25))
	}
	lw := int(math.Round(15 * (1 - 0.85*o)))
	if lw < 2 {
		lw = 2
	}
	x1 := 19 + lw - 1
	for y := 9; y <= 15; y++ {
		fx, _ := vaSedanEdges(y, false)
		vaRow(b, y, max(19, fx+2), min(x1, 31), vaGlassC)
	}
	vaRow(b, 16, 19, x1, shade(sceneCarBody, 0.75))
	vaRow(b, 17, 19, x1, shade(sceneCarBody, 1.28))
	for y := 18; y <= 26; y++ {
		vaRow(b, y, 19, x1, sceneCarBody)
	}
	vaRow(b, 23, 19, x1, shade(sceneCarBody, 0.86))
	b.rect(x1, 9, 1, 18, vaInk) // the free edge of the leaf
	if lw > 5 {
		b.rect(x1-4, 19, 3, 1, c(0xe0e4ea)) // the handle
	}
}

// sceneCarImg is the cached picture of sceneCarPix.
func sceneCarImg(door, spin int) *ebiten.Image {
	door = max(0, min(door, sceneCarDoors))
	if spin >= 0 {
		spin %= sceneCarSpinFr
	} else {
		spin = -1
	}
	key := [2]int{door, spin}
	if im, ok := sceneCarCache[key]; ok {
		return im
	}
	im := sceneCarPix(door, spin).img()
	sceneCarCache[key] = im
	return im
}

// sceneCarOp places the car with its nose to the right: x is where the old car's tail started and
// y the line its wheels stand on (both in screen pixels).
func sceneCarOp(x, y float64) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-sceneCarK, sceneCarK)
	op.GeoM.Translate(math.Round(x)-10+sceneCarW*sceneCarK, math.Round(y)+3-(sceneCarH-1)*sceneCarK)
	return op
}

// drawSceneCar draws the scene car (door 0..1 open, spin < 0 parked) at x, y (see sceneCarOp).
func drawSceneCar(dst *ebiten.Image, x, y, door float64, spin int) {
	sceneCarUse++
	d := int(math.Round(door * sceneCarDoors))
	dst.DrawImage(sceneCarImg(d, spin), sceneCarOp(x, y))
}
