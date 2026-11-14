package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The painted backdrop of BARCELONA (barcelona.go): 960 x bcnBackH, made once. A blue sky with white clouds, the sea
// on the horizon with sailing boats and the tall hotel shaped like a sail, the hill of the Tibidabo with its church and
// its communications tower (the cable car that climbs it is an animated prop, barcelona_props.go), the roofs of the
// city, and the front row standing on the square:
//   x   0-122  the modernist house: wavy cream facade with mosaic, bone-like balconies and the scaly roof
//   x 122-262  a corner building of the Eixample (balconies, the chamfered corner)
//   x 268-420  the city hall: stone facade, columns, the balcony with the two flags (flagmini props) and the clock tower
//   x 432-592  the market hall: red brick, a big arch of stained glass and the iron roof
//   x 602-768  the Sagrada Família: the tall stone spires with their pinnacles and the crane beside them
//   x 760-880  the Arc de Triomf in red brick
//   x 884-960  another Eixample building
// The top left is under the HUD: only the lower floors of the first two buildings show there. Drawn by code in the
// palette of the game; no picture is copied. The places keep their real names; no real person is named.
// ---------------------------------------------------------------------------

var (
	bcnSkyTop  = rgb(0x3a6ac8)
	bcnSkyLow  = rgb(0xa8d4f2)
	bcnSea     = rgb(0x2a64a8)
	bcnSeaL    = rgb(0x5a94d0)
	bcnHill    = rgb(0x4a7a3e)
	bcnHillD   = rgb(0x335a2e)
	bcnStone   = rgb(0xe2cfa8)
	bcnStoneD  = rgb(0xb09a76)
	bcnStoneL  = rgb(0xf4e6c4)
	bcnBrick   = rgb(0xb0503a)
	bcnBrickD  = rgb(0x7e3426)
	bcnBrickL  = rgb(0xd07050)
	bcnRoofRed = rgb(0xb8583c)
	bcnWin     = rgb(0x2e3a52)
	bcnIron    = rgb(0x2a2c34)
)

// bcnTowerFlags: where the two small flags stand on the balcony of the city hall (flagmini props, barcelona.go).
var bcnTowerFlags = []Vec{{332, 104}, {360, 104}}

func barcelonaBackdrop() *pixBuf {
	if bcnBackBuf != nil {
		return bcnBackBuf
	}
	W, H := 960, int(bcnBackH)
	b := newPix(W, H)
	// the sky, lighter towards the horizon, and the clouds
	for y := 0; y < H; y++ {
		b.rect(0, y, W, 1, mixc(bcnSkyTop, bcnSkyLow, math.Min(1, float64(y)/78)))
	}
	for i := 0; i < 14; i++ {
		x, y := float64(phHash(i, 11)%W), float64(8+phHash(i, 13)%40)
		w := 14 + float64(phHash(i, 17)%22)
		b.ellipse(x, y+2, w, 4, rgb(0xd8e8f6))
		b.ellipse(x-w/3, y, w/2, 5, rgb(0xf6faff))
		b.ellipse(x+w/4, y-1, w/2.4, 5, rgb(0xffffff))
	}
	// the sea on the horizon (right half) with sailing boats
	b.rect(560, 62, 400, 14, bcnSea)
	for i := 0; i < 30; i++ {
		x, y := 560+phHash(i, 19)%400, 63+phHash(i, 23)%12
		b.rect(x, y, 3+phHash(i, 29)%5, 1, bcnSeaL)
	}
	for _, x := range []int{640, 700, 790} {
		b.rect(x, 66, 7, 2, rgb(0xf4f2ec))
		for k := 0; k < 6; k++ {
			b.rect(x+3-k/2, 60+k, 1+k/2, 1, rgb(0xffffff))
		}
	}
	// the hotel shaped like a sail, on the shore (right)
	for y := 22; y < 76; y++ {
		t := float64(y-22) / 54
		x0 := 900 + int(12*(1-math.Sin(t*math.Pi*0.9)))
		x1 := 930
		for x := x0; x < x1; x++ {
			c := mixc(rgb(0x6aa0d8), rgb(0x2a4a7a), float64(x-x0)/float64(maxI(1, x1-x0)))
			if y%3 == 0 {
				c = shade(c, 0.85)
			}
			b.set(x, y, c)
		}
	}
	// the hill of the Tibidabo (left of the middle): woods, the church on top, the communications tower
	for x := 240; x < 600; x++ {
		fx := float64(x)
		top := 34 + 28*math.Pow((fx-390)/170, 2) + 2*math.Sin(fx/9)
		if top > 90 {
			continue
		}
		b.rect(x, int(top), 1, H, bcnHill)
		if phHash(x, 3)%3 == 0 {
			b.rect(x, int(top)+2+phHash(x, 5)%20, 2, 2, bcnHillD)
		}
	}
	for i := 0; i < 50; i++ { // trees on the slopes
		x := 250 + phHash(i, 31)%340
		top := 34 + 28*math.Pow((float64(x)-390)/170, 2)
		b.ellipse(float64(x), top+6+float64(phHash(i, 37)%18), 4, 3, bcnHillD)
	}
	// the church: a stone body with towers and the tall central spire
	b.rect(372, 22, 30, 14, bcnStone)
	b.rect(372, 22, 30, 2, bcnStoneL)
	for _, x := range []int{372, 397} {
		b.rect(x, 14, 5, 10, bcnStone)
		b.rect(x+1, 11, 3, 3, bcnStoneD)
	}
	b.rect(384, 6, 6, 16, bcnStone)
	b.rect(385, 1, 4, 5, bcnStoneD)
	b.rect(386, 0, 2, 2, rgb(0xd8b84a)) // the statue on top
	for x := 376; x < 400; x += 5 {
		b.rect(x, 27, 2, 4, bcnWin)
	}
	// the communications tower
	b.rect(462, 4, 2, 34, rgb(0x8a8e98))
	b.rect(457, 18, 12, 3, rgb(0xc8ccd4))
	b.rect(459, 21, 8, 2, rgb(0x6a6e78))
	b.rect(462, 0, 1, 4, rgb(0xe04040))
	// the roofs of the city: rows of small houses, ochre and cream, red roofs
	for row := 0; row < 3; row++ {
		for i := 0; i < 80; i++ {
			x := phHash(i, 41+row)%W - 10
			y := 70 + row*12 + phHash(i, 43+row)%6
			w := 12 + phHash(i, 47+row)%18
			c := []color.RGBA{bcnStone, rgb(0xd8b088), rgb(0xe8d8c0), rgb(0xc89a70)}[phHash(i, 53+row)%4]
			c = shade(c, 0.78+0.08*float64(row))
			b.rect(x, y+3, w, 40, c)
			b.rect(x-1, y, w+2, 3, shade(bcnRoofRed, 0.8+0.08*float64(row)))
			for wx := x + 2; wx < x+w-2; wx += 4 {
				b.rect(wx, y+6, 2, 2, shade(bcnWin, 1.2))
			}
		}
	}
	// trees in the streets between the front buildings
	for _, x := range []int{124, 264, 426, 597, 756, 882} {
		b.ellipse(float64(x), 138, 8, 7, rgb(0x3a6a34))
		b.ellipse(float64(x)-2, 135, 5, 4, rgb(0x5a8a48))
	}
	bcnModernist(b)
	bcnEixample(b, 140, 252, 92)
	bcnEixample(b, 888, 960, 86)
	bcnCityHall(b)
	bcnSagrada(b)
	bcnMarket(b)
	bcnArc(b)
	// the pavement line where the backdrop meets the square
	b.rect(0, H-2, W, 2, rgb(0x8e8678))
	bcnBackBuf = b
	return b
}

// bcnEixample: a cream building of the grid, from x0 to x1, its roof at y top: rows of balconies with iron rails.
func bcnEixample(b *pixBuf, x0, x1, top int) {
	H := int(bcnBackH)
	b.rect(x0, top, x1-x0, H-top, bcnStoneL)
	b.rect(x0, top, x1-x0, 3, bcnStoneD)
	b.rect(x0, top-3, x1-x0, 3, bcnRoofRed)
	for y := top + 8; y < H-14; y += 12 {
		for x := x0 + 4; x+6 < x1; x += 11 {
			b.rect(x, y, 6, 8, bcnWin)
			b.rect(x-1, y+7, 8, 1, bcnIron)
			for k := 0; k < 8; k += 2 {
				b.set(x-1+k, y+6, bcnIron)
			}
		}
	}
	b.rect(x0, H-14, x1-x0, 12, rgb(0xc8b494)) // the shops on the ground floor
	for x := x0 + 4; x+10 < x1; x += 16 {
		b.rect(x, H-12, 10, 10, rgb(0x3a3028))
	}
}

// bcnModernist: the wavy house covered with mosaic (left).
func bcnModernist(b *pixBuf) {
	H := int(bcnBackH)
	x0, x1 := 0, 122
	for x := x0; x < x1; x++ {
		top := 74 + int(4*math.Sin(float64(x)/7))
		for y := top; y < H; y++ {
			c := bcnTile(rgb([]uint32{0xd8e0e8, 0xa8c8d8, 0xe8d0b8, 0xc0d8c0}[phHash(x/3, y/3)%4]), x, y)
			b.set(x, y, c)
		}
		// the scaly roof: a dragon's back, blue-green and pink scales
		for y := top - 10; y < top; y++ {
			c := rgb(0x4a8a8a)
			if (x/3+y/2)%2 == 0 {
				c = rgb(0x6aaa9a)
			}
			if (x/6)%5 == 0 {
				c = rgb(0xc87a8a)
			}
			b.set(x, y, c)
		}
	}
	b.rect(104, 52, 4, 14, bcnStoneL) // the little tower with its cross
	b.ellipse(106, 52, 4, 3, rgb(0xd8a040))
	b.rect(105, 42, 2, 8, rgb(0xe8e0d0))
	b.rect(102, 45, 8, 2, rgb(0xe8e0d0))
	// the windows with their bone-like frames and the balconies like masks
	for y := 88; y < H-16; y += 14 {
		for x := 8; x+10 < x1; x += 18 {
			wv := int(2 * math.Sin(float64(x+y)/6))
			b.ellipse(float64(x+5), float64(y+5+wv), 6, 5, rgb(0xe8dcc4))
			b.rect(x+1, y+2+wv, 8, 7, bcnWin)
			b.rect(x, y+9+wv, 10, 2, bcnIron)
			b.set(x+3, y+8+wv, bcnIron)
			b.set(x+6, y+8+wv, bcnIron)
		}
	}
	for x := 4; x < x1-4; x += 10 { // the bone columns of the ground floor
		b.rect(x, H-16, 3, 14, rgb(0xd8ccb0))
	}
}

// bcnCityHall: the stone city hall with its clock tower and the flag balcony.
func bcnCityHall(b *pixBuf) {
	H := int(bcnBackH)
	x0, x1 := 270, 420
	b.rect(x0, 78, x1-x0, H-78, bcnStone)
	b.rect(x0, 78, x1-x0, 3, bcnStoneL)
	b.rect(x0, 74, x1-x0, 4, bcnStoneD) // the cornice
	for x := x0 + 4; x < x1; x += 8 {   // the balustrade on the roof
		b.rect(x, 70, 3, 4, bcnStoneL)
	}
	b.rect(x0, 70, x1-x0, 1, bcnStoneL)
	for y := 86; y < H-18; y += 16 {
		for x := x0 + 6; x+8 < x1; x += 13 {
			b.rect(x, y, 7, 11, bcnWin)
			b.rect(x-1, y-2, 9, 2, bcnStoneL)
		}
	}
	// the columns of the porch and the door
	for x := 314; x <= 378; x += 16 {
		b.rect(x, 108, 5, H-108, bcnStoneL)
		b.rect(x+4, 108, 1, H-108, bcnStoneD)
	}
	b.rect(330, 120, 30, H-120, rgb(0x3a2a20))
	// the balcony of the flags over the porch
	b.rect(316, 104, 64, 3, bcnStoneD)
	for x := 318; x < 378; x += 3 {
		b.set(x, 103, bcnIron)
	}
	// the clock tower
	tx := 345
	b.rect(tx-14, 32, 28, 42, bcnStone)
	b.rect(tx-14, 32, 3, 42, bcnStoneL)
	b.rect(tx+11, 32, 3, 42, bcnStoneD)
	b.ellipse(float64(tx), 44, 8, 8, rgb(0xf4f0e4))
	b.ellipse(float64(tx), 44, 7, 7, rgb(0xfaf8f0))
	b.rect(tx, 38, 1, 6, bcnIron) // the hands
	b.rect(tx, 44, 5, 1, bcnIron)
	b.rect(tx-7, 56, 4, 10, bcnWin)
	b.rect(tx+3, 56, 4, 10, bcnWin)
	b.rect(tx-16, 28, 32, 4, bcnStoneD)
	for y := 12; y < 28; y++ { // the dome
		hw := int(14 * math.Sqrt(math.Max(0, 1-math.Pow(float64(28-y)/16, 2))))
		b.rect(tx-hw, y, 2*hw, 1, mixc(rgb(0x6a7a8a), rgb(0x3a4a5a), float64(y-12)/16))
	}
	b.rect(tx-1, 4, 2, 8, bcnIron)
}

// bcnSagrada: the basilica with its spires, the pinnacles and the crane.
func bcnSagrada(b *pixBuf) {
	H := int(bcnBackH)
	stone, stoneD, stoneL := rgb(0xd0b48a), rgb(0x9a7e5a), rgb(0xe8d0a6)
	// the body of the church
	b.rect(602, 92, 126, H-92, stone)
	for x := 602; x < 728; x += 6 {
		b.rect(x, 92, 1, H-92, stoneD)
	}
	// the portal and the rose of the facade
	b.rect(650, 112, 30, H-112, stoneD)
	b.ellipse(665, 112, 15, 10, stoneD)
	b.ellipse(665, 112, 10, 7, rgb(0x3a2a24))
	b.rect(656, 116, 18, H-116, rgb(0x3a2a24))
	// the spires: tall, round-topped cones full of holes, with a coloured pinnacle
	spire := func(cx, top, w int) {
		for y := top; y < 100; y++ {
			t := float64(y-top) / float64(100-top)
			hw := int(math.Round(float64(w) * (0.35 + 0.65*math.Sqrt(t))))
			for x := cx - hw; x <= cx+hw; x++ {
				c := stone
				if x < cx-hw/2 {
					c = stoneL
				} else if x > cx+hw/2 {
					c = stoneD
				}
				if (y-top)%6 == 3 && x%3 == 0 && hw > 2 {
					c = rgb(0x4a3a30) // the holes of the spire
				}
				b.set(x, y, c)
			}
		}
		b.rect(cx-2, top-5, 5, 5, rgb(0xd84a3a)) // the pinnacle with its mosaic
		b.set(cx, top-6, rgb(0xf0c040))
		b.set(cx-1, top-3, rgb(0xf0f0f0))
	}
	spire(620, 30, 6)
	spire(638, 18, 7)
	spire(692, 18, 7)
	spire(710, 30, 6)
	spire(665, 6, 9) // the tallest, in the middle
	spire(652, 40, 5)
	spire(678, 40, 5)
	// the crane (yellow lattice) beside it
	crane := rgb(0xe8b828)
	b.rect(742, 4, 3, H-60, crane)
	for y := 6; y < H-60; y += 4 {
		b.set(741, y, rgb(0xa88018))
		b.set(745, y+2, rgb(0xa88018))
	}
	b.rect(696, 4, 72, 2, crane)
	for x := 698; x < 766; x += 4 {
		b.set(x, 6, rgb(0xa88018))
	}
	b.rect(762, 6, 6, 5, rgb(0x6a6e78)) // the counterweight
	b.rect(706, 6, 1, 20, bcnIron)      // the hook, over the tallest spire
	b.rect(704, 26, 5, 2, bcnIron)
}

// bcnMarket: the brick market hall with its big arch of stained glass.
func bcnMarket(b *pixBuf) {
	H := int(bcnBackH)
	x0, x1 := 432, 592
	cx := (x0 + x1) / 2
	// the gable roof
	for y := 82; y < 100; y++ {
		hw := (y - 82) * 5
		b.rect(cx-hw, y, 2*hw, 1, shade(rgb(0x6a5a5e), 0.9+float64(y-82)/60))
	}
	b.rect(x0, 98, x1-x0, H-98, bcnBrick)
	for y := 100; y < H; y += 4 {
		b.rect(x0, y, x1-x0, 1, bcnBrickD)
	}
	b.rect(x0, 96, x1-x0, 3, bcnBrickL)
	// the big arch with the stained glass (blue, yellow, red, green panes)
	for y := 104; y < H-2; y++ {
		for x := cx - 26; x <= cx+26; x++ {
			dx, dy := float64(x-cx)/26, float64(y-124)/20
			if y < 124 && dx*dx+dy*dy > 1 {
				continue
			}
			c := rgb([]uint32{0x3a6ab8, 0xe8c040, 0xc84a3a, 0x4a9a5a, 0x8ac0e0}[phHash(x/4, y/4)%5])
			if x%4 == 0 || y%4 == 0 {
				c = bcnIron
			}
			b.set(x, y, c)
		}
	}
	b.rect(cx-8, H-24, 16, 22, rgb(0x3a2a20)) // the door under the glass
	// the side arches
	for _, ax := range []int{x0 + 22, x1 - 22} {
		b.ellipse(float64(ax), 118, 10, 8, bcnBrickD)
		b.rect(ax-10, 118, 20, H-120, bcnBrickD)
		b.rect(ax-8, 118, 16, H-120, rgb(0x2e3a52))
		b.ellipse(float64(ax), 118, 8, 6, rgb(0x2e3a52))
	}
}

// bcnArc: the triumphal arch of red brick.
func bcnArc(b *pixBuf) {
	H := int(bcnBackH)
	x0, x1 := 768, 876
	cx := (x0 + x1) / 2
	b.rect(x0, 78, x1-x0, H-78, bcnBrick)
	for y := 80; y < H; y += 4 {
		b.rect(x0, y, x1-x0, 1, bcnBrickD)
	}
	b.rect(x0-2, 76, x1-x0+4, 4, bcnStoneL) // the frieze
	b.rect(x0, 84, x1-x0, 4, bcnStone)
	// the opening
	for y := 96; y < H; y++ {
		for x := cx - 22; x <= cx+22; x++ {
			dx, dy := float64(x-cx)/22, float64(y-118)/22
			if y < 118 && dx*dx+dy*dy > 1 {
				continue
			}
			b.set(x, y, mixc(bcnSkyLow, rgb(0x6a9a5a), float64(y-96)/54))
		}
	}
	// the corner towers with their domes and the crown on top
	for _, tx := range []int{x0 + 4, x1 - 4} {
		b.rect(tx-6, 62, 12, 16, bcnBrickL)
		b.ellipse(float64(tx), 60, 6, 5, rgb(0xd8b84a))
		b.rect(tx, 50, 1, 6, rgb(0xd8b84a))
	}
	b.rect(cx-14, 66, 28, 10, bcnBrickL)
	b.ellipse(float64(cx), 64, 10, 6, rgb(0xd8b84a))
}
