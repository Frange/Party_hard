package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The painted backdrop of ALCALÁ DE HENARES (alcala.go): 960 x alcBackH, made once. Dusk sky (purple above, orange on
// the mountains), the mountains and the far roofs, and the front row of facades standing on the square:
//   x   0-262  the arcaded building of the terraces (arches with the lit cafés, balconies, red roof)
//   x 270-326  the crenellated stone tower (a stork's nest on top)
//   x 334-648  the plateresque university: three rows of framed windows, pilasters, the crest over the portal
//   x 668-716  the belfry (a stork's nest on top)
//   x 716-884  the gothic church: pinnacles, the rose window and the pointed portal
//   x 884-960  white houses with balconies and a chimney (a stork's nest)
// Drawn with the muted dusk palette of the game; no picture is copied.
// ---------------------------------------------------------------------------

// alcStorkSpots: where the storks stand (the anchor = the bottom of the nest), on the tower, the belfry and a chimney.
var alcStorkSpots = []Vec{{298, 24}, {692, 22}, {928, 70}}

var (
	alcSkyTop   = rgb(0x3a2c66)
	alcSkyLow   = rgb(0xd88a6c)
	alcMount1   = rgb(0x6a5486)
	alcMount2   = rgb(0x4e3e6e)
	alcRoof     = rgb(0xa8503c)
	alcRoofD    = rgb(0x7a3428)
	alcStone    = rgb(0xd8c2a2)
	alcStoneD   = rgb(0xa8927a)
	alcStoneL   = rgb(0xeedcc0)
	alcBrown    = rgb(0xb08a68)
	alcDark     = rgb(0x2e2228)
	alcLit      = rgb(0xe8b468)
	alcWhite    = rgb(0xe6dccc)
	alcIron     = rgb(0x2a2630)
	alcWood     = rgb(0x5a3626)
	alcTreeDark = rgb(0x2c4632)
)

func alcalaBackdrop() *pixBuf {
	if alcBackBuf != nil {
		return alcBackBuf
	}
	W, H := 960, int(alcBackH)
	b := newPix(W, H)
	// the sky: purple above, orange towards the mountains, a few pink streaks of cloud
	for y := 0; y < H; y++ {
		t := math.Pow(math.Min(1, float64(y)/72), 1.25)
		c := mixc(alcSkyTop, alcSkyLow, t)
		b.rect(0, y, W, 1, c)
	}
	for i := 0; i < 14; i++ {
		x, y := phHash(i, 3)%W, 8+phHash(i, 5)%40
		w := 30 + phHash(i, 7)%70
		b.rect(x, y, w, 1, mixc(rgb(0xe89a8a), alcSkyTop, float64(y)/80))
		b.rect(x+6, y+1, w-12, 1, mixc(rgb(0xc87a8a), alcSkyTop, float64(y)/80))
	}
	// two ranges of mountains
	for x := 0; x < W; x++ {
		fx := float64(x)
		h1 := 50 + 7*math.Sin(fx/74) + 4*math.Sin(fx/23+1) + 2*math.Sin(fx/9)
		h2 := 62 + 5*math.Sin(fx/51+2) + 3*math.Sin(fx/17)
		b.rect(x, int(h1), 1, H, alcMount1)
		b.rect(x, int(h2), 1, H, alcMount2)
	}
	// the far town: rows of little roofs and dark treetops
	for i := 0; i < 70; i++ {
		x := phHash(i, 11) % W
		y := 72 + phHash(i, 13)%14
		if i%3 == 0 {
			b.ellipse(float64(x), float64(y), 7, 6, alcTreeDark)
			continue
		}
		w := 14 + phHash(i, 17)%16
		b.rect(x, y+4, w, 30, shade(alcWhite, 0.62))
		b.rect(x-1, y, w+2, 4, shade(alcRoof, 0.7))
	}
	b.rect(0, 92, W, H-92, shade(alcWhite, 0.55))

	alcArcade(b)
	alcTower(b, 270, 326, 22)
	alcUniversity(b)
	for _, x := range []int{652, 664} { // two cypresses in the gap
		alcCypress(b, x, 148)
	}
	alcBelfry(b, 668, 716, 20)
	alcChurch(b)
	alcHouses(b)
	// the kerb where the facades meet the square
	b.rect(0, H-4, W, 4, alcStoneD)
	b.rect(0, H-4, W, 1, alcStoneL)
	alcBackBuf = b
	return b
}

// alcWall fills a wall with a light left edge, a dark right edge and a little plaster noise.
func alcWall(b *pixBuf, x0, y0, x1, y1 int, c color.RGBA, seed int) {
	b.rect(x0, y0, x1-x0, y1-y0, c)
	for y := y0; y < y1; y++ {
		b.set(x0, y, shade(c, 1.12))
		b.set(x1-1, y, shade(c, 0.72))
		if phHash(y, seed)%4 == 0 {
			b.rect(x0+phHash(y, seed+1)%maxI(1, x1-x0-4), y, 3, 1, shade(c, 0.93))
		}
	}
}

// alcRoofBand: a band of red tiles (rows of tiles with darker joints).
func alcRoofBand(b *pixBuf, x0, x1, y0, h int) {
	for y := y0; y < y0+h; y++ {
		c := mixc(alcRoof, alcRoofD, float64(y-y0)/float64(h))
		b.rect(x0, y, x1-x0, 1, c)
		if (y-y0)%3 == 2 {
			b.rect(x0, y, x1-x0, 1, shade(c, 0.8))
		}
	}
	for x := x0; x < x1; x += 4 {
		for y := y0; y < y0+h; y += 3 {
			b.set(x+(y/3)%2*2, y, shade(alcRoof, 1.15))
		}
	}
	b.rect(x0, y0+h, x1-x0, 1, alcRoofD)
}

// alcWindow: a dark window, some lit (dusk), with a stone frame.
func alcWindow(b *pixBuf, x, y, w, h int, lit bool, frame color.RGBA) {
	b.rect(x-1, y-1, w+2, h+2, frame)
	in := alcDark
	if lit {
		in = alcLit
	}
	b.rect(x, y, w, h, in)
	b.rect(x+w/2, y, 1, h, shade(frame, 0.8))
	if lit {
		b.rect(x, y, w, 1, shade(alcLit, 1.1))
	}
}

// alcBalcony: an iron balcony under a window.
func alcBalcony(b *pixBuf, x, y, w int) {
	b.rect(x, y, w, 1, alcIron)
	b.rect(x, y+4, w, 1, alcIron)
	for xx := x; xx < x+w; xx += 2 {
		b.rect(xx, y, 1, 5, alcIron)
	}
	b.rect(x, y+5, w, 1, shade(alcStoneD, 0.8))
}

// the arcaded building: two floors of balconies over a row of arches (the lit cafés of the terraces inside)
func alcArcade(b *pixBuf) {
	x0, x1 := 0, 262
	alcRoofBand(b, x0, x1, 66, 9)
	alcWall(b, x0, 76, x1, 146, rgb(0xd8c6a6), 21)
	for i := 0; i < 9; i++ {
		x := 8 + i*28
		alcWindow(b, x+6, 82, 8, 11, i%4 == 1, alcStoneL)
		alcBalcony(b, x+3, 92, 14)
		alcWindow(b, x+6, 100, 8, 9, i%3 == 2, alcStoneL)
	}
	b.rect(x0, 112, x1-x0, 2, alcStoneL) // the cornice over the arcade
	for i := 0; i < 10; i++ {            // the arches
		x := 2 + i*26
		b.rect(x, 116, 6, 30, alcStone)         // the pillar
		b.rect(x, 116, 1, 30, alcStoneL)        // lit edge
		b.rect(x+6, 122, 20, 24, rgb(0x3a2a2a)) // the shade under the arcade
		b.ellipse(float64(x+16), 123, 10, 7, rgb(0x3a2a2a))
		if i%2 == 0 { // the warm light of a café inside
			b.rect(x+11, 128, 10, 8, rgb(0x8a5a3a))
			b.rect(x+13, 130, 6, 4, alcLit)
		}
	}
}

// the crenellated stone tower: blocks, arrow slits and merlons
func alcTower(b *pixBuf, x0, x1, top int) {
	alcWall(b, x0, top, x1, 146, alcBrown, 33)
	for y := top + 4; y < 146; y += 6 { // courses of stone
		b.rect(x0+1, y, x1-x0-2, 1, shade(alcBrown, 0.82))
		for x := x0 + (y/6%2)*5 + 3; x < x1-1; x += 10 {
			b.rect(x, y-5, 1, 5, shade(alcBrown, 0.86))
		}
	}
	for x := x0; x < x1; x += 10 { // merlons
		b.rect(x, top-6, 6, 6, alcBrown)
		b.rect(x, top-6, 6, 1, shade(alcBrown, 1.15))
	}
	b.rect(x0-2, top, x1-x0+4, 2, shade(alcBrown, 0.75))
	for _, y := range []int{top + 16, top + 44, top + 74} {
		mx := (x0 + x1) / 2
		b.rect(mx-2, y, 4, 12, alcDark)
		b.rect(mx-1, y-1, 2, 1, alcDark)
	}
	b.rect((x0+x1)/2-8, 118, 16, 28, alcWood) // a small door
	b.ellipse(float64(x0+x1)/2, 118, 8, 5, alcWood)
}

// the plateresque university: a long sandstone facade with framed windows, pilasters, the red roof, the crest over the
// big portal, corner pinnacles and the steps
func alcUniversity(b *pixBuf) {
	x0, x1 := 334, 648
	mid := (x0 + x1) / 2
	alcRoofBand(b, x0+6, x1-6, 46, 8)
	alcWall(b, x0, 55, x1, 146, alcStone, 41)
	// the frontispiece over the portal: stepped attic with the coat of arms
	b.rect(mid-46, 40, 92, 16, alcStone)
	b.rect(mid-30, 30, 60, 10, alcStone)
	b.rect(mid-12, 22, 24, 8, alcStone)
	b.rect(mid-46, 40, 92, 1, alcStoneL)
	b.rect(mid-30, 30, 60, 1, alcStoneL)
	b.rect(mid-12, 22, 24, 1, alcStoneL)
	b.ellipse(float64(mid), 44, 9, 8, alcStoneD) // the coat of arms
	b.ellipse(float64(mid), 44, 6, 6, rgb(0xc4a882))
	b.rect(mid-1, 39, 2, 10, alcStoneD)
	b.rect(mid-5, 43, 10, 2, alcStoneD)
	for _, x := range []int{mid - 44, mid + 42, mid - 28, mid + 26, mid - 10, mid + 8} { // finials
		y := 34
		if x == mid-10 || x == mid+8 {
			y = 16
		} else if x == mid-28 || x == mid+26 {
			y = 24
		}
		b.rect(x, y, 2, 6, alcStoneD)
		b.set(x, y-1, alcStoneL)
	}
	for _, x := range []int{x0, x1 - 6} { // corner pinnacles
		b.rect(x, 36, 6, 20, alcStone)
		b.rect(x+2, 30, 2, 6, alcStoneD)
	}
	// cornices between the floors
	for _, y := range []int{58, 84, 110} {
		b.rect(x0, y, x1-x0, 2, alcStoneL)
		b.rect(x0, y+2, x1-x0, 1, alcStoneD)
	}
	// pilasters
	for x := x0 + 4; x < x1; x += 40 {
		b.rect(x, 61, 3, 85, shade(alcStone, 1.06))
		b.rect(x+3, 61, 1, 85, alcStoneD)
	}
	// three rows of windows with carved frames; the middle row with balconies
	for i := 0; i < 15; i++ {
		x := x0 + 14 + i*20
		if x > mid-18 && x < mid+14 {
			continue // the portal axis
		}
		lit := phHash(i, 51)%4 == 0
		alcWindow(b, x, 66, 7, 12, lit, alcStoneL)
		b.rect(x-2, 64, 11, 1, alcStoneD) // the carved lintel
		alcWindow(b, x, 90, 7, 13, phHash(i, 53)%5 == 0, alcStoneL)
		if i%2 == 0 {
			alcBalcony(b, x-2, 103, 11)
		}
		alcWindow(b, x, 116, 7, 12, false, alcStoneL)
	}
	// the portal: a round arch with a wooden door, columns either side, and the steps
	b.rect(mid-16, 112, 32, 34, alcStoneL)
	b.rect(mid-11, 120, 22, 26, alcWood)
	b.ellipse(float64(mid), 120, 11, 7, alcWood)
	b.rect(mid, 118, 1, 28, shade(alcWood, 0.7))
	for _, x := range []int{mid - 16, mid + 13} {
		b.rect(x, 86, 3, 60, alcStoneL)
	}
	b.rect(mid-10, 88, 20, 14, alcStoneD) // the carved panel over the portal
	b.rect(mid-8, 90, 16, 10, rgb(0xc8ae8a))
	for i := 0; i < 3; i++ { // the steps
		b.rect(mid-26-i*6, 140+i*2, 52+i*12, 2, shade(alcStoneL, 1-float64(i)*0.06))
	}
}

// the belfry: a square tower with arched openings for the bells and a small hipped roof cut flat for the nest
func alcBelfry(b *pixBuf, x0, x1, top int) {
	alcWall(b, x0, top, x1, 146, alcStone, 61)
	b.rect(x0-2, top, x1-x0+4, 3, alcStoneL)
	for _, y := range []int{top + 8, top + 40} { // the bells' openings
		for _, x := range []int{x0 + 8, x0 + 28} {
			b.rect(x, y, 12, 18, alcDark)
			b.ellipse(float64(x+6), float64(y), 6, 4, alcDark)
			b.rect(x+4, y+6, 4, 5, rgb(0xb08a40)) // a bell
		}
		b.rect(x0, y+20, x1-x0, 2, alcStoneL)
	}
	alcWindow(b, (x0+x1)/2-3, top+78, 6, 14, true, alcStoneL)
	b.rect((x0+x1)/2-3, top+102, 6, 14, alcDark)
}

// the gothic church: the gable, pinnacles, the rose window, lancets and the pointed portal
func alcChurch(b *pixBuf) {
	x0, x1 := 716, 884
	mid := (x0 + x1) / 2
	alcWall(b, x0, 56, x1, 146, alcStone, 71)
	for y := 26; y < 56; y++ { // the gable
		hw := (y - 26) * 84 / 30
		b.rect(mid-hw, y, 2*hw, 1, alcStone)
		b.set(mid-hw, y, alcStoneL)
		b.set(mid+hw-1, y, alcStoneD)
	}
	b.rect(mid-1, 14, 2, 12, alcStoneD) // the cross
	b.rect(mid-4, 17, 8, 2, alcStoneD)
	for _, x := range []int{x0, x0 + 22, x1 - 28, x1 - 6} { // buttresses with their pinnacles
		b.rect(x, 40, 6, 106, shade(alcStone, 1.05))
		b.rect(x+5, 40, 1, 106, alcStoneD)
		for y := 24; y < 40; y++ {
			w := (y - 24) * 3 / 16
			b.rect(x+3-w, y, 2*w, 1, alcStone)
		}
		b.set(x+2, 22, alcStoneL)
	}
	// the rose window
	b.ellipse(float64(mid), 72, 14, 14, alcStoneD)
	b.ellipse(float64(mid), 72, 12, 12, rgb(0x5a3a5a))
	for a := 0.0; a < 2*math.Pi; a += math.Pi / 6 {
		for r := 2.0; r < 12; r++ {
			b.set(mid+int(math.Round(math.Cos(a)*r)), 72+int(math.Round(math.Sin(a)*r)), alcStoneL)
		}
	}
	b.ellipse(float64(mid), 72, 3, 3, alcLit)
	// lancet windows either side
	for _, x := range []int{x0 + 36, x0 + 52, x1 - 58, x1 - 42} {
		b.rect(x, 70, 6, 26, rgb(0x4a3450))
		b.ellipse(float64(x)+3, 70, 3, 4, rgb(0x4a3450))
		b.rect(x+2, 76, 2, 14, rgb(0x8a5a6a))
	}
	// the pointed portal with its archivolts
	for i := 3; i >= 0; i-- {
		c := shade(alcStone, 0.8+float64(i)*0.07)
		w := 14 + i*3
		b.rect(mid-w, 112-i*2, 2*w, 34+i*2, c)
		for y := 0; y < 16; y++ {
			hw := w * y / 16
			b.rect(mid-hw, 96-i*2+y, 2*hw, 1, c)
		}
	}
	b.rect(mid-10, 116, 20, 30, alcWood)
	for y := 0; y < 12; y++ {
		hw := 10 * y / 12
		b.rect(mid-hw, 104+y, 2*hw, 1, alcWood)
	}
	b.rect(mid, 106, 1, 40, shade(alcWood, 0.7))
	for i := 0; i < 3; i++ { // the steps
		b.rect(mid-24-i*5, 140+i*2, 48+i*10, 2, shade(alcStoneL, 1-float64(i)*0.06))
	}
}

// the white houses on the right: red roofs, balconies, green shutters and a chimney with a stork's nest
func alcHouses(b *pixBuf) {
	x0, x1 := 884, 960
	alcRoofBand(b, x0, x1, 74, 8)
	b.rect(918, 66, 14, 9, rgb(0xc8b8a4)) // the chimney under the nest
	b.rect(916, 64, 18, 3, alcStoneD)
	alcWall(b, x0, 83, x1, 146, alcWhite, 81)
	for i := 0; i < 3; i++ {
		x := x0 + 8 + i*24
		alcWindow(b, x, 90, 9, 13, i == 1, alcStoneL)
		b.rect(x-3, 90, 2, 13, rgb(0x3a6a4a)) // shutters
		b.rect(x+10, 90, 2, 13, rgb(0x3a6a4a))
		alcBalcony(b, x-3, 104, 15)
		alcWindow(b, x, 118, 9, 14, false, alcStoneL)
	}
	b.rect(x0+30, 122, 14, 24, alcWood) // a door
}

// alcCypress: a dark cypress standing on the ground at y.
func alcCypress(b *pixBuf, x, y int) {
	for yy := y - 70; yy < y; yy++ {
		t := float64(yy-(y-70)) / 70
		w := 5 * math.Sin(math.Pi*math.Pow(t, 0.7))
		b.rect(x-int(w), yy, 2*int(w)+1, 1, alcTreeDark)
		if int(w) > 1 {
			b.set(x-int(w), yy, shade(alcTreeDark, 1.3))
		}
	}
}
