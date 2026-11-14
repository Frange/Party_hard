package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Props of BARCELONA (barcelona.go). All drawn by code in the palette of the game:
//   bcn_mosaic     the big round mosaic of broken tiles in the middle of the square (flat, under everybody): the dance
//                  floor, with the DJ at its back edge
//   bcn_fountain   a round fountain in a square flower bed with a hedge (animated: the jet, the spray and the ripples)
//   bcn_pavilion      the gingerbread pavilions of the park: rough stone walls, the white "icing" roofs covered with
//                  mosaic, the chequered tower (0) or the little cross turret (1)
//   bcn_bench      the long wavy bench covered with mosaic (guests sit on it)
//   bcn_kiosk      the blue newspaper kiosk ("PREMSA": the Catalan word for the press)
//   bcn_cablecar   the cable car of the backdrop: its cable and two red cabins going up and down (animated scenery)
// The flags (Spain, Catalonia) are flags.go; palms, trees, lamps, benches, café tables, bins and the punch are the
// usual ones. No building of the square can be entered: the landmarks of the backdrop are painted (barcelona_backdrop.go).
// ---------------------------------------------------------------------------

const (
	bcnMosaicW, bcnMosaicH = 200, 96
	bcnMosaicRX            = 96.0
	bcnMosaicRY            = 44.0
	bcnFountW, bcnFountH   = 132, 96
	bcnFountFrames         = 4
	bcnCableW, bcnCableH   = 140, 50
	bcnCableFrames         = 16
)

func init() {
	pk := propKinds
	pk["bcn_mosaic"] = &PropKind{W: bcnMosaicW, H: bcnMosaicH, FW: 4, FH: 3, NoSolid: true, SortOff: -600, Draw: drawBcnMosaic}
	pk["bcn_fountain"] = &PropKind{W: bcnFountW, H: bcnFountH, FW: 124, FH: 50, Draw: func(b *pixBuf, v int) { drawBcnFountain(b, 0) }}
	pk["bcn_pavilion"] = &PropKind{W: 92, H: 104, FW: 80, FH: 34, Vars: 2, Draw: drawBcnPavilion}
	pk["bcn_bench"] = &PropKind{W: 74, H: 24, FW: 68, FH: 10, UseKind: "lounge", Use: []Vec{{-22, 12}, {0, 12}, {22, 12}}, Draw: drawBcnBench}
	pk["bcn_kiosk"] = &PropKind{W: 50, H: 64, FW: 42, FH: 22, Hide: true, Draw: drawBcnKiosk}
	pk["bcn_cablecar"] = &PropKind{W: bcnCableW, H: bcnCableH, FW: 4, FH: 3, NoSolid: true, Draw: func(b *pixBuf, v int) { drawBcnCableCar(b, 0) }}
	for _, n := range []string{"bcn_mosaic", "bcn_cablecar"} {
		overlayKinds[n] = true
	}
	propAnims["bcn_fountain"] = &propAnim{n: bcnFountFrames, fps: 6, draw: func(b *pixBuf, k *PropKind, v, f int) { drawBcnFountain(b, f) }}
	propAnims["bcn_cablecar"] = &propAnim{n: bcnCableFrames, fps: 2, draw: func(b *pixBuf, k *PropKind, v, f int) { drawBcnCableCar(b, f) }}
}

// bcnTile: a broken-tile (trencadís) speckle: the base colour, lighter or darker by a hash of the pixel, with the white
// grout between the pieces.
func bcnTile(c color.RGBA, x, y int) color.RGBA {
	h := phHash(x*31+y*17, x^y)
	switch {
	case (x+y*3)%7 == 0 || (x*5+y)%11 == 0:
		return rgb(0xf0ece2) // grout
	case h%5 == 0:
		return shade(c, 1.18)
	case h%5 == 1:
		return shade(c, 0.84)
	}
	return c
}

// ---- the mosaic of the square ------------------------------------------------------------------------------------

func drawBcnMosaic(b *pixBuf, v int) {
	cx, cy := float64(bcnMosaicW)/2, float64(bcnMosaicH)/2
	rx, ry := bcnMosaicRX, bcnMosaicRY
	terra, cream, slate, rose := rgb(0xb8644a), rgb(0xe6d6b4), rgb(0x56668a), rgb(0xc88a7a)
	for y := 0; y < bcnMosaicH; y++ {
		for x := 0; x < bcnMosaicW; x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			r := math.Hypot(dx, dy)
			if r > 1 {
				continue
			}
			var c color.RGBA
			a := math.Atan2(dy, dx)
			switch {
			case r > 0.9:
				c = terra
			case r > 0.84:
				c = cream
			case r > 0.36:
				// eight rays, slate and rose, on a cream ground
				s := int(math.Floor((a + math.Pi) / (math.Pi / 4)))
				m := math.Abs(math.Remainder(a, math.Pi/4)) / (math.Pi / 8) // 0 on a ray, 1 between two
				switch {
				case m < 0.5-0.3*(r-0.36):
					c = slate
					if s%2 == 1 {
						c = rose
					}
				default:
					c = cream
				}
			case r > 0.3:
				c = terra
			default:
				// the star in the middle: a diamond on cream
				if math.Abs(dx)+math.Abs(dy) < 0.26 {
					c = slate
				} else {
					c = rose
				}
			}
			b.set(x, y, bcnTile(c, x, y))
		}
	}
}

// ---- the fountains ---------------------------------------------------------------------------------------------

func drawBcnFountain(b *pixBuf, f int) {
	W, H := bcnFountW, bcnFountH
	top := H - 50 // the bed's back edge (the footprint)
	hedge, hedgeD, hedgeL := rgb(0x2e6a34), rgb(0x1e4a24), rgb(0x4a8a44)
	soil := rgb(0x6a4a30)
	stone, stoneD, stoneL := rgb(0xd8ccb4), rgb(0x9a8e78), rgb(0xf0e6d0)
	water, waterD, waterL := rgb(0x3a8ad0), rgb(0x24609e), rgb(0x9ad0f4)
	b.ellipse(float64(W)/2, float64(H)-2, float64(W)/2, 4, color.RGBA{0, 0, 0, 50})
	// the bed: soil with flowers, the clipped hedge all round
	b.rect(4, top, W-8, H-top-2, soil)
	for i := 0; i < 90; i++ {
		x, y := 8+phHash(i, 3)%(W-16), top+4+phHash(i, 7)%(H-top-10)
		c := rgb([]uint32{0xe04a5a, 0xf0c040, 0xf2f0ea, 0xd070c0, 0x5aa04a}[i%5])
		b.set(x, y, c)
		b.set(x+1, y, shade(c, 0.8))
	}
	for x := 4; x < W-4; x++ {
		for _, y0 := range []int{top - 3, H - 8} {
			b.rect(x, y0, 1, 6, hedge)
			b.set(x, y0, hedgeL)
			if (x+y0)%3 == 0 {
				b.set(x, y0+2, hedgeL)
			}
			b.set(x, y0+5, hedgeD)
		}
	}
	for y := top - 3; y < H-2; y++ {
		for _, x0 := range []int{4, W - 10} {
			b.rect(x0, y, 6, 1, hedge)
			if (y+x0)%3 == 0 {
				b.set(x0+2, y, hedgeL)
			}
		}
	}
	b.rect(W/2-8, H-8, 16, 6, rgb(0xb0a48e)) // the gap in front: a path to the basin
	// the round basin
	cx, cy := float64(W)/2, float64(top)+24
	b.ellipse(cx, cy+3, 38, 15, stoneD)
	b.ellipse(cx, cy, 38, 15, stone)
	b.ellipse(cx, cy-1, 36, 13, stoneL)
	b.ellipse(cx, cy, 33, 11, water)
	for y := int(cy - 11); y <= int(cy+11); y++ {
		for x := int(cx - 33); x <= int(cx+33); x++ {
			dx, dy := (float64(x)-cx)/33, (float64(y)-cy)/11
			r := math.Hypot(dx, dy)
			if r > 1 {
				continue
			}
			// rings of ripples running outwards
			if int(r*10+float64(f)*0.75)%3 == 0 {
				b.set(x, y, waterL)
			} else if y < int(cy-6) {
				b.set(x, y, waterD)
			}
		}
	}
	// the column and its bowl
	b.rect(int(cx)-3, int(cy)-16, 6, 16, stone)
	b.rect(int(cx)-3, int(cy)-16, 2, 16, stoneL)
	b.ellipse(cx, cy-17, 11, 3, stoneD)
	b.ellipse(cx, cy-18, 10, 2.5, water)
	// the jet: a white column that grows and falls back, and the drops
	jh := 26 + []int{0, 3, 5, 2}[f%4]
	for y := int(cy) - 18 - jh; y < int(cy)-18; y++ {
		b.set(int(cx), y, rgb(0xf4fbff))
		b.set(int(cx)-1, y, waterL)
		b.set(int(cx)+1, y, waterL)
	}
	for i := 0; i < 10; i++ {
		t := math.Mod(float64(i)/10+float64(f)/bcnFountFrames, 1)
		for _, s := range []float64{-1, 1} {
			x := cx + s*(4+t*22)
			y := cy - 18 - float64(jh) + 4 + t*t*float64(jh+10)
			b.set(int(x), int(y), waterL)
		}
	}
}

// ---- the pavilions of the park ---------------------------------------------------------------------------------

func drawBcnPavilion(b *pixBuf, v int) {
	W, H := 92, 104
	wall, wallD, wallL := rgb(0xc89a66), rgb(0x8e6a46), rgb(0xe0b88a)
	icing, icingD := rgb(0xf2ece0), rgb(0xc8beb0)
	brown := rgb(0x6a4228)
	b.ellipse(float64(W)/2, float64(H)-2, 44, 4, color.RGBA{0, 0, 0, 60})
	// the walls: rough stone in blocks
	x0, x1, y0 := 8, W-8, 50
	b.rect(x0, y0, x1-x0, H-y0-2, wall)
	for y := y0; y < H-2; y++ {
		for x := x0; x < x1; x++ {
			h := phHash(x/5+y/4*13, 5)
			if (y-y0)%5 == 0 || (x+(y/5)*3)%9 == 0 {
				b.set(x, y, wallD)
			} else if h%4 == 0 {
				b.set(x, y, wallL)
			}
		}
	}
	b.rect(x1-6, y0, 6, H-y0-2, shade(wall, 0.82)) // the side in shade
	// the door and windows, framed with white tiles
	b.rect(W/2-8, H-26, 16, 24, brown)
	b.rect(W/2-8, H-26, 16, 2, icing)
	b.ellipse(float64(W)/2, float64(H)-26, 8, 4, brown)
	for _, wx := range []int{16, W - 26} {
		b.rect(wx-1, y0+12, 12, 14, icing)
		b.rect(wx, y0+13, 10, 12, rgb(0x2a3a4a))
		b.rect(wx+4, y0+13, 2, 12, icing)
	}
	// the white roof: a wavy icing that drips down the walls
	for x := x0 - 4; x < x1+4; x++ {
		drip := 4 + int(3*math.Abs(math.Sin(float64(x)/5)))
		b.rect(x, y0-10, 1, 10+drip, icing)
		b.set(x, y0+drip, icingD)
	}
	// the mosaic on the roof (blue, white, ochre broken tiles)
	for y := y0 - 18; y < y0-6; y++ {
		hw := 34 - (y0-6-y)*2
		for x := W/2 - hw; x < W/2+hw; x++ {
			c := rgb([]uint32{0x4a7ac8, 0xf2ece0, 0xd8a040, 0x5ab0b8}[phHash(x/2, y/2)%4])
			b.set(x, y, bcnTile(c, x, y))
		}
	}
	if v == 0 {
		// the chequered tower (blue and white) with its mushroom top
		tx := W - 24
		for y := 8; y < y0-6; y++ {
			for x := tx - 6; x < tx+6; x++ {
				c := rgb(0x3a68b8)
				if (x/3+y/3)%2 == 0 {
					c = rgb(0xf2ece0)
				}
				b.set(x, y, c)
			}
		}
		b.ellipse(float64(tx), 8, 10, 6, icing)
		b.ellipse(float64(tx), 6, 9, 4, rgb(0xc84a3a))
		for i := 0; i < 6; i++ {
			b.set(tx-6+i*2, 6, icing)
		}
		return
	}
	// the small cross turret
	tx := W/2 + 10
	b.rect(tx-5, 18, 10, y0-24, icing)
	for y := 18; y < y0-6; y += 3 {
		b.rect(tx-5, y, 10, 1, rgb(0x5ab0b8))
	}
	b.ellipse(float64(tx), 18, 7, 4, rgb(0xd8a040))
	b.rect(tx-1, 4, 3, 12, rgb(0xe4ddd0))
	b.rect(tx-5, 8, 11, 3, rgb(0xe4ddd0))
}

// ---- the mosaic bench ------------------------------------------------------------------------------------------

func drawBcnBench(b *pixBuf, v int) {
	W, H := 74, 24
	b.ellipse(float64(W)/2, float64(H)-2, 36, 3, color.RGBA{0, 0, 0, 60})
	for x := 2; x < W-2; x++ {
		wave := int(math.Round(2 * math.Sin(float64(x)/9)))
		top := 4 + wave
		for y := top; y < H-3; y++ {
			c := rgb([]uint32{0x4a8ac8, 0xf0ece2, 0xd8a040, 0x5ab06a, 0xc85a4a}[phHash(x/3, y/3)%5])
			if y < top+4 {
				c = rgb(0xf0ece2) // the rounded back
				if y == top {
					c = rgb(0xffffff)
				}
			}
			b.set(x, y, bcnTile(c, x, y))
		}
		b.set(x, H-3, rgb(0x8a7a66))
	}
}

// ---- the newspaper kiosk ---------------------------------------------------------------------------------------

func drawBcnKiosk(b *pixBuf, v int) {
	const W, H = 50, 64
	blue, blueD, blueL := rgb(0x2a4a8a), rgb(0x1a2e5e), rgb(0x5a7ac0)
	b.ellipse(W/2, H-2, 24, 3, color.RGBA{0, 0, 0, 60})
	b.cube(4, 28, W-8, H-30, 10, blueD, blue)
	b.rect(W-9, 28, 5, H-30, blueD)
	for r := 0; r < 2; r++ {
		for i := 0; i < 5; i++ {
			x, y := 7+i*7, 40+r*10
			c := rgb([]uint32{0xf0e8d0, 0xe05a4a, 0xf0c040, 0x5ab0d8, 0xd070b0}[(i+r*3)%5])
			b.rect(x, y, 6, 8, c)
			b.rect(x+1, y+1, 4, 2, shade(c, 0.6))
			b.rect(x+1, y+4, 3, 1, shade(c, 0.75))
		}
	}
	// the octagonal roof: a dark blue hood with a light rim, the sign on top and the little lantern
	for y := 14; y < 26; y++ {
		hw := 12 + (y - 14)
		b.rect(W/2-hw, y, 2*hw, 1, mixc(blueL, blue, float64(y-14)/12))
	}
	b.rect(1, 26, W-2, 2, blueD)
	b.rect(7, 4, W-14, 10, rgb(0xf2ecd8))
	b.rect(7, 4, W-14, 1, blueL)
	gdlTextC(b, "PREMSA", W/2, 7, blueD)
	b.rect(W/2-1, 0, 3, 4, rgb(0xf0c040))
}

// ---- the cable car of the backdrop ------------------------------------------------------------------------------

// bcnCablePoint: the point of the cable at t (0 = the upper station on the hill, left; 1 = the lower end, right).
func bcnCablePoint(t float64) (float64, float64) {
	x := 6 + t*(bcnCableW-12)
	y := 8 + t*30 + 6*math.Sin(t*math.Pi) // it sags a little
	return x, y
}

func drawBcnCableCar(b *pixBuf, f int) {
	cable := rgb(0x2a2a30)
	for i := 0; i <= 400; i++ {
		x, y := bcnCablePoint(float64(i) / 400)
		b.set(int(x), int(y), cable)
		b.set(int(x), int(y)+3, shade(cable, 1.4)) // the second cable, a little lower
	}
	// the pylons at both ends
	for _, t := range []float64{0, 1} {
		x, y := bcnCablePoint(t)
		b.rect(int(x)-1, int(y), 3, bcnCableH-int(y), rgb(0x5a5e68))
		b.rect(int(x)-3, int(y)-1, 7, 2, rgb(0x5a5e68))
	}
	// two cabins: one goes down while the other comes up
	p := float64(f) / bcnCableFrames
	for i, t := range []float64{0.1 + 0.8*p, 0.9 - 0.8*p} {
		x, y := bcnCablePoint(t)
		cx, cy := int(x), int(y)
		b.rect(cx, cy, 1, 4, cable)
		b.rect(cx-4, cy+4, 9, 7, rgb(0xc8282a))
		b.rect(cx-4, cy+4, 9, 1, rgb(0xe85a4a))
		b.rect(cx-3, cy+6, 3, 2, rgb(0xd8ecf8))
		b.rect(cx+1, cy+6, 3, 2, rgb(0xd8ecf8))
		b.rect(cx-4, cy+10, 9, 1, rgb(0x7a1418))
		if i == 1 {
			b.rect(cx-4, cy+9, 9, 1, rgb(0xf0ece2)) // the second cabin has a white band
		}
	}
}
