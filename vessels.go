package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Every boat and spaceship of the game is painted here, in one pixel-art style:
// a dark ink outline, flat shaded bands, blue windows, a cyan canopy and neon
// lights on the ships. The same few drawers are shared by all their uses:
//   - genBoat / genBoatView: the launches of the cruise and of Isla Nublada
//     (police, military with its machine gun, ambulance, fireboat, yacht...);
//   - genShip / genShipView: the little spaceships of the space station and the
//     shuttle of the arrival at Isla Nublada (arrival_travel.go arrShip);
//   - genFerry: the ferry moored at the pier of Isla Nublada (jp_boat);
//   - genPirateShip / genHorizonShip: the ships at sea of the beach levels (seasky.go);
//   - genRocket: the rocket that now and then crosses the sky (events.go).
// The sizes are the ones the old sprites had, so anchors, lanes and hitboxes stay.
// ---------------------------------------------------------------------------

// vInk: the outline of every vessel.
var vInk = rgb(0x14101a)

// craft styles: the extras each livery carries (light bar, gun, red cross...).
const (
	crPlain = iota
	crPolice
	crFBI
	crMilitary
	crSWAT
	crRangers
	crMedic
	crFire
	crTech
	crKids
	crHearse
	crLimo
	crCount
)

// transitCraft: the style of a vehicle of a boat or space level.
func transitCraft(v *Vehicle) int {
	switch v.kind {
	case 0:
		switch v.unit {
		case unitFBI:
			return crFBI
		case unitMilitary:
			return crMilitary
		case unitSWAT:
			return crSWAT
		case unitRangers:
			return crRangers
		}
		return crPolice
	case 1:
		return crMedic
	case 4:
		return crHearse
	case 6:
		return crLimo
	case 7:
		return crFire
	case 8:
		return crTech
	case 9:
		return crKids
	}
	return crPlain
}

// craftLights: the two colours of the light bar of a style (zero alpha: none).
func craftLights(style int) (a, b color.RGBA) {
	switch style {
	case crPolice, crRangers:
		return rgb(0x3a7aff), rgb(0xff3a3a)
	case crFBI, crSWAT:
		return rgb(0x3a7aff), rgb(0x3a7aff)
	case crMedic:
		return rgb(0x3a9aff), rgb(0xff3a3a)
	case crFire:
		return rgb(0xff3a3a), rgb(0xff3a3a)
	case crTech:
		return rgb(0xffa020), rgb(0xffa020)
	case crHearse:
		return rgb(0xb05aff), rgb(0xb05aff)
	}
	return color.RGBA{}, color.RGBA{}
}

// openCraft: the styles that sail an open speedboat (a wind-screen and the crew) instead of a cabin boat.
func openCraft(style int) bool {
	return style == crMilitary || style == crSWAT || style == crFBI || style == crKids
}

// ---- small shared parts --------------------------------------------------------------

// vUnder paints c at (x, y) only where the sprite is still transparent (shadows and foam go under the outline).
func vUnder(b *pixBuf, x, y int, c color.RGBA) {
	if b.get(x, y).A == 0 {
		b.set(x, y, c)
	}
}

// vWater: the soft shadow on the water and the foam of the wake along row y, from x0 to x1.
func vWater(b *pixBuf, x0, x1, y int) {
	for x := x0; x <= x1; x++ {
		vUnder(b, x, y, color.RGBA{10, 30, 60, 90})
		if x%5 == 0 {
			vUnder(b, x, y, c(0xe8f4ff))
		}
	}
	for x := x0 + 3; x <= x1-3; x += 3 {
		vUnder(b, x, y+1, color.RGBA{10, 30, 60, 60})
	}
}

// vWindow: a blue pane with its glint.
func vWindow(b *pixBuf, x, y, w, h int) {
	b.rect(x, y, w, h, c(0x2c64b0))
	b.rect(x, y, w, 1, c(0x8ad0f4))
	b.set(x, y+1, c(0x8ad0f4))
	if w > 3 {
		b.rect(x+w-1, y+1, 1, h-1, c(0x1c4280))
	}
}

// vFlag: a pennant at the top of a mast at (x, y), waving (frame 0 or 1), pointing back (left).
func vFlag(b *pixBuf, x, y int, col color.RGBA, frame int) {
	b.rect(x-4, y, 4, 3, col)
	b.rect(x-4, y, 4, 1, shade(col, 1.25))
	if frame == 0 {
		b.set(x-5, y+1, col)
		b.set(x-6, y+2, col)
	} else {
		b.set(x-5, y, col)
		b.set(x-6, y+1, col)
	}
}

// vRail: a guard rail along the deck, posts every 4 px.
func vRail(b *pixBuf, x0, x1, y int, col color.RGBA) {
	b.rect(x0, y, x1-x0, 1, col)
	for x := x0; x < x1; x += 4 {
		b.rect(x, y+1, 1, 2, shade(col, 0.8))
	}
}

// vCross: a red cross on a white field, 7 px square.
func vCross(b *pixBuf, x, y int) {
	b.rect(x, y, 7, 7, c(0xf8f8fc))
	b.rect(x+2, y+1, 3, 5, c(0xd82a2a))
	b.rect(x+1, y+2, 5, 3, c(0xd82a2a))
}

// hullSide paints a hull facing right from the stern x0 to the bow tip x1, deck line at top and keel
// at bot. The bow is raked over bowL px and the sheer rises sh px towards it. Below the deck: a light
// edge, the trim stripe, the body and the dark boot-top on the waterline.
func hullSide(b *pixBuf, x0, x1, top, bot, bowL, sh int, body, trim, boot color.RGBA) {
	for x := x0; x <= x1; x++ {
		f := 0.0
		if bowL > 0 && x > x1-bowL {
			f = float64(x-(x1-bowL)) / float64(bowL)
		}
		t := top - int(math.Round(float64(sh)*f*f))
		bm := bot - int(math.Round(float64(bot-t-1)*math.Pow(f, 1.4)))
		for y := t; y <= bm; y++ {
			col := body
			switch {
			case y == t:
				col = shade(body, 1.35)
			case y >= top+3 && y <= top+4:
				col = trim
			case y >= bot-2:
				col = boot
			case y == bot-3:
				col = shade(body, 0.8)
			}
			if x == x0 && y > t {
				col = shade(col, 0.82)
			}
			b.set(x, y, col)
		}
	}
}

// ---- boats ----------------------------------------------------------------------------

// boatBoot: the antifouling colour under the waterline.
func boatBoot(body color.RGBA) color.RGBA {
	if vLum(body) < 60 {
		return c(0x8a2a2a)
	}
	return shade(body, 0.45)
}

// lum: rough brightness of a colour (0..255).
func vLum(c color.RGBA) float64 { return 0.3*float64(c.R) + 0.59*float64(c.G) + 0.11*float64(c.B) }

// boatWall: the colour of the superstructure: a white tinted with the livery.
func boatWall(body color.RGBA) color.RGBA { return mixc(body, c(0xf6f6fa), 0.78) }

// genBoat paints a launch facing right, 90 px wide (112 for the long one), waterline on row 34.
// frame (0/1) waves the flag.
func genBoat(body, trim color.RGBA, long bool, style, frame int) *pixBuf {
	w := 90
	if long {
		w = 112
	}
	b := newPix(w, 42)
	x0, x1 := 5, w-3
	hullSide(b, x0, x1, 25, 35, 22, 3, body, trim, boatBoot(body))
	// the outboard motor
	b.rect(1, 22, 4, 12, c(0x2a2a32))
	b.rect(1, 21, 4, 3, c(0x6a6a76))
	b.rect(1, 21, 4, 1, c(0x9a9aa6))
	b.rect(2, 30, 1, 4, c(0x4a4a56))
	lightA, lightB := craftLights(style)
	switch {
	case style == crLimo || long:
		boatYacht(b, w, body, trim, frame)
	case openCraft(style):
		boatOpen(b, w, body, trim, style, frame)
	default:
		boatCabin(b, w, body, trim, style, frame, lightA, lightB)
	}
	if style == crRangers { // the five colours along the hull
		for i, cc := range []uint32{0xe03030, 0x3060e0, 0xf0d020, 0xf070b0, 0x30c050} {
			b.rect(30+i*6, 31, 6, 2, c(cc))
		}
	}
	o := b.outlined(vInk)
	vWater(o, 2, w-4, 36)
	return o
}

// boatCabin: the work boats (police, ambulance, fire, technicians, hearse): a white cabin with a
// slanted wind-screen, a mast with its pennant, a rail on the aft deck and the light bar.
func boatCabin(b *pixBuf, w int, body, trim color.RGBA, style, frame int, la, lb color.RGBA) {
	wall := boatWall(body)
	if style == crHearse {
		wall = c(0x2a2232)
	}
	cx0, cx1 := 24, 62
	for y := 13; y <= 24; y++ { // the front leans back: a slanted wind-screen
		fx := cx1 - (24-y)/2
		b.rect(cx0, y, fx-cx0, 1, wall)
		b.set(fx, y, c(0x5ab0e0))
		b.set(fx+1, y, c(0x2c64b0))
	}
	b.rect(cx0, 13, cx1-cx0-4, 1, shade(wall, 1.1))
	b.rect(cx0, 24, cx1-cx0+1, 1, shade(wall, 0.75))
	b.rect(cx0-1, 11, cx1-cx0-3, 2, trim) // roof
	b.rect(cx0-1, 10, cx1-cx0-3, 1, shade(trim, 1.25))
	for x := cx0 + 3; x < cx1-12; x += 9 {
		vWindow(b, x, 15, 6, 5)
	}
	b.rect(cx1-11, 15, 6, 9, shade(wall, 0.82)) // the door
	vWindow(b, cx1-10, 16, 4, 3)
	b.set(cx1-7, 20, c(0xd8b040))
	vRail(b, 7, cx0-1, 22, c(0xc8ccd8)) // aft rail
	vRail(b, cx1+3, w-14, 22, c(0xc8ccd8))
	// the mast and its pennant
	b.rect(cx0+4, 3, 1, 8, c(0x5a5a66))
	flag := trim
	switch style {
	case crMedic:
		flag = c(0xd82a2a)
	case crHearse:
		flag = c(0x14101a)
	}
	vFlag(b, cx0+4, 3, flag, frame)
	if style == crHearse { // a little skull on the black flag
		b.set(cx0+2, 4, c(0xf0f0f0))
	}
	if la.A > 0 { // the light bar
		b.rect(cx0+12, 8, 4, 2, la)
		b.rect(cx0+16, 8, 4, 2, lb)
		b.rect(cx0+12, 8, 8, 1, shade(la, 1.3))
	}
	switch style {
	case crMedic:
		vCross(b, cx0+19, 14)
		b.ellipse(14, 17, 3.5, 3.5, c(0xf06a20)) // the life ring on the aft deck
		b.ellipse(14, 17, 1.5, 1.5, color.RGBA{})
		b.rect(11, 16, 2, 2, c(0xf8f8fc))
		b.rect(16, 16, 2, 2, c(0xf8f8fc))
	case crFire: // the water monitor on the foredeck
		b.rect(w-30, 20, 4, 4, c(0x8a8e9c))
		for i := 0; i < 7; i++ {
			b.rect(w-28+i, 19-i/2, 2, 2, c(0xc8ccd8))
		}
		b.set(w-21, 15, c(0x9ad8f4))
	case crTech: // a dish on the roof
		b.ellipse(float64(cx0+28), 7, 3, 2, c(0xe8e8ee))
		b.rect(cx0+28, 8, 1, 3, c(0x6a6a76))
	case crHearse: // purple neon along the hull
		b.rect(10, 30, w-30, 1, c(0xb05aff))
	}
}

// boatOpen: the open speedboats: a wind-screen and the crew at the wheel. The military one (as in the
// war films) carries a gunner in his helmet behind a machine gun on a pintle.
func boatOpen(b *pixBuf, w int, body, trim color.RGBA, style, frame int) {
	// the wind-screen: a triangle of glass leaning back
	for y := 12; y <= 24; y++ {
		n := (y - 12) * 2 / 3
		b.rect(38-n, y, n+1, 1, c(0x8ad0f4))
		b.set(38, y, c(0xd8f2ff))
	}
	b.rect(26, 23, 16, 2, shade(body, 0.7)) // the console
	head := c(0xd8a070)
	switch style {
	case crMilitary:
		helmet := c(0x5a6448)
		b.rect(48, 15, 7, 9, c(0x4a5a30)) // the gunner: jacket, face, helmet
		b.rect(49, 11, 5, 4, head)
		b.set(53, 12, c(0x2a1a10))
		b.rect(47, 9, 8, 3, helmet)
		b.rect(48, 8, 6, 1, shade(helmet, 1.2))
		b.rect(46, 11, 10, 1, shade(helmet, 0.8))
		b.rect(62, 16, 2, 8, c(0x3a3a42))  // the pintle
		b.rect(56, 14, 10, 3, c(0x3a3a42)) // the gun, its jacket and the barrel
		b.rect(56, 14, 10, 1, c(0x6a6a76))
		for x := 66; x < 78; x++ {
			b.set(x, 15, c(0x2a2a30))
		}
		for x := 66; x < 74; x += 2 {
			b.set(x, 14, c(0x5a5a64))
		}
		b.rect(57, 17, 4, 3, c(0x6a7040)) // ammo box
		b.rect(54, 17, 3, 2, head)        // his hands on the grips
	case crSWAT:
		b.rect(46, 15, 6, 9, c(0x22242a))
		b.rect(47, 11, 4, 4, head)
		b.rect(46, 9, 6, 3, c(0x14141a))
		b.rect(52, 17, 9, 2, c(0x2a2a30)) // his rifle
		b.rect(10, 26, w-34, 1, c(0xffca26))
	case crFBI:
		b.rect(28, 16, 6, 8, c(0x14141c))
		b.rect(29, 12, 4, 4, head)
		b.rect(29, 13, 4, 1, c(0x14101a)) // sunglasses
		b.rect(29, 11, 4, 1, c(0x2a1c14))
		b.rect(12, 9, 1, 14, c(0x8a8a94)) // the antenna
		b.set(12, 8, c(0xff4040))
	default: // kids: the pilot and a flag
		b.rect(28, 16, 6, 8, trim)
		b.rect(29, 12, 4, 4, head)
		b.rect(28, 11, 6, 1, body)
		b.rect(10, 6, 1, 17, c(0x8a8a94))
		vFlag(b, 10, 6, trim, frame)
	}
	vRail(b, 8, 24, 23, c(0xc8ccd8))
}

// boatYacht: the long one (the limousine of the sea): two white decks with windows and rails, the
// bridge, a mast with the pennant and the funnel.
func boatYacht(b *pixBuf, w int, body, trim color.RGBA, frame int) {
	wall := boatWall(body)
	b.rect(14, 15, w-40, 10, wall) // lower deck
	for y := 15; y < 25; y++ {
		b.set(w-26+(25-y)/2, y, wall)
	}
	b.rect(14, 15, w-40, 1, shade(wall, 1.1))
	for x := 18; x < w-32; x += 8 {
		vWindow(b, x, 17, 5, 4)
	}
	b.rect(12, 13, w-36, 2, trim)
	vRail(b, 12, w-30, 10, c(0xc8ccd8))
	b.rect(36, 5, 34, 8, wall) // the bridge
	for x := 39; x < 64; x += 7 {
		vWindow(b, x, 7, 5, 3)
	}
	b.rect(68, 6, 2, 6, c(0x2c64b0))
	b.rect(35, 3, 36, 2, trim)
	b.rect(26, 7, 6, 6, c(0x2a2a32)) // funnel
	b.rect(26, 7, 6, 2, trim)
	b.rect(48, 0, 1, 3, c(0x5a5a66))
	vFlag(b, 48, 0, trim, frame)
	b.rect(4, 22, w-10, 1, shade(body, 1.35))
}

// genBoatView: the launch seen from the bow (front) or the stern.
func genBoatView(body, trim color.RGBA, long, front bool, style int) *pixBuf {
	w := 48
	if long {
		w = 58
	}
	b := newPix(w, 44)
	hw := w - 8
	boot := boatBoot(body)
	for i := 0; i <= 10; i++ {
		n := hw
		if front { // the hull narrows towards the waterline: the bow comes at us
			n = hw - i*3
		}
		col := body
		switch {
		case i == 0:
			col = shade(body, 1.35)
		case i == 3 || i == 4:
			col = trim
		case i >= 8:
			col = boot
		}
		b.rect((w-n)/2, 26+i, n, 1, col)
	}
	if front {
		b.rect(w/2, 26, 1, 10, shade(body, 1.25)) // the stem
	} else {
		b.rect(w/2-3, 26, 6, 12, c(0x2a2a32)) // the outboard motor and its wash
		b.rect(w/2-3, 25, 6, 2, c(0x6a6a76))
		b.rect(w/2-9, 34, 5, 2, c(0xe8f4ff))
		b.rect(w/2+4, 34, 5, 2, c(0xe8f4ff))
	}
	la, lb := craftLights(style)
	wall := boatWall(body)
	if style == crHearse {
		wall = c(0x2a2232)
	}
	cw := 26
	if long || style == crLimo {
		cw = 34
	}
	cx := (w - cw) / 2
	if openCraft(style) {
		if front {
			for y := 16; y < 25; y++ { // the wind-screen
				n := 18 - (24-y)/2
				b.rect(w/2-n/2, y, n, 1, c(0x8ad0f4))
			}
		}
		b.rect(w/2-3, 12, 6, 6, c(0xd8a070)) // the one at the wheel
		hat := shade(body, 0.8)
		if style == crMilitary {
			hat = c(0x5a6448)
		}
		b.rect(w/2-4, 10, 8, 3, hat)
		if style == crMilitary && front { // the muzzle of the machine gun, at us
			b.rect(w/2+5, 16, 5, 5, c(0x2a2a30))
			b.rect(w/2+6, 17, 3, 3, c(0x6a6a76))
			b.set(w/2+7, 18, c(0x14101a))
		}
	} else {
		b.rect(cx, 13, cw, 13, wall)
		b.rect(cx-1, 11, cw+2, 2, trim)
		if front {
			b.rect(cx+2, 16, cw-4, 6, c(0x2c64b0)) // wind-screen
			b.rect(cx+2, 16, cw-4, 1, c(0x8ad0f4))
			b.rect(w/2, 16, 1, 6, wall)
		} else {
			vWindow(b, cx+3, 16, 5, 5)
			vWindow(b, cx+cw-8, 16, 5, 5)
			b.rect(w/2-3, 17, 6, 8, shade(wall, 0.82)) // the door
		}
		if style == crMedic {
			vCross(b, w/2-3, 4)
		}
		if long || style == crLimo { // the bridge
			b.rect(w/2-9, 6, 18, 6, wall)
			b.rect(w/2-7, 8, 14, 2, c(0x2c64b0))
		}
	}
	if la.A > 0 {
		b.rect(w/2-4, 8, 4, 2, la)
		b.rect(w/2, 8, 4, 2, lb)
	} else if !openCraft(style) {
		b.rect(w/2, 4, 1, 7, c(0x5a5a66)) // the mast
	}
	o := b.outlined(vInk)
	vWater(o, 2, w-3, 37)
	return o
}

// ---- spaceships -------------------------------------------------------------------------

// shipNeon: the colour of the neon strips of a ship.
func shipNeon(style int) color.RGBA {
	if style == crHearse {
		return rgb(0xb05aff)
	}
	return rgb(0xff58d8)
}

// genShip paints a little spaceship facing right, 96 px long (116 for the long one): an angular
// hull, a big cyan intake, the canopy, a fin swept back, wings swept down and neon lights.
func genShip(body, trim color.RGBA, long bool, style int) *pixBuf {
	w := 96
	if long {
		w = 116
	}
	b := newPix(w, 42)
	nose := w - 3
	hi, lo, seam := shade(body, 1.22), shade(body, 0.68), shade(body, 0.82)
	neon := shipNeon(style)
	cyan, cyanHi := c(0x3ad8f0), c(0xc8faff)
	// the fin, swept back
	for y := 3; y <= 12; y++ {
		f := float64(y-3) / 9
		xl, xr := 10+int(f*4), 16+int(f*14)
		b.rect(xl, y, xr-xl, 1, seam)
		if y <= 4 {
			b.rect(xl, y, xr-xl, 1, trim)
		}
	}
	b.set(11, 2, neon)
	// the wing, swept down and back, with a pod on its tip
	for y := 29; y <= 38; y++ {
		f := float64(y-29) / 9
		xl, xr := 36-int(f*12), 62-int(f*20)
		col := shade(body, 0.75)
		if y == 34 {
			col = trim
		}
		b.rect(xl, y, xr-xl, 1, col)
		b.set(xr-1, y, shade(body, 1.05))
	}
	b.rect(20, 38, 12, 2, c(0x3a3e4a))
	b.set(31, 38, neon)
	// the fuselage: a wedge from the engine block to the nose
	for x := 6; x <= nose; x++ {
		top, bot := 12, 31
		if x > 34 {
			top = 12 + int(math.Round(float64(x-34)*9/float64(nose-34)))
		}
		if x > w-28 {
			bot = 31 - int(math.Round(float64(x-(w-28))*7/25))
		}
		for y := top; y <= bot; y++ {
			col := body
			switch {
			case y == top:
				col = hi
			case y < top+3:
				col = shade(body, 1.1)
			case y >= bot-2:
				col = lo
			case y == 22:
				col = seam
			}
			b.set(x, y, col)
		}
		if x > 36 && x < w-14 && (x-36)%13 == 0 { // the panel seams
			b.rect(x, top+3, 1, bot-top-5, seam)
		}
	}
	b.rect(6, 12, 16, 20, shade(body, 0.88)) // the engine block
	b.rect(6, 12, 16, 1, hi)
	b.rect(6, 29, 16, 3, lo)
	b.rect(1, 15, 5, 14, c(0x3a3e4a)) // the nozzle with its cyan glow
	b.rect(1, 17, 2, 10, cyan)
	b.rect(1, 19, 1, 6, cyanHi)
	b.ellipse(27, 22, 6, 7, c(0x2a2e3a)) // the big intake ring
	b.ellipse(27, 22, 4.4, 5.4, cyan)
	b.ellipse(26.5, 21, 2, 2.6, cyanHi)
	b.rect(36, 24, w-60, 2, trim) // the accent stripe and the details
	b.rect(10, 16, 6, 2, trim)
	b.rect(10, 26, 8, 1, trim)
	for x := 42; x < w-46; x += 7 { // portholes
		b.rect(x, 18, 2, 2, c(0xfff0b0))
	}
	// the canopy
	for y := 6; y <= 13; y++ {
		k := 13 - y
		xl, xr := w-48+k*2, w-24-k
		if y == 13 {
			b.rect(xl, y, xr-xl, 1, c(0x1c4a68))
			continue
		}
		b.rect(xl, y, xr-xl, 1, cyan)
		if y <= 8 {
			b.rect(xl+2, y, 4, 1, cyanHi)
		}
	}
	b.rect(w-36, 2, 1, 4, c(0x8a8e9c)) // antenna
	// the neon: the nose light and one under the body
	b.rect(nose-8, 22, 7, 2, neon)
	b.rect(nose-8, 22, 7, 1, shade(neon, 1.3))
	b.rect(w-42, 27, 6, 2, neon)
	b.ellipse(50, 32.5, 2.5, 2, cyan) // the round thrusters under it
	b.ellipse(float64(w-30), 31.5, 2, 1.6, cyan)
	la, lb := craftLights(style)
	switch style {
	case crMilitary: // a gun under the nose and missiles on the wing
		b.rect(w-34, 30, 20, 2, c(0x2a2a30))
		b.rect(40, 35, 10, 2, c(0x6a7040))
		b.set(50, 35, c(0xd02a2a))
	case crMedic:
		vCross(b, 44, 15)
	case crTech:
		b.ellipse(48, 9, 4, 2, c(0xe8e8ee))
		b.rect(48, 10, 1, 3, c(0x6a6a76))
	case crRangers:
		for i, cc := range []uint32{0xe03030, 0x3060e0, 0xf0d020, 0xf070b0, 0x30c050} {
			b.rect(38+i*5, 27, 5, 2, c(cc))
		}
	}
	if la.A > 0 { // the light bar behind the canopy
		b.rect(w-56, 9, 3, 2, la)
		b.rect(w-53, 9, 3, 2, lb)
	}
	if long { // a second row of portholes: the limousine of space
		for x := 40; x < w-50; x += 7 {
			b.rect(x, 27, 3, 2, c(0xfff0b0))
		}
	}
	return b.outlined(vInk)
}

// genShipView: the spaceship seen from the nose (front) or from the engine (rear).
func genShipView(body, trim color.RGBA, long, front bool, style int) *pixBuf {
	w := 70
	if long {
		w = 86
	}
	b := newPix(w, 46)
	cx := w / 2
	neon := shipNeon(style)
	cyan, cyanHi := c(0x3ad8f0), c(0xc8faff)
	// the wings: an inverted V seen end-on, with a pod and a light on each tip
	span := w/2 - 5
	for i := 0; i <= span-8; i++ {
		y := 27 + i*9/(span-8)
		for _, s := range []int{-1, 1} {
			x := cx + s*(8+i)
			b.rect(x, y, 1, 3, shade(body, 0.75))
			b.set(x, y+1, trim)
		}
	}
	for _, s := range []int{-1, 1} {
		b.rect(cx+s*span-2, 35, 4, 4, c(0x3a3e4a))
		b.set(cx+s*span, 36, neon)
	}
	// the fin
	b.rect(cx-1, 4, 3, 10, shade(body, 0.82))
	b.rect(cx-1, 4, 3, 2, trim)
	// the fuselage
	b.ellipse(float64(cx)+0.5, 23, 13, 11, shade(body, 0.68))
	b.ellipse(float64(cx)+0.5, 22, 13, 10, body)
	b.ellipse(float64(cx)+0.5, 19, 10, 5, shade(body, 1.18))
	b.rect(cx-12, 25, 25, 1, trim)
	if front {
		b.ellipse(float64(cx)+0.5, 17, 7, 4, cyan) // the canopy
		b.rect(cx-4, 15, 3, 1, cyanHi)
		b.rect(cx-5, 27, 11, 2, neon) // the nose light
		b.ellipse(float64(cx-9), 31, 2, 1.6, cyan)
		b.ellipse(float64(cx+10), 31, 2, 1.6, cyan)
	} else {
		b.ellipse(float64(cx)+0.5, 22, 7, 7, c(0x2a2e3a)) // the engine
		b.ellipse(float64(cx)+0.5, 22, 5, 5, cyan)
		b.ellipse(float64(cx)+0.5, 21, 2, 2, cyanHi)
		b.rect(cx-12, 29, 4, 2, neon)
		b.rect(cx+9, 29, 4, 2, neon)
	}
	la, lb := craftLights(style)
	if la.A > 0 {
		b.rect(cx-4, 11, 4, 2, la)
		b.rect(cx+1, 11, 4, 2, lb)
	}
	if style == crMedic {
		vCross(b, cx-3, 30)
	}
	if style == crMilitary && front {
		b.rect(cx-2, 33, 5, 4, c(0x2a2a30))
		b.set(cx, 35, c(0x14101a))
	}
	return b.outlined(vInk)
}

// ---- the ferry of Isla Nublada -----------------------------------------------------------

// genFerry paints the ferry moored at the pier (jp_boat, 124x66): a hull in red and orange bands, a white
// two-storey cabin with blue windows and a door, rails, the funnel and a mast with its flag.
func genFerry(b *pixBuf) {
	white, roof := c(0xf2f2f6), c(0x8a3a20)
	for x := 6; x <= 120; x++ { // the hull: white topsides, the stripes, a dark keel
		f := 0.0
		if x > 102 {
			f = float64(x-102) / 18
		}
		top := 36 - int(math.Round(4*f*f))
		bot := 56 - int(math.Round(16*math.Pow(f, 1.4)))
		for y := top; y <= bot; y++ {
			col := white
			switch {
			case y == top:
				col = c(0xffffff)
			case y >= 44 && y <= 46:
				col = c(0xd83a2a)
			case y >= 47 && y <= 49:
				col = c(0xf08a20)
			case y >= 50 && y <= 52:
				col = c(0xc82a2a)
			case y >= 53:
				col = c(0x24305a)
			}
			if x == 6 && y > top {
				col = shade(col, 0.82)
			}
			b.set(x, y, col)
		}
	}
	vRail(b, 8, 26, 31, c(0xc87a30)) // the rails, fore and aft
	vRail(b, 90, 112, 30, c(0xc87a30))
	// the main cabin
	b.rect(26, 24, 64, 12, white)
	b.rect(24, 22, 68, 2, roof)
	b.rect(24, 22, 68, 1, shade(roof, 1.3))
	for x := 30; x < 82; x += 9 {
		if x >= 57 && x < 66 {
			continue
		}
		vWindow(b, x, 27, 6, 5)
	}
	b.rect(58, 26, 7, 10, c(0xd84a2a)) // the door
	b.rect(59, 27, 5, 4, c(0x5ab0e0))
	b.set(63, 32, c(0xf0d040))
	b.rect(26, 35, 64, 1, c(0xb8bcc8))
	// the bridge on top
	b.rect(38, 12, 38, 10, white)
	b.rect(36, 10, 42, 2, roof)
	b.rect(36, 10, 42, 1, shade(roof, 1.3))
	for x := 42; x < 72; x += 9 {
		vWindow(b, x, 14, 6, 5)
	}
	vRail(b, 78, 92, 19, c(0xc87a30))
	// the funnel and the mast with its flag
	b.rect(28, 12, 6, 10, c(0xf2f2f6))
	b.rect(28, 12, 6, 3, c(0xd83a2a))
	b.rect(28, 11, 6, 1, c(0x20202a))
	b.rect(56, 1, 2, 9, c(0x6a3a20))
	b.rect(58, 1, 6, 3, c(0xf0c040))
	b.rect(58, 2, 7, 1, c(0xf0c040))
	b.rect(58, 1, 6, 1, c(0xf8e080))
	// the foam on the waterline (painted under the outline by the prop pass: light, not white)
	for k := 0; k < 4; k++ {
		b.ellipse(10+float64(k)*3, 58+float64(k%2), 2.4, 1.2, c(0xd8eef8))
	}
}

// ---- the ships at sea of the beach levels ------------------------------------------------

// genPirateShip paints the pirate ship facing right, 66x54 (waterline on row 49): a brown hull with
// a yellow rail, portholes and an anchor, a big grey sail with the skull and crossbones on the
// foremast, red and grey sails on the mainmast, a bowsprit with its rope and red pennants
// waving (frame 0/1).
func genPirateShip(frame int) *pixBuf {
	b := newPix(68, 56)
	wood, woodD, woodL := c(0x8a5428), c(0x5a3418), c(0xa86a34)
	mast := c(0x6a4020)
	// the masts (behind the sails)
	b.rect(22, 3, 2, 37, mast)
	b.rect(46, 6, 2, 34, mast)
	// the mainmast: two tiers of grey sails with a red band, billowing to the right
	for _, s := range [][2]int{{8, 9}, {20, 11}} {
		y0, h := s[0], s[1]
		for y := y0; y < y0+h; y++ {
			k := y - y0
			bulge := int(math.Round(2 * math.Sin(math.Pi*float64(k)/float64(h-1))))
			xl, xr := 12, 33+bulge
			b.rect(xl, y, xr-xl, 1, c(0x5a6470))
			b.rect(18, y, 6, 1, c(0xb83030))
			b.set(xr-1, y, c(0x3a4048))
			if k == 0 {
				b.rect(xl, y, xr-xl, 1, c(0x7a8490))
			}
		}
		b.rect(11, y0-1, 24, 1, mast) // the yard
	}
	// the foremast: one big sail with the skull and crossbones
	for y := 12; y < 34; y++ {
		k := y - 12
		bulge := int(math.Round(3 * math.Sin(math.Pi*float64(k)/21)))
		xl, xr := 36, 58+bulge
		b.rect(xl, y, xr-xl, 1, c(0x5a6470))
		b.set(xr-1, y, c(0x3a4048))
		b.set(xl, y, c(0x7a8490))
	}
	b.rect(34, 11, 27, 1, mast)
	b.rect(34, 34, 27, 1, mast)
	skull := c(0xf0f0f0)
	b.rect(44, 17, 7, 6, skull) // the skull
	b.rect(45, 23, 5, 2, skull)
	b.rect(45, 19, 2, 2, c(0x14101a))
	b.rect(48, 19, 2, 2, c(0x14101a))
	b.set(46, 24, c(0x5a6470))
	b.set(48, 24, c(0x5a6470))
	for i := 0; i < 7; i++ { // the crossed bones
		b.set(41+i*2, 25+i/2, skull)
		b.set(53-i*2, 25+i/2, skull)
	}
	b.rect(44, 7, 6, 2, woodD) // the crow's nest
	// the pennants
	vFlag(b, 23, 0, c(0xc83030), frame)
	b.rect(23, 0, 4, 3, c(0xc83030))
	b.rect(47, 3, 4, 3, c(0xc83030))
	if frame == 0 {
		b.rect(51, 4, 2, 1, c(0xc83030))
	} else {
		b.rect(51, 3, 2, 1, c(0xc83030))
	}
	// the bowsprit and its rope
	for i := 0; i < 8; i++ {
		b.set(57+i, 39-i/2, mast)
	}
	for i := 0; i < 16; i += 2 {
		b.set(64-i, 35-i*2, c(0x3a2a1a))
	}
	// the hull: a raised stern castle, planks, the yellow rail, portholes and the anchor
	b.rect(6, 33, 16, 7, wood)
	b.rect(6, 33, 16, 1, c(0xf0c030))
	b.rect(8, 35, 3, 2, c(0xf8d860)) // the stern lantern window
	for x := 6; x <= 62; x++ {
		f := 0.0
		if x > 50 {
			f = float64(x-50) / 12
		}
		top := 40 - int(math.Round(4*f*f))
		bot := 50 - int(math.Round(8*math.Pow(f, 1.5)))
		for y := top; y <= bot; y++ {
			col := wood
			switch {
			case y <= top+1:
				col = c(0xf0c030)
			case y == top+2:
				col = c(0xb88a20)
			case y >= 48:
				col = woodD
			case (y-top)%3 == 0:
				col = shade(wood, 0.82)
			case (x+y*7)%11 == 0:
				col = woodL
			}
			b.set(x, y, col)
		}
	}
	for _, px := range []int{20, 30, 40} { // portholes
		b.rect(px, 43, 4, 4, c(0xa0a4b0))
		b.rect(px+1, 44, 2, 2, c(0x1c2c48))
	}
	gold := c(0xf0c030) // the anchor
	b.rect(53, 41, 1, 5, gold)
	b.rect(52, 40, 3, 1, gold)
	b.rect(51, 45, 5, 1, gold)
	b.set(51, 44, gold)
	b.set(55, 44, gold)
	o := b.outlined(vInk)
	vWater(o, 4, 62, 51)
	return o
}

// horizonShipW, horizonShipH: the little ship on the sunset horizon (seasky.go), a silhouette.
const horizonShipW, horizonShipH = 40, 16

// genHorizonShip paints the far ship on the horizon: a dark silhouette (against the sunset) with a
// raked bow, two decks, a funnel, a mast and a few lit windows. The hull's top-left is (0, 8).
func genHorizonShip(col color.RGBA) *pixBuf {
	b := newPix(horizonShipW, horizonShipH)
	for x := 0; x < horizonShipW; x++ {
		top := 8
		if x > 33 {
			top = 8 - (x-33)/3
		}
		bot := 15
		if x > 30 {
			bot = 15 - (x-30)*2/3
		}
		b.rect(x, top, 1, bot-top+1, col)
	}
	b.rect(8, 4, 20, 4, col)  // the decks
	b.rect(12, 1, 10, 3, col) // the bridge
	b.rect(24, 1, 3, 3, col)  // the funnel
	b.rect(16, 0, 1, 1, col)  // the mast
	lit := mixc(col, c(0xf0c060), 0.75)
	for x := 10; x < 27; x += 4 {
		b.set(x, 5, lit)
	}
	b.set(14, 2, lit)
	b.set(18, 2, lit)
	return b
}

// ---- the rocket ----------------------------------------------------------------------------

// rocketW, rocketH: the size of the rocket sprite (its nozzles end on row 51).
const rocketW, rocketH = 40, 54

// genRocket paints the rocket that crosses the sky, pointing up: a cream body shaded from the left,
// a blue cockpit window, swept wings with red stripes and two engines (the flame is drawn live).
func genRocket() *pixBuf {
	b := newPix(rocketW, rocketH)
	cx := rocketW / 2
	cream, light, dark := c(0xe8e2cc), c(0xfaf6e8), c(0xb4ae98)
	// the wings, behind the body
	for y := 26; y <= 44; y++ {
		ext := (y - 26) / 2
		if ext > 9 {
			ext = 9
		}
		xl := cx - 8 - ext
		b.rect(xl, y, cx+8+ext-xl, 1, cream)
		b.set(xl, y, light)
		b.set(cx+8+ext-1, y, dark)
		if y >= 37 && y <= 39 {
			b.rect(xl, y, ext, 1, c(0xd03030))
			b.rect(cx+8, y, ext, 1, c(0xb02424))
		}
		if y == 44 {
			b.rect(xl, y, cx+8+ext-xl, 1, dark)
		}
	}
	// the body: a nose cone and a cylinder, a highlight on the left and a shadow on the right
	for y := 1; y <= 45; y++ {
		hw := 8
		if y < 14 {
			hw = 1 + (y-1)*7/13
		}
		for x := cx - hw; x < cx+hw; x++ {
			u := float64(x-(cx-hw)) / float64(2*hw)
			col := cream
			switch {
			case u < 0.22:
				col = light
			case u > 0.74:
				col = dark
			}
			b.set(x, y, col)
		}
	}
	b.rect(cx-8, 28, 16, 1, dark) // a panel seam and the spine
	b.rect(cx-1, 30, 2, 15, c(0xc8c2ac))
	b.ellipse(float64(cx), 17, 4.2, 5.5, c(0x1a3a5a)) // the cockpit window
	b.ellipse(float64(cx), 17, 3.2, 4.5, c(0x2a78b8))
	b.rect(cx-2, 14, 2, 2, c(0x8ad0f0))
	// the engines
	for _, ex := range []int{cx - 7, cx + 2} {
		b.rect(ex, 45, 5, 6, c(0x8a8e9c))
		b.rect(ex+3, 45, 2, 6, c(0x5a5e6a))
		b.rect(ex, 50, 5, 1, c(0xc86a2a))
	}
	return b.outlined(vInk)
}

// ---- the flying saucer ---------------------------------------------------------------------

// genUFO paints the flying saucer (34x18, as before): a grey hull with a red rim stripe, the cyan
// dome, a row of pink and cyan neon lights and the beam emitter underneath.
func genUFO() *pixBuf {
	b := newPix(34, 18)
	b.ox, b.oy = 1, 1
	b.ellipse(16, 9.5, 15, 4.5, c(0x4a4e5a)) // the underside
	b.ellipse(16, 8.5, 15, 3.8, c(0x8a90a0))
	b.ellipse(16, 7.5, 13, 2.6, c(0xc8ccd8)) // the lit top of the disc
	b.rect(2, 9, 28, 1, c(0xd03a3a))         // the red rim stripe
	b.ellipse(16, 5, 7, 5, c(0x2a2e3a))      // the dome: frame, glass, glint
	b.ellipse(16, 5, 6, 4.2, c(0x3ad8f0))
	b.ellipse(14, 3.5, 2.5, 1.6, c(0xc8faff))
	b.rect(9, 7, 15, 1, c(0x6a7080))
	for i := 0; i < 5; i++ { // the neon lights
		col := c(0xff58d8)
		if i%2 == 1 {
			col = c(0x3ad8f0)
		}
		b.rect(5+i*5, 11, 2, 1, col)
	}
	b.rect(11, 13, 10, 2, c(0x3a3e4a)) // the beam emitter
	b.rect(13, 14, 6, 1, c(0x3ad8f0))
	return b.outlined(vInk)
}
