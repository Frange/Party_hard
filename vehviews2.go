package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Front and rear views of the special vehicles that are not plain road traffic:
// the tank, the tow truck and the horse with its rider. Same mechanism as
// vehviews.go: on a vertical road they come down seen head on and go up seen
// from behind. The tow truck also gets a proper side view (it used to be a
// yellow car).
// ---------------------------------------------------------------------------

// roadSpecials are the vehicles that are drawn by hand instead of coming from propKinds.
var roadSpecials = map[string]bool{"tank": true, "tow": true, "horse": true}

func init() {
	reg := func(name string, w, h int, f func(b *pixBuf, v int, front bool)) {
		vehViewDefs[name] = &vehViewDef{w: w, h: h, draw: f}
	}
	reg("tank", 42, 36, func(b *pixBuf, v int, front bool) { tankView(b, front) })
	reg("tow", 34, 40, func(b *pixBuf, v int, front bool) { towView(b, front) })
	reg("horse", 30, 56, func(b *pixBuf, v int, front bool) { horseView(b, v, front, true) })
}

func tankView(b *pixBuf, front bool) {
	w := 42
	b.ellipse(21, 33.5, 20, 3, color.RGBA{0, 0, 0, 90})
	track, link := c(0x2a3a22), c(0x14180e)
	body, hi, lo := c(0x4a6a3a), c(0x6a8a52), c(0x3a5a2c)
	for _, x := range []int{0, w - 8} {
		b.rect(x, 14, 8, 20, track)
		b.rect(x+1, 14, 1, 20, c(0x3c4e32))
		for y := 16; y < 33; y += 4 {
			b.rect(x+2, y, 4, 2, link)
		}
	}
	b.rect(7, 16, w-14, 17, body)
	b.rect(7, 16, w-14, 2, hi)
	b.rect(7, 31, w-14, 2, lo)
	if front {
		b.rect(9, 20, w-18, 8, shade(body, 1.14)) // the sloped front plate
		b.rect(9, 20, w-18, 1, hi)
		b.rect(10, 29, 5, 2, vvLight)
		b.rect(w-15, 29, 5, 2, vvLight)
		b.rect(18, 24, 6, 2, c(0x1c2414)) // the driver's slit
	} else {
		b.rect(9, 19, w-18, 11, c(0x35502a)) // engine deck
		for x := 11; x < w-11; x += 3 {
			b.rect(x, 21, 1, 8, c(0x1c2a18))
		}
		b.ellipse(11, 31, 2.2, 2.2, link) // the exhausts
		b.ellipse(float64(w-11), 31, 2.2, 2.2, link)
		b.rect(w/2-2, 31, 4, 2, vvTail)
	}
	// the turret
	b.ellipse(21, 11, 13, 8, body)
	b.ellipse(21, 9, 11, 5.5, hi)
	b.ellipse(21, 13, 12, 4, lo)
	if front {
		b.rect(14, 7, 14, 9, lo) // the gun mantlet
		b.ellipse(21, 11.5, 3.6, 3.6, link)
		b.ellipse(21, 11.5, 1.8, 1.8, c(0x050806))
		b.set(19, 9, c(0x6a7a58))
	} else {
		b.rect(12, 9, 18, 8, c(0x35502a)) // the turret bustle with the kit stowed on it
		b.rect(14, 10, 14, 2, hi)
		b.rect(15, 13, 5, 3, c(0x6a5a3a))
		b.rect(23, 13, 5, 3, c(0x8a8a6a))
		b.rect(33, 0, 1, 12, c(0x1c2414)) // the aerial
	}
	b.ellipse(29, 6.5, 2.6, 2, c(0x1c2414)) // the commander's hatch
}

func towView(b *pixBuf, front bool) {
	w := 34
	yel, yelHi, yelLo := c(0xe8b820), c(0xf8d850), c(0xb88810)
	vvShadow(b, w)
	vvWheels(b, w, 24, 38)
	if front {
		b.rect(5, 14, w-10, 18, yel) // cab
		b.rect(5, 14, w-10, 2, yelHi)
		b.rect(5, 30, w-10, 2, yelLo)
		b.rect(8, 17, w-16, 7, vvGlass)
		b.rect(w/2-1, 17, 2, 7, shade(vvGlass, 0.7))
		// the beacon bar on the roof
		b.rect(8, 11, w-16, 3, c(0x3a3a44))
		b.rect(9, 11, 5, 3, c(0xff9a20))
		b.rect(w-14, 11, 5, 3, c(0xff9a20))
		b.rect(w/2-2, 11, 4, 3, c(0xe03030))
		b.rect(6, 26, 5, 3, vvLight)
		b.rect(w-11, 26, 5, 3, vvLight)
		b.rect(10, 29, w-20, 3, vvGrille)
		b.rect(4, 33, w-8, 3, c(0x8a8a96)) // the bumper
		vvPlateAt(b, w, 33)
		return
	}
	// seen from behind: the flat bed, the winch post and the hook
	b.rect(4, 14, w-8, 4, yel) // the back of the cab above the bed
	b.rect(8, 15, w-16, 2, vvGlass)
	b.rect(2, 18, w-4, 13, c(0x6a6a76)) // bed
	b.rect(2, 18, w-4, 2, c(0x9a9aa6))
	for x := 4; x < w-6; x += 8 { // hazard stripes
		b.rect(x, 21, 4, 3, c(0xe8b820))
		b.rect(x+4, 21, 4, 3, c(0x2a2a30))
	}
	b.rect(w/2-1, 2, 2, 17, c(0x3a3a44)) // winch post
	b.rect(w/2-4, 2, 8, 2, c(0x3a3a44))
	b.line(w/2, 4, w/2, 20, c(0xc8c8d4)) // chain
	b.rect(w/2-1, 20, 3, 3, c(0xe03030)) // hook
	b.rect(3, 26, 4, 3, vvTail)
	b.rect(w-7, 26, 4, 3, vvTail)
	b.rect(2, 31, w-4, 2, yelLo)
	vvPlateAt(b, w, 33)
}

func horseView(b *pixBuf, v int, front bool, withRider bool) {
	cc := horseCoats[v%len(horseCoats)]
	body, dark, light, mane := cc.body, cc.dark, cc.light, cc.mane
	shirt, skin, hat := c(0xc84a3a), c(0xe8b890), c(0xa8803a)
	b.ellipse(15, 53, 12, 3, color.RGBA{0, 0, 0, 90})
	rider := func() {
		if !withRider {
			return
		}
		b.rect(10, 9, 10, 14, shirt) // torso
		b.rect(10, 9, 10, 2, shade(shirt, 1.25))
		b.rect(7, 11, 3, 11, shade(shirt, 0.85)) // arms
		b.rect(20, 11, 3, 11, shade(shirt, 0.85))
		b.rect(7, 21, 3, 2, skin)
		b.rect(20, 21, 3, 2, skin)
		b.ellipse(15, 6, 4, 3.6, skin) // head
		b.rect(12, 0, 6, 3, hat)       // hat
		b.rect(9, 3, 12, 2, shade(hat, 0.82))
	}
	hoof := func(x, y int) { b.rect(x-1, y, 5, 3, c(0x1a1410)) }
	if front {
		rider()
		// forelegs
		for _, x := range []int{9, 18} {
			b.rect(x, 38, 3, 15, shade(body, 0.85))
			hoof(x, 52)
		}
		// chest and neck
		b.ellipse(15, 38, 10.5, 9, body)
		b.ellipse(15, 41, 8, 5, shade(body, 0.9))
		// neck tapering from a narrow poll to the broad chest
		for y := 20; y < 36; y++ {
			hw := 3 + (y-20)*3/15 // half width 3 -> 6
			b.rect(15-hw, y, 2*hw, 1, body)
			b.rect(15-hw-1, y, 1, 1, mane)
			b.rect(15+hw, y, 1, 1, mane)
		}
		// head seen head on: long and narrow, hanging in front of the neck
		b.ellipse(15, 30, 3.3, 8, body)
		b.rect(14, 24, 2, 13, light) // the blaze
		b.ellipse(15, 37, 2.6, 3, shade(light, 0.95))
		b.set(14, 37, dark)
		b.set(16, 37, dark)
		b.set(12, 28, c(0x14100c))
		b.set(18, 28, c(0x14100c))
		b.rect(12, 21, 2, 3, body) // small ears, close together
		b.rect(16, 21, 2, 3, body)
		b.rect(12, 21, 1, 2, dark)
		b.rect(17, 21, 1, 2, dark)
		b.rect(13, 22, 4, 2, mane) // forelock
		return
	}
	// seen from behind: rump, tail and hind legs
	for _, x := range []int{8, 19} {
		b.rect(x, 40, 3, 13, shade(body, 0.85))
		hoof(x, 52)
	}
	// rump kept no wider than the chest seen head on (10.5) so it matches the side view
	b.ellipse(15, 36, 9.5, 9, body)
	b.ellipse(15, 33, 7.5, 5, light)
	b.ellipse(15, 41, 7.5, 3.5, shade(body, 0.85))
	b.rect(14, 29, 2, 10, shade(body, 0.7)) // the dip between the haunches
	rider()
	if withRider {
		b.rect(5, 22, 4, 11, c(0x3a4a7a)) // the rider's legs
		b.rect(21, 22, 4, 11, c(0x3a4a7a))
		b.rect(5, 31, 4, 3, c(0x2a1c14))
		b.rect(21, 31, 4, 3, c(0x2a1c14))
	}
	// the tail
	for y := 28; y < 50; y++ {
		sw := int(math.Round(math.Sin(float64(y)*0.35) * 1.2))
		b.rect(13+sw, y, 4, 1, mane)
	}
}

// ---- the tow truck seen from the side ----------------------------------------------

var towSideCache *PropImg

// towSide: the tow truck in profile, nose to the left like the cars (the wreck it carries is
// drawn separately, on its bed). The anchor is the middle of its base, like every vehicle sprite.
func towSide() PropImg {
	if towSideCache != nil {
		return *towSideCache
	}
	const w, h = 58, 32
	b := newPix(w+2, h+2)
	b.ox, b.oy = 1, 1
	yel, yelHi, yelLo := c(0xe8b820), c(0xf8d850), c(0xb88810)
	b.ellipse(w/2, 30, w/2-1, 3, color.RGBA{0, 0, 0, 80})
	// the cab (left)
	b.rect(1, 14, 18, 14, yel)
	b.rect(1, 14, 18, 2, yelHi)
	b.rect(1, 26, 18, 2, yelLo)
	b.rect(3, 8, 11, 8, yel) // cab roof / upper part
	b.rect(3, 8, 11, 2, yelHi)
	b.rect(4, 10, 8, 5, vvGlass) // windscreen
	b.rect(1, 21, 2, 3, vvLight) // headlight
	b.rect(0, 25, 3, 3, c(0x8a8a96))
	b.rect(5, 5, 7, 3, c(0xff9a20)) // beacon
	b.rect(7, 4, 3, 1, c(0xe03030))
	// the flat bed
	b.rect(19, 20, 38, 6, c(0x6a6a76))
	b.rect(19, 20, 38, 1, c(0x9a9aa6))
	b.rect(19, 24, 38, 2, c(0x4a4a56))
	for x := 22; x < 55; x += 8 {
		b.rect(x, 21, 4, 2, yel)
		b.rect(x+4, 21, 4, 2, c(0x2a2a30))
	}
	// the winch post and the cable, leaning back over the bed
	b.line(48, 20, 42, 6, c(0x3a3a44))
	b.line(49, 20, 43, 6, c(0x3a3a44))
	b.line(42, 6, 33, 9, c(0x3a3a44))
	b.line(33, 9, 33, 18, c(0xc8c8d4))
	b.rect(32, 18, 3, 3, c(0xe03030))
	b.rect(55, 21, 2, 3, vvTail)
	// the wheels
	for _, x := range []float64{10, 28, 47} {
		b.ellipse(x, 27, 4.6, 4.6, vvDark)
		b.ellipse(x, 27, 2, 2, c(0x8a8a96))
	}
	sb := stylizeSprite(b.scaled(wk).detail(7001, 0.035).outlinedSel(0.45))
	pi := PropImg{img: sb.img(), ox: float64(w/2+1) * wk, oy: float64(h-1) * wk}
	towSideCache = &pi
	return pi
}

// ---- the free horse (the animal that walks about the yards) seen head on / from behind ---------

var horseFreeCache = map[string]PropImg{}

// horseFreeView: a horse without a rider, a bit bigger than the ridden one's body so that it matches
// the side sprite of the animal. coat picks the colour; anchored at the middle of its hooves.
func horseFreeView(coat int, front bool) PropImg {
	key := string(rune('0'+coat%3)) + map[bool]string{true: "f", false: "b"}[front]
	if pi, ok := horseFreeCache[key]; ok {
		return pi
	}
	b := newPix(32, 58)
	b.ox, b.oy = 1, 1
	horseView(b, coat, front, false)
	const k = 1.3
	sb := stylizeSprite(b.scaled(k*wk).detail(int64(8800+coat), 0.035).outlinedSel(0.45))
	pi := PropImg{img: sb.img(), ox: float64(16) * k * wk, oy: float64(57) * k * wk}
	horseFreeCache[key] = pi
	return pi
}
